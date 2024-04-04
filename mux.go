package Artifex

import (
	"strconv"
	"sync"

	"github.com/gookit/goutil/maputil"
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

type NewSubjectFunc[Message any] func(*Message) string

// NewMux
// If routeDelimiter is an empty string, RouteParam cannot be used.
// routeDelimiter can only be set to a string of length 1.
// this parameter determines the delimiter used between different parts of the route.
func NewMux[Message any](routeDelimiter string, getSubject NewSubjectFunc[Message]) *Mux[Message] {
	if len(routeDelimiter) > 1 {
		panic("routeDelimiter can only be set to a string of length 1.")
	}

	var qty int
	mux := &Mux[Message]{
		node:           newTrie[Message](routeDelimiter),
		routeDelimiter: routeDelimiter,
		middlewareQty:  &qty,
	}

	mux.SetSubjectFunc(getSubject)
	return mux
}

// Mux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a HandleFunc, but with added routing capabilities.
//
// Message represents a high-level abstraction data structure containing metadata (e.g. header) + body
type Mux[Message any] struct {
	node           *trie[Message]
	routeDelimiter string
	middlewareQty  *int

	messagePool  *sync.Pool
	resetMessage func(*Message)
}

// HandleMessage to handle various messages
//
// - route parameter can nil
func (mux *Mux[Message]) HandleMessage(message *Message, route *RouteParam) (err error) {
	if mux.messagePool != nil {
		defer func() {
			mux.resetMessage(message)
			mux.messagePool.Put(message)
		}()
	}

	if route == nil {
		route = routeParamPool.Get()
		defer func() {
			route.Reset()
			routeParamPool.Put(route)
		}()
	}

	path := &routeHandler[Message]{
		middlewares: make([]Middleware[Message], 0, *mux.middlewareQty),
	}

	if mux.node.transforms != nil {
		return mux.node.handleMessage("", 0, path, message, route)
	}

	subject := mux.node.getSubject(message)
	return mux.node.handleMessage(subject, 0, path, message, route)
}

func (mux *Mux[Message]) Handler(subject string, h HandleFunc[Message]) *Mux[Message] {
	handler := &routeHandler[Message]{
		handler: h,
	}
	mux.node.addRoute(subject, 0, handler)
	return mux
}

func (mux *Mux[Message]) HandlerByNumber(subject int, h HandleFunc[Message]) *Mux[Message] {
	return mux.Handler(mux.routeDelimiter+strconv.Itoa(subject), h)
}

func (mux *Mux[Message]) Group(groupName string) *Mux[Message] {
	handler := &routeHandler[Message]{}
	groupNode := mux.node.addRoute(groupName, 0, handler)
	return &Mux[Message]{
		node:           groupNode,
		routeDelimiter: mux.routeDelimiter,
		middlewareQty:  mux.middlewareQty,
		messagePool:    mux.messagePool,
		resetMessage:   mux.resetMessage,
	}
}

func (mux *Mux[Message]) GroupByNumber(groupName int) *Mux[Message] {
	return mux.Group(mux.routeDelimiter + strconv.Itoa(groupName))
}

func (mux *Mux[Message]) Transform(transforms ...HandleFunc[Message]) *Mux[Message] {
	handler := &routeHandler[Message]{
		transforms: transforms,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Message]) SetSubjectFunc(getSubject NewSubjectFunc[Message]) *Mux[Message] {
	handler := &routeHandler[Message]{
		getSubject: getSubject,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Message]) Middleware(middlewares ...Middleware[Message]) *Mux[Message] {
	*mux.middlewareQty += len(middlewares)
	handler := &routeHandler[Message]{
		middlewares: middlewares,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Message]) PreMiddleware(handleFuncs ...HandleFunc[Message]) *Mux[Message] {
	*mux.middlewareQty += len(handleFuncs)
	handler := &routeHandler[Message]{}
	for _, h := range handleFuncs {
		handler.middlewares = append(handler.middlewares, h.PreMiddleware())
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Message]) PostMiddleware(handleFuncs ...HandleFunc[Message]) *Mux[Message] {
	*mux.middlewareQty += len(handleFuncs)
	handler := &routeHandler[Message]{}
	for _, h := range handleFuncs {
		handler.middlewares = append(handler.middlewares, h.PostMiddleware())
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Message]) SetDefaultHandler(h HandleFunc[Message]) *Mux[Message] {
	handler := &routeHandler[Message]{
		defaultHandler: h,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Message]) SetNotFoundHandler(h HandleFunc[Message]) *Mux[Message] {
	handler := &routeHandler[Message]{
		notFoundHandler: h,
	}
	mux.node.addRoute("", 0, handler)
	return mux
}

func (mux *Mux[Message]) SetMessagePool(pool *sync.Pool, reset func(*Message)) *Mux[Message] {
	mux.messagePool = pool
	mux.resetMessage = reset
	return mux
}

func (mux *Mux[Message]) GetSubjectAndHandler() (subjects, functions []string) {
	return mux.node.endpoint()
}
