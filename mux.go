package Artifex

import (
	"fmt"

	"github.com/gookit/goutil/maputil"
	"golang.org/x/exp/constraints"
)

// RouteParam are used to capture values from subject.
// These parameters represent resources or identifiers.
//
// Example:
//
//	define subject = "/users/{id}"
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

type HandleFunc[Message any] func(message *Message, route *RouteParam) error

func (h HandleFunc[Message]) PreMiddleware() Middleware[Message] {
	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(message *Message, route *RouteParam) error {
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
		return func(message *Message, route *RouteParam) error {
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

type NewSubjectFunc[Message any] func(*Message) (string, error)

func NewMux[Subject constraints.Ordered, Message any](routeDelimiter string, getSubject NewSubjectFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{
		getSubject: getSubject,
	}

	node := newTrie[Message](routeDelimiter)

	node.addRoute("", 0, handler)

	return &Mux[Subject, Message]{
		node: node,
		errorAndRecover: func(_ *Message, _ *RouteParam, err error) error {
			if r := recover(); r != nil {
				return fmt.Errorf("%v", r)
			}
			return err
		},

		enableAutoDelimiter: false,
		routeDelimiter:      routeDelimiter,
		delimiterAtLeft:     true,
	}
}

// Mux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a HandleFunc, but with added routing capabilities.
//
// Message represents a high-level abstraction data structure containing metadata (e.g. header) + body
type Mux[Subject constraints.Ordered, Message any] struct {
	node            *trie[Message]
	errorAndRecover func(*Message, *RouteParam, error) error

	enableAutoDelimiter bool
	delimiterAtLeft     bool
	routeDelimiter      string
}

// HandleMessage to handle various messages
//
// - route parameter can nil
func (mux *Mux[Subject, Message]) HandleMessage(message *Message, route *RouteParam) (err error) {
	if route == nil {
		route = routeParamPool.Get()
		defer func() {
			route.Reset()
			routeParamPool.Put(route)
		}()
	}

	defer func() {
		err = mux.errorAndRecover(message, route, err)
	}()

	if mux.node.transforms != nil {
		path := &routeHandler[Message]{
			middlewares: []Middleware[Message]{},
		}
		return mux.node.handleMessage("", 0, path, message, route)
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

func (mux *Mux[Subject, Message]) Handler(s Subject, h HandleFunc[Message]) *Mux[Subject, Message] {
	subject := mux.calcuSubject(s)
	handler := &routeHandler[Message]{
		handler: h,
	}
	mux.node.addRoute(subject, 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) Group(s Subject) *Mux[Subject, Message] {
	groupName := mux.calcuSubject(s)
	handler := &routeHandler[Message]{}
	groupNode := mux.node.addRoute(groupName, 0, handler)
	return &Mux[Subject, Message]{
		node:            groupNode,
		errorAndRecover: mux.errorAndRecover,

		enableAutoDelimiter: mux.enableAutoDelimiter,
		delimiterAtLeft:     mux.delimiterAtLeft,
		routeDelimiter:      mux.routeDelimiter,
	}
}

func (mux *Mux[Subject, Message]) Transform(transforms ...HandleFunc[Message]) *Mux[Subject, Message] {
	handler := &routeHandler[Message]{
		transforms: transforms,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Subject, Message]) SetSubjectFunc(getSubject NewSubjectFunc[Message]) *Mux[Subject, Message] {
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

func (mux *Mux[Subject, Message]) SetErrorAndRecoverHandler(errorAndRecover func(*Message, *RouteParam, error) error) *Mux[Subject, Message] {
	mux.errorAndRecover = errorAndRecover
	return mux
}

func (mux *Mux[Subject, Message]) Subjects() (result []string) {
	subjects, _ := mux.node.endpoint()
	return subjects
}

func (mux *Mux[Subject, Message]) calcuSubject(s Subject) string {
	if !mux.enableAutoDelimiter {
		return fmt.Sprintf("%v", s)
	}
	if mux.delimiterAtLeft {
		return mux.routeDelimiter + fmt.Sprintf("%v", s)
	}
	return fmt.Sprintf("%v", s) + mux.routeDelimiter
}

func (mux *Mux[Subject, Message]) SetAutoDelimiter(atLeft bool) *Mux[Subject, Message] {
	mux.enableAutoDelimiter = true
	mux.delimiterAtLeft = atLeft
	return mux
}
