package Artifex

import (
	"fmt"
	"sort"
	"strings"

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

type NewSubjectFunc[Message any] func(Message) (string, error)

func NewMessageMux[Subject constraints.Ordered, Message any](newSubject NewSubjectFunc[Message]) *MessageMux[Subject, Message] {
	return &MessageMux[Subject, Message]{
		logger:         DefaultLogger(),
		groupDelimiter: "/",
		newSubject:     newSubject,
		handlers:       make(map[string]MessageHandler[Message]),
		middlewares:    make([]MessageDecorator[Message], 0),
	}
}

// MessageMux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a MessageHandler, but with added routing capabilities.
//
// rMessage represents a high-level abstraction data structure containing metadata (e.g. header) + body
type MessageMux[Subject constraints.Ordered, Message any] struct {
	logger Logger

	parentGroupName string
	groupDelimiter  string
	transform       func(old Message) (fresh Message, err error)

	// newSubject 是為了避免 generic type 呼叫 method 所造成的效能降低
	// 同時可以因應不同情境, 改變取得 subject 的規則
	//
	// https://www.youtube.com/watch?v=D1hI55EcBB4&t=20260s
	//
	// https://hackmd.io/@fieliapm/BkHvJjYq3#/5/2
	newSubject  NewSubjectFunc[Message]
	handlers    map[string]MessageHandler[Message]
	middlewares []MessageDecorator[Message]

	notFoundHandler MessageHandleFunc[Message]
	defaultHandler  MessageHandleFunc[Message]
}

func (mux *MessageMux[Subject, Message]) SetLogger(logger Logger) {
	mux.logger = logger
}

func (mux *MessageMux[Subject, Message]) Transform(transform func(old Message) (fresh Message, err error)) *MessageMux[Subject, Message] {
	if mux.transform != nil {
		panic("mux transform has set")
	}
	mux.transform = transform
	return mux
}

func (mux *MessageMux[Subject, Message]) HandleMessage(message Message) (err error) {
	if mux.transform != nil {
		message, err = mux.transform(message)
		if err != nil {
			Err := ErrorWrapWithMessage(err, "Failed to transform")
			mux.logger.Error("%v", Err)
			return err
		}
	}

	subject, err := mux.newSubject(message)
	if err != nil {
		Err := ErrorWrapWithMessage(err, "Failed to parse subject from the message")
		mux.logger.Error("%v", Err)
		return Err
	}
	mux.logger.Debug("handle subject=%v", subject)

	err = handleMessage(mux, message, subject)
	if err != nil {
		mux.logger.Error("hande subject=%v fail: %v", subject, err)
		return err
	}
	mux.logger.Debug("hande subject=%v success", subject)
	return nil
}

func handleMessage[Subject constraints.Ordered, Message any](mux *MessageMux[Subject, Message], message Message, subject string) (err error) {
	cursor := len(subject) - 1
	for cursor >= 0 {
		// 將給定的主題字串按照指定的分組分隔符進行解析, 並返回所有可能的分組組合.
		// 例如, subject = "1/2/3/", 可能的分組:
		// - ["1/2/3/", ""]
		// - ["1/2/", "3/"]
		// - ["1/", "2/3/"]
		prefix := subject[:cursor+1]
		postfix := subject[cursor+1:]

		handler, found := mux.handlers[prefix]
		if !found {
			cursor--
			for cursor >= 0 && subject[cursor] != mux.groupDelimiter[0] {
				cursor--
			}
			continue
		}

		groupMux, isGroup := handler.(*MessageMux[Subject, Message])
		if !isGroup {
			return LinkMiddlewares(handler.HandleMessage, mux.middlewares...)(message)
		}

		if groupMux.transform == nil {
			return handleMessage(groupMux, message, postfix)
		}

		message, err = groupMux.transform(message)
		if err != nil {
			return ErrorWrapWithMessage(err, "Failed to transform")
		}

		subject, err = groupMux.newSubject(message)
		if err != nil {
			return ErrorWrapWithMessage(err, "Failed to parse subject from the message")
		}

		return handleMessage(groupMux, message, subject)
	}

	if mux.defaultHandler != nil {
		return LinkMiddlewares(mux.defaultHandler, mux.middlewares...)(message)
	}

	if mux.notFoundHandler != nil {
		return mux.notFoundHandler(message)
	}

	return ErrorWrapWithMessage(ErrNotFound, "mux subject")
}

func parseGroupSubject(subject string, groupDelimiter string) [][]string {
	var result [][]string
	cursor := len(subject) - 1
	for {
		result = append(result, []string{subject[:cursor+1], subject[cursor+1:]})
		cursor = strings.LastIndexByte(subject[:cursor], groupDelimiter[0])
		if cursor == -1 {
			break
		}
	}
	return result
}

func (mux *MessageMux[Subject, Message]) Subjects() (result []string) {
	return mux.subjects()
}

func (mux *MessageMux[Subject, Message]) subjects() (result []string) {
	for subject, handler := range mux.handlers {
		m, ok := handler.(*MessageMux[Subject, Message])
		if ok {
			result = append(result, m.subjects()...)
			continue
		}
		result = append(result, mux.parentGroupName+subject)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return
}

func (mux *MessageMux[Subject, Message]) RegisterHandler(s Subject, h MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	subject := CleanSubject(s) + mux.groupDelimiter
	_, ok := mux.handlers[subject]
	if ok {
		panic(fmt.Sprintf("mux have duplicate subject=%v", subject))
	}

	mux.handlers[subject] = h
	return mux
}

func (mux *MessageMux[Subject, Message]) AddMiddleware(middlewares ...MessageDecorator[Message]) *MessageMux[Subject, Message] {
	mux.middlewares = append(mux.middlewares, middlewares...)
	return mux
}

func (mux *MessageMux[Subject, Message]) AddPreMiddleware(handlers ...MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	for _, handler := range handlers {
		mux.middlewares = append(mux.middlewares, handler.PreMiddleware())
	}
	return mux
}

func (mux *MessageMux[Subject, Message]) AddPostMiddleware(handlers ...MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	for _, handler := range handlers {
		mux.middlewares = append(mux.middlewares, handler.PostMiddleware())
	}
	return mux
}

func (mux *MessageMux[Subject, Message]) SetNotFoundHandler(h MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	mux.notFoundHandler = h
	return mux
}

func (mux *MessageMux[Subject, Message]) SetDefaultHandler(h MessageHandleFunc[Message]) *MessageMux[Subject, Message] {
	mux.defaultHandler = h
	return mux
}

func (mux *MessageMux[Subject, Message]) SetGroupDelimiter(delimiter string) *MessageMux[Subject, Message] {
	mux.groupDelimiter = delimiter
	return mux
}

func (mux *MessageMux[Subject, Message]) Group(s Subject) *MessageMux[Subject, Message] {
	groupName := CleanSubject(s) + mux.groupDelimiter
	_, ok := mux.handlers[groupName]
	if ok {
		panic(fmt.Sprintf("mux have duplicate group=%v", groupName))
	}

	groupMux := newGroupMux(mux, groupName)
	mux.handlers[groupName] = groupMux
	return groupMux
}

func CleanSubject[Subject constraints.Ordered](s Subject) string {
	actions := []func(s string) string{
		strings.TrimSpace,
		func(s string) string { return strings.Trim(s, `/.\`) },
		strings.TrimSpace,
	}

	subject := fmt.Sprintf("%v", s)
	for _, action := range actions {
		subject = action(subject)
	}
	return subject
}

func newGroupMux[Subject constraints.Ordered, Message any](parent *MessageMux[Subject, Message], groupName string) *MessageMux[Subject, Message] {
	return &MessageMux[Subject, Message]{
		logger:          SilentLogger(),
		parentGroupName: parent.parentGroupName + groupName,
		groupDelimiter:  parent.groupDelimiter,
		transform:       nil,
		newSubject:      parent.newSubject,
		handlers:        make(map[string]MessageHandler[Message]),
		middlewares:     append([]MessageDecorator[Message]{}, parent.middlewares...),
		notFoundHandler: parent.notFoundHandler,
		defaultHandler:  nil,
	}
}
