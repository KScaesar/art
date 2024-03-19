package Artifex

import (
	"fmt"
	"strings"

	"github.com/gookit/goutil/maputil"
	"golang.org/x/exp/constraints"
)

// RouteParam are used to capture values from subject.
// These parameters represent resources or identifiers.
//
// Example:
//
//	define subject = "/users/:id"
//	ingress message subject = /users/1017
//
//	route:
//		key : value
//	→	id  : 1017
type RouteParam struct {
	maputil.Data
}

func (r *RouteParam) Reset() {
	for k := range r.Data {
		delete(r.Data, k)
	}
}

type HandleFunc[Message any] func(message Message, route *RouteParam) error

func (h HandleFunc[Message]) PreMiddleware() Middleware[Message] {
	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(message Message, route *RouteParam) error {
			err := h(message, route)
			if err != nil {
				return err
			}
			return next(message, route)
		}
	}
}

func (h HandleFunc[Message]) PostMiddleware() Middleware[Message] {
	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(message Message, route *RouteParam) error {
			err := next(message, route)
			if err != nil {
				return err
			}
			return h(message, route)
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

	node := newTrie[Message]()
	node.addRoute("", 0, handler)

	return &Mux[Subject, Message]{
		node: node,
		errorHandler: func(_ Message, _ *RouteParam, err error) error {
			return err
		},
	}
}

// Mux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a HandleFunc, but with added routing capabilities.
//
// Message represents a high-level abstraction data structure containing metadata (e.g. header) + body
type Mux[Subject constraints.Ordered, Message any] struct {
	node            *trie[Message]
	delimiter       string
	delimiterAtLeft bool
	isCleanSubject  bool
	errorHandler    func(Message, *RouteParam, error) error
}

// HandleMessage to handle various messages coming from the adapter
//
// - route parameter can nil
func (mux *Mux[Subject, Message]) HandleMessage(message Message, route *RouteParam) (err error) {
	if route == nil {
		route = routeParamPool.Get()
		defer func() {
			route.Reset()
			routeParamPool.Put(route)
		}()
	}

	defer func() {
		if err != nil {
			err = mux.errorHandler(message, route, err)
		}
	}()

	if mux.node.transforms != nil {
		err = mux.node.transform(message, route)
		if err != nil {
			return err
		}
	}

	subject, err := mux.node.getSubject(message)
	if err != nil {
		return err
	}

	path := &routeHandler[Message]{
		middlewares: []Middleware[Message]{},
	}
	return mux.node.handleMessage(subject, 0, path, message, route)
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
		subject = mux.delimiter + CleanSubject(s, mux.isCleanSubject)
	} else {
		subject = CleanSubject(s, mux.isCleanSubject) + mux.delimiter
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

func (mux *Mux[Subject, Message]) SetErrorHandler(errorHandler func(Message, *RouteParam, error) error) {
	mux.errorHandler = errorHandler
}

func (mux *Mux[Subject, Message]) Group(s Subject) *Mux[Subject, Message] {
	var groupName string
	if mux.delimiterAtLeft {
		groupName = mux.delimiter + CleanSubject(s, mux.isCleanSubject)
	} else {
		groupName = CleanSubject(s, mux.isCleanSubject) + mux.delimiter
	}

	handler := &routeHandler[Message]{}
	groupNode := mux.node.addRoute(groupName, 0, handler)
	return &Mux[Subject, Message]{
		node:            groupNode,
		delimiter:       mux.delimiter,
		delimiterAtLeft: mux.delimiterAtLeft,
		isCleanSubject:  mux.isCleanSubject,
	}
}

func (mux *Mux[Subject, Message]) SetDelimiter(delimiter string, atLeft bool) *Mux[Subject, Message] {
	mux.delimiter = delimiter
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
