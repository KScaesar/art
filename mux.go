package Artifex

import (
	"fmt"
	"sync"

	"golang.org/x/exp/constraints"
)

type MessageHandler[Message any] func(dto Message) error

type MessageDecorator[Message any] func(next MessageHandler[Message]) MessageHandler[Message]

func PreMiddleware[Message any](handler MessageHandler[Message]) MessageDecorator[Message] {
	return func(next MessageHandler[Message]) MessageHandler[Message] {
		return func(dto Message) error {
			err := handler(dto)
			if err != nil {
				return err
			}
			return next(dto)
		}
	}
}

func PostMiddleware[Message any](handler MessageHandler[Message]) MessageDecorator[Message] {
	return func(next MessageHandler[Message]) MessageHandler[Message] {
		return func(dto Message) error {
			err := next(dto)
			if err != nil {
				return err
			}
			return handler(dto)
		}
	}
}

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
}

func (mux *MessageMux[Subject, Message]) handle(dto Message) (err error) {
	logger := mux.logger

	subject, err := mux.getSubject(dto)
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
		if mux.notFoundHandler == nil {
			return ErrNotFound
		}
		return mux.notFoundHandler(dto)
	}
	return LinkMiddlewares(fn)(dto)
}

func (mux *MessageMux[Subject, Message]) Subjects() (result []Subject) {
	for subject := range mux.handlers {
		result = append(result, subject)
	}
	return
}

func (mux *MessageMux[Subject, Message]) HandleMessageWithoutMutex(dto Message) error {
	return mux.handle(dto)
}

func (mux *MessageMux[Subject, Message]) HandleMessage(dto Message) error {
	mux.mu.RLock()
	defer mux.mu.RUnlock()
	return mux.handle(dto)
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

func (mux *MessageMux[Subject, Message]) SetNotFoundHandler(fn MessageHandler[Message]) *MessageMux[Subject, Message] {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.notFoundHandler = fn
	return mux
}
