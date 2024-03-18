package Artifex

import (
	"fmt"
	"strings"

	"golang.org/x/exp/constraints"
)

type HandleFunc[Message any] func(message Message) error

func (h HandleFunc[Message]) PreMiddleware() Middleware[Message] {
	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(message Message) error {
			err := h(message)
			if err != nil {
				return err
			}
			return next(message)
		}
	}
}

func (h HandleFunc[Message]) PostMiddleware() Middleware[Message] {
	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(message Message) error {
			err := next(message)
			if err != nil {
				return err
			}
			return h(message)
		}
	}
}

func (h HandleFunc[Message]) LinkMiddlewares(middlewares ...Middleware[Message]) HandleFunc[Message] {
	return LinkMiddlewares(h, middlewares...)
}

type Middleware[Message any] func(next HandleFunc[Message]) HandleFunc[Message]

func LinkMiddlewares[Message any](handler HandleFunc[Message], middlewares ...Middleware[Message]) HandleFunc[Message] {
	n := len(middlewares)
	for i := n - 1; 0 <= i; i-- {
		decorator := middlewares[i]
		handler = decorator(handler)
	}
	return handler
}

//

type NewSubjectFunc[Message any] func(Message) (string, error)

func NewMux[Subject constraints.Ordered, Message any](getSubject NewSubjectFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{
		getSubject: getSubject,
	}

	node := &trie[Message]{
		child: make(map[string]*trie[Message]),
	}
	node.addRoute("", 0, handler)

	return &Mux[Subject, Message]{
		logger: DefaultLogger(),
		node:   node,
	}
}

// Mux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a HandleFunc, but with added routing capabilities.
//
// Message represents a high-level abstraction data structure containing metadata (e.g. header) + body
type Mux[Subject constraints.Ordered, Message any] struct {
	logger          Logger
	node            *trie[Message]
	groupDelimiter  string
	delimiterAtLeft bool
	isCleanSubject  bool
}

func (mux *Mux[Subject, Message]) HandleMessage(message Message) (err error) {
	if mux.node.transforms != nil {
		err = mux.node.transform(message)
		if err != nil {
			mux.logger.Error("%v", err)
			return err
		}
	}

	subject, err := mux.node.getSubject(message)
	if err != nil {
		mux.logger.Error("%v", err)
		return err
	}
	mux.logger.Debug("handle subject=%v", subject)

	path := &routeHandler[Message]{}
	err = mux.node.handleMessage(subject, 0, path, message)
	if err != nil {
		mux.logger.Error("hande subject=%v fail: %v", subject, err)
		return err
	}
	mux.logger.Debug("hande subject=%v success", subject)
	return nil
}

func (mux *Mux[Subject, Message]) SetLogger(logger Logger) *Mux[Subject, Message] {
	mux.logger = logger
	return mux
}

func (mux *Mux[Subject, Message]) Transform(transforms ...HandleFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{
		transforms: transforms,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) SubjectFunc(getSubject NewSubjectFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{
		getSubject: getSubject,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) Middleware(middlewares ...Middleware[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{
		middlewares: middlewares,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) PreMiddleware(handleFuncs ...HandleFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{}
	for _, h := range handleFuncs {
		handler.middlewares = append(handler.middlewares, h.PreMiddleware())
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) PostMiddleware(handleFuncs ...HandleFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{}
	for _, h := range handleFuncs {
		handler.middlewares = append(handler.middlewares, h.PostMiddleware())
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) Handler(s Subject, h HandleFunc[Message]) *Mux[Subject, Message] {
	var subject string
	if mux.delimiterAtLeft {
		subject = mux.groupDelimiter + CleanSubject(s, mux.isCleanSubject)
	} else {
		subject = CleanSubject(s, mux.isCleanSubject) + mux.groupDelimiter
	}

	handler := &routeHandler[Message]{
		handler: h,
	}
	mux.node.addRoute(subject, 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) SetDefaultHandler(h HandleFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{
		defaultHandler: h,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) SetNotFoundHandler(h HandleFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{
		notFoundHandler: h,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) Group(s Subject) *Mux[Subject, Message] {
	var groupName string
	if mux.delimiterAtLeft {
		groupName = mux.groupDelimiter + CleanSubject(s, mux.isCleanSubject)
	} else {
		groupName = CleanSubject(s, mux.isCleanSubject) + mux.groupDelimiter
	}

	handler := &routeHandler[Message]{}
	groupNode := mux.node.addRoute(groupName, 0, handler)
	return &Mux[Subject, Message]{
		logger:          mux.logger,
		node:            groupNode,
		groupDelimiter:  mux.groupDelimiter,
		delimiterAtLeft: mux.delimiterAtLeft,
		isCleanSubject:  mux.isCleanSubject,
	}
}

func (mux *Mux[Subject, Message]) SetGroupDelimiter(groupDelimiter string, atLeft bool) *Mux[Subject, Message] {
	mux.groupDelimiter = groupDelimiter
	mux.delimiterAtLeft = atLeft
	return mux
}

func (mux *Mux[Subject, Message]) Subjects() (result []string) {
	subjects, _ := mux.node.endpoint()
	return subjects
}

func (mux *Mux[Subject, Message]) SetCleanSubject(cleanSubject bool) *Mux[Subject, Message] {
	mux.isCleanSubject = cleanSubject
	return mux
}

func CleanSubject[Subject constraints.Ordered](s Subject, isClean bool) string {
	actions := []func(s string) string{
		strings.TrimSpace,
		func(s string) string { return strings.Trim(s, `/.\`) },
		strings.TrimSpace,
	}

	subject := fmt.Sprintf("%v", s)
	if !isClean {
		return subject
	}

	for _, action := range actions {
		subject = action(subject)
	}
	return subject
}
