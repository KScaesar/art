package Artifex

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"golang.org/x/exp/constraints"
)

type MessageHandler[Message any] interface {
	HandleMessage(message Message) error
}

type MessageHandleFunc[Message any] func(message Message) error

func (fn MessageHandleFunc[Message]) HandleMessage(message Message) error {
	return fn(message)
}

func (fn MessageHandleFunc[Message]) PreMiddleware() MessageDecorator[Message] {
	return func(next MessageHandleFunc[Message]) MessageHandleFunc[Message] {
		return func(message Message) error {
			err := fn(message)
			if err != nil {
				return err
			}
			return next(message)
		}
	}
}

func (fn MessageHandleFunc[Message]) PostMiddleware() MessageDecorator[Message] {
	return func(next MessageHandleFunc[Message]) MessageHandleFunc[Message] {
		return func(message Message) error {
			err := next(message)
			if err != nil {
				return err
			}
			return fn(message)
		}
	}
}

func (fn MessageHandleFunc[Message]) LinkMiddlewares(middlewares ...MessageDecorator[Message]) MessageHandleFunc[Message] {
	return LinkMiddlewares(fn, middlewares...)
}

type MessageDecorator[Message any] func(next MessageHandleFunc[Message]) MessageHandleFunc[Message]

func LinkMiddlewares[Message any](handler MessageHandleFunc[Message], middlewares ...MessageDecorator[Message]) MessageHandleFunc[Message] {
	n := len(middlewares)
	for i := n - 1; 0 <= i; i-- {
		decorator := middlewares[i]
		handler = decorator(handler)
	}
	return handler
}

//

type SubjectFactory[Message any] func(Message) (string, error)

func NewMessageMux[Subject constraints.Ordered, Message any](getSubject SubjectFactory[Message], logger Logger) *MessageMux[Subject, Message] {
	return &MessageMux[Subject, Message]{
		logger:         logger,
		isRoot:         true,
		groupDelimiter: "/",
		getSubject:     getSubject,
		handlers:       make(map[string]MessageHandler[Message]),
		middlewares:    make([]MessageDecorator[Message], 0),
	}
}

// MessageMux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a MessageHandler, but with added routing capabilities.
//
// Message represents a high-level abstraction data structure containing metadata (e.g. header) + body
type MessageMux[Subject constraints.Ordered, Message any] struct {
	mu     sync.RWMutex
	logger Logger

	isRoot         bool
	groupName      string
	groupDelimiter string

	// getSubject 是為了避免 generic type 呼叫 method 所造成的效能降低
	// 同時可以因應不同情境, 改變取得 subject 的規則
	//
	// https://www.youtube.com/watch?v=D1hI55EcBB4&t=20260s
	//
	// https://hackmd.io/@fieliapm/BkHvJjYq3#/5/2
	getSubject  SubjectFactory[Message]
	handlers    map[string]MessageHandler[Message]
	middlewares []MessageDecorator[Message]

	notFoundHandler MessageHandleFunc[Message]
	defaultHandler  MessageHandleFunc[Message]
}

func (mux *MessageMux[Subject, Message]) handle(message Message) (err error) {
	logger := mux.logger

	subject, err := mux.getSubject(message)
	if err != nil {
		Err := ErrorWrapWithMessage(err, "Failed to parse subject from the message")
		if mux.isRoot {
			logger.Error("%v", Err)
		}
		return Err
	}
	if mux.isRoot {
		logger.Info("receive subject=%v", subject)
	}

	defer func() {
		if err != nil {
			if mux.isRoot {
				logger.Error("hande subject=%v fail", subject)
			}
			return
		}
		if mux.isRoot {
			logger.Info("hande subject=%v success", subject)
		}
	}()

	handler, ok := mux.handlers[subject]
	if ok {
		return LinkMiddlewares(handler.HandleMessage, mux.middlewares...)(message)
	}

	if mux.defaultHandler != nil {
		return LinkMiddlewares(mux.defaultHandler, mux.middlewares...)(message)
	}

	if mux.notFoundHandler != nil {
		return mux.notFoundHandler(message)
	}

	return ErrNotFound
}

func (mux *MessageMux[Subject, Message]) Subjects() (result []string) {
	for subject, handler := range mux.handlers {
		m, ok := handler.(*MessageMux[Subject, Message])
		if ok {
			result = append(result, m.Subjects()...)
			continue
		}
		result = append(result, mux.groupName+subject)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return
}

func (mux *MessageMux[Subject, Message]) HandleMessageWithoutMutex(message Message) error {
	return mux.handle(message)
}

func (mux *MessageMux[Subject, Message]) HandleMessage(message Message) error {
	mux.mu.RLock()
	defer mux.mu.RUnlock()
	return mux.handle(message)
}

func (mux *MessageMux[Subject, Message]) RegisterHandler(s Subject, h MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	subject := CleanSubject(s) + mux.groupDelimiter
	_, ok := mux.handlers[subject]
	if ok {
		panic(fmt.Sprintf("mux have duplicate subject=%v", subject))
	}

	mux.handlers[subject] = h
	return mux
}

func (mux *MessageMux[Subject, Message]) AddMiddleware(middlewares ...MessageDecorator[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.middlewares = append(mux.middlewares, middlewares...)
	return mux
}

func (mux *MessageMux[Subject, Message]) AddPreMiddleware(handlers ...MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	for _, handler := range handlers {
		mux.middlewares = append(mux.middlewares, handler.PreMiddleware())
	}
	return mux
}

func (mux *MessageMux[Subject, Message]) AddPostMiddleware(handlers ...MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	for _, handler := range handlers {
		mux.middlewares = append(mux.middlewares, handler.PostMiddleware())
	}
	return mux
}

func (mux *MessageMux[Subject, Message]) SetNotFoundHandler(h MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.notFoundHandler = h
	return mux
}

func (mux *MessageMux[Subject, Message]) SetDefaultHandler(h MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.defaultHandler = h
	return mux
}

func (mux *MessageMux[Subject, Message]) SetGroupDelimiter(delimiter string) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.groupDelimiter = delimiter
}

func (mux *MessageMux[Subject, Message]) Group(s Subject) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	childName := CleanSubject(s) + mux.groupDelimiter
	groupName := mux.groupName + childName

	_, ok := mux.handlers[childName]
	if ok {
		panic(fmt.Sprintf("mux have duplicate group=%v", groupName))
	}

	groupMux := NewMessageMux[Subject, Message](mux.getSubject, mux.logger)
	groupMux.isRoot = false
	groupMux.groupName = groupName
	groupMux.groupDelimiter = mux.groupDelimiter
	// groupMux.defaultHandler = func(message Message) error { return nil }

	mux.handlers[childName] = groupMux
	return groupMux
}

func CleanSubject[Subject constraints.Ordered](s Subject) string {
	subject := fmt.Sprintf("%v", s)
	return strings.Trim(strings.TrimSpace(subject), `/.\`)
}
