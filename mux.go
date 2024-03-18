package Artifex

import (
	"fmt"
	"strings"

	"golang.org/x/exp/constraints"
)

type MessageHandler[Message any] func(message Message) error

func (fn MessageHandler[Message]) HandleMessage(message Message) error {
	return fn(message)
}

func (fn MessageHandler[Message]) PreMiddleware() MessageDecorator[Message] {
	return func(next MessageHandler[Message]) MessageHandler[Message] {
		return func(message Message) error {
			err := fn(message)
			if err != nil {
				return err
			}
			return next(message)
		}
	}
}

func (fn MessageHandler[Message]) PostMiddleware() MessageDecorator[Message] {
	return func(next MessageHandler[Message]) MessageHandler[Message] {
		return func(message Message) error {
			err := next(message)
			if err != nil {
				return err
			}
			return fn(message)
		}
	}
}

func (fn MessageHandler[Message]) LinkMiddlewares(middlewares ...MessageDecorator[Message]) MessageHandler[Message] {
	return LinkMiddlewares(fn, middlewares...)
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

type NewSubjectFunc[Message any] func(Message) (string, error)

func NewMessageMux[Subject constraints.Ordered, Message any](getSubject NewSubjectFunc[Message]) *MessageMux[Subject, Message] {
	node := &trie[Message]{
		kind: rootKind,
	}
	node.insert("", 0, handlerParam[Message]{
		getSubject: getSubject,
	})
	return &MessageMux[Subject, Message]{
		logger:         DefaultLogger(),
		groupDelimiter: "/",
		node:           node,
	}
}

// MessageMux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a MessageHandler, but with added routing capabilities.
//
// rMessage represents a high-level abstraction data structure containing metadata (e.g. header) + body
type MessageMux[Subject constraints.Ordered, Message any] struct {
	logger         Logger
	groupDelimiter string
	node           *trie[Message]
}

func (mux *MessageMux[Subject, Message]) SetLogger(logger Logger) *MessageMux[Subject, Message] {
	mux.logger = logger
	return mux
}

func (mux *MessageMux[Subject, Message]) Transform(transforms []func(old Message) (fresh Message, err error)) *MessageMux[Subject, Message] {
	mux.node.insert("", 0, handlerParam[Message]{
		transforms: transforms,
	})
	return mux
}

func (mux *MessageMux[Subject, Message]) HandleMessage(message Message) (err error) {
	if mux.node.transforms != nil {
		for _, transform := range mux.node.transforms {
			message, err = transform(message)
			if err != nil {
				Err := ErrorWrapWithMessage(err, "Failed to transform")
				mux.logger.Error("%v", Err)
				return err
			}
		}
	}

	subject, err := mux.node.getSubject(message)
	if err != nil {
		Err := ErrorWrapWithMessage(err, "Failed to parse subject from the message")
		mux.logger.Error("%v", Err)
		return Err
	}
	mux.logger.Debug("handle subject=%v", subject)

	err = mux.node.handle(subject, 0, message)
	if err != nil {
		mux.logger.Error("hande subject=%v fail: %v", subject, err)
		return err
	}
	mux.logger.Debug("hande subject=%v success", subject)
	return nil
}

func (mux *MessageMux[Subject, Message]) Subjects() (result []string) {
	subjects, _ := mux.node.endpoint()
	return subjects
}

func (mux *MessageMux[Subject, Message]) Handler(s Subject, h MessageHandler[Message]) *MessageMux[Subject, Message] {
	subject := CleanSubject(s) + mux.groupDelimiter
	mux.node.insert(subject, 0, handlerParam[Message]{
		handler: h,
	})
	return mux
}

func (mux *MessageMux[Subject, Message]) Middleware(middlewares ...MessageDecorator[Message]) *MessageMux[Subject, Message] {
	mux.node.insert("", 0, handlerParam[Message]{
		middlewares: middlewares,
	})
	return mux
}

func (mux *MessageMux[Subject, Message]) AddPreMiddleware(handlers ...MessageHandler[Message]) *MessageMux[Subject, Message] {
	param := handlerParam[Message]{}
	for _, handler := range handlers {
		param.middlewares = append(param.middlewares, handler.PreMiddleware())
	}
	mux.node.insert("", 0, param)
	return mux
}

func (mux *MessageMux[Subject, Message]) AddPostMiddleware(handlers ...MessageHandler[Message]) *MessageMux[Subject, Message] {
	param := handlerParam[Message]{}
	for _, handler := range handlers {
		param.middlewares = append(param.middlewares, handler.PostMiddleware())
	}
	mux.node.insert("", 0, param)
	return mux
}

func (mux *MessageMux[Subject, Message]) SetNotFoundHandler(h MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.node.insert("", 0, handlerParam[Message]{
		notFoundHandler: h,
	})
	return mux
}

func (mux *MessageMux[Subject, Message]) SetDefaultHandler(h MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.node.insert("", 0, handlerParam[Message]{
		defaultHandler: h,
	})
	return mux
}

func (mux *MessageMux[Subject, Message]) SetGroupDelimiter(delimiter string) *MessageMux[Subject, Message] {
	mux.groupDelimiter = delimiter
	return mux
}

func (mux *MessageMux[Subject, Message]) Group(s Subject) *MessageMux[Subject, Message] {
	groupName := CleanSubject(s) + mux.groupDelimiter
	groupNode := mux.node.insert(groupName, 0, handlerParam[Message]{})
	return &MessageMux[Subject, Message]{
		logger:         mux.logger,
		groupDelimiter: mux.groupDelimiter,
		node:           groupNode,
	}
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
