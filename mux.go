package Artifex

import (
	"fmt"
	"sync"

	"golang.org/x/exp/constraints"
)

type MessageHandler[Message any] func(message Message) error

func (handler MessageHandler[Message]) PreMiddleware() MessageDecorator[Message] {
	return func(next MessageHandler[Message]) MessageHandler[Message] {
		return func(message Message) error {
			err := handler(message)
			if err != nil {
				return err
			}
			return next(message)
		}
	}
}

func (handler MessageHandler[Message]) PostMiddleware() MessageDecorator[Message] {
	return func(next MessageHandler[Message]) MessageHandler[Message] {
		return func(message Message) error {
			err := next(message)
			if err != nil {
				return err
			}
			return handler(message)
		}
	}
}

func (handler MessageHandler[Message]) LinkMiddlewares(middlewares ...MessageDecorator[Message]) MessageHandler[Message] {
	return LinkMiddlewares(handler, middlewares...)
}

type MessageDecorator[Message any] func(next MessageHandler[Message]) MessageHandler[Message]

func LinkMiddlewares[Message any](handler MessageHandler[Message], middlewares ...MessageDecorator[Message]) MessageHandler[Message] {
	n := len(middlewares)
	for i := n - 1; 0 <= i; i-- {
		decorator := middlewares[i]
		handler = decorator(handler)
	}
	return handler
}

//

type SubjectFactory[Subject constraints.Ordered, Message any] func(Message) (Subject, error)

func NewMessageMux[Subject constraints.Ordered, Message any](getSubject SubjectFactory[Subject, Message], logger Logger) *MessageMux[Subject, Message] {
	return &MessageMux[Subject, Message]{
		logger:      logger,
		getSubject:  getSubject,
		handlers:    make(map[Subject]MessageHandler[Message]),
		middlewares: make([]MessageDecorator[Message], 0),
	}
}

// MessageMux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a MessageHandler, but with added routing capabilities.
//
// Message represents a high-level abstraction data structure containing metadata (e.g. header) + body
type MessageMux[Subject constraints.Ordered, Message any] struct {
	mu     sync.RWMutex
	logger Logger

	// getSubject 是為了避免 generic type 呼叫 method 所造成的效能降低
	// 同時可以因應不同情境, 改變取得 subject 的規則
	//
	// https://www.youtube.com/watch?v=D1hI55EcBB4&t=20260s
	//
	// https://hackmd.io/@fieliapm/BkHvJjYq3#/5/2
	getSubject  SubjectFactory[Subject, Message]
	handlers    map[Subject]MessageHandler[Message]
	middlewares []MessageDecorator[Message]

	notFoundHandler MessageHandler[Message]
	defaultHandler  MessageHandler[Message]
}

func (mux *MessageMux[Subject, Message]) handle(message Message) (err error) {
	logger := mux.logger

	subject, err := mux.getSubject(message)
	if err != nil {
		Err := ErrorWrapWithMessage(err, "Failed to parse subject from the message")
		logger.Error("%v", Err)
		return Err
	}
	logger.Info("receive subject=%v", subject)

	defer func() {
		if err != nil {
			logger.Error("hande subject=%v fail", subject)
			return
		}
		logger.Info("hande subject=%v success", subject)
	}()

	fn, ok := mux.handlers[subject]
	if !ok {
		if mux.defaultHandler != nil {
			return mux.defaultHandler.LinkMiddlewares(mux.middlewares...)(message)
		}

		if mux.notFoundHandler == nil {
			return ErrNotFound
		}

		return mux.notFoundHandler(message)
	}
	return fn.LinkMiddlewares(mux.middlewares...)(message)
}

func (mux *MessageMux[Subject, Message]) Subjects() (result []Subject) {
	for subject := range mux.handlers {
		result = append(result, subject)
	}
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

func (mux *MessageMux[Subject, Message]) RegisterHandler(subject Subject, fn MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	_, ok := mux.handlers[subject]
	if ok {
		panic(fmt.Sprintf("mux have duplicate subject=%v", subject))
	}

	mux.handlers[subject] = fn
	return mux
}

func (mux *MessageMux[Subject, Message]) ReRegisterHandler(subject Subject, fn MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	mux.handlers[subject] = fn
	return mux
}

func (mux *MessageMux[Subject, Message]) RemoveHandler(subject Subject) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	delete(mux.handlers, subject)
}

func (mux *MessageMux[Subject, Message]) AddMiddleware(middlewares ...MessageDecorator[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.middlewares = append(mux.middlewares, middlewares...)
	return mux
}

func (mux *MessageMux[Subject, Message]) AddPreMiddleware(handlers ...MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	for _, handler := range handlers {
		mux.middlewares = append(mux.middlewares, handler.PreMiddleware())
	}
	return mux
}

func (mux *MessageMux[Subject, Message]) AddPostMiddleware(handlers ...MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	for _, handler := range handlers {
		mux.middlewares = append(mux.middlewares, handler.PostMiddleware())
	}
	return mux
}

func (mux *MessageMux[Subject, Message]) SetNotFoundHandler(fn MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.notFoundHandler = fn
	return mux
}

func (mux *MessageMux[Subject, Message]) SetDefaultHandler(fn MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.defaultHandler = fn
	return mux
}
