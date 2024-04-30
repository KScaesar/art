package Artifex

import (
	"strconv"
)

// NewMux
// If routeDelimiter is an empty string, Message.RouteParam cannot be used.
// RouteDelimiter can only be set to a string of length 1.
// This parameter determines different parts of the Message.Subject.
func NewMux(routeDelimiter string) *Mux {
	if len(routeDelimiter) > 1 {
		panic("routeDelimiter can only be set to a string of length 1.")
	}

	var mux = &Mux{
		node:           newTrie(routeDelimiter),
		routeDelimiter: routeDelimiter,
	}
	return mux
}

// Mux refers to a router or multiplexer, which can be used to handle different message.
// Itself is also a HandleFunc, but with added routing capabilities.
//
// Message represents a high-level abstraction data structure containing metadata (e.g. header) + body
type Mux struct {
	node              *trie
	routeDelimiter    string
	errorHandlers     []Middleware
	enableMessagePool bool
}

// HandleMessage to handle various messages
//
// - route parameter can nil
func (mux *Mux) HandleMessage(message *Message, dependency any) (err error) {
	if mux.enableMessagePool {
		defer PutMessage(message)
	}

	defer func() {
		if mux.errorHandlers != nil {
			err = Link(func(message *Message, dep any) error {
				return err
			}, mux.errorHandlers...)(message, dependency)
		}
	}()

	if mux.node.transform != nil {
		return mux.node.handleMessage("", 0, message, dependency)
	}

	return mux.node.handleMessage(message.Subject, 0, message, dependency)
}

// Middleware
// Before registering handler, middleware must be defined; otherwise, the handler won't be able to use middleware.
func (mux *Mux) Middleware(middlewares ...Middleware) *Mux {
	param := &paramHandler{
		middlewares: middlewares,
	}

	mux.node.addRoute("", 0, param, []Middleware{})
	return mux
}

func (mux *Mux) PreMiddleware(handleFuncs ...HandleFunc) *Mux {
	param := &paramHandler{}
	for _, h := range handleFuncs {
		param.middlewares = append(param.middlewares, h.PreMiddleware())
	}

	mux.node.addRoute("", 0, param, []Middleware{})
	return mux
}

func (mux *Mux) PostMiddleware(handleFuncs ...HandleFunc) *Mux {
	param := &paramHandler{}
	for _, h := range handleFuncs {
		param.middlewares = append(param.middlewares, h.PostMiddleware())
	}

	mux.node.addRoute("", 0, param, []Middleware{})
	return mux
}

// Transform
// Originally, the message passed through the mux would only call 'getSubject' once.
// However, if there is a definition of Transform,
// when the message passes through the Transform function, 'getSubject' will be called again.
func (mux *Mux) Transform(transform HandleFunc) *Mux {
	param := &paramHandler{
		transform: transform,
	}

	mux.node.addRoute("", 0, param, []Middleware{})
	return mux
}

func (mux *Mux) Handler(subject string, h HandleFunc, mw ...Middleware) *Mux {
	param := &paramHandler{
		handler: h,
	}
	if mw != nil {
		param.handler = Link(param.handler, mw...)
		param.handlerName = functionName(h)
	}

	mux.node.addRoute(subject, 0, param, []Middleware{})
	return mux
}

func (mux *Mux) HandlerByNumber(subject int, h HandleFunc, mw ...Middleware) *Mux {
	return mux.Handler(strconv.Itoa(subject)+mux.routeDelimiter, h, mw...)
}

func (mux *Mux) Group(groupName string) *Mux {
	groupNode := mux.node.addRoute(groupName, 0, nil, []Middleware{})
	return &Mux{
		node:              groupNode,
		routeDelimiter:    mux.routeDelimiter,
		errorHandlers:     mux.errorHandlers,
		enableMessagePool: mux.enableMessagePool,
	}
}

func (mux *Mux) GroupByNumber(groupName int) *Mux {
	return mux.Group(strconv.Itoa(groupName) + mux.routeDelimiter)
}

// DefaultHandler
// When a subject cannot be found, execute the 'Default'.
//
// "The difference between 'Default' and 'NotFound' is
// that the 'Default' handler will utilize middleware,
// whereas 'NotFound' won't use middleware."
func (mux *Mux) DefaultHandler(h HandleFunc, mw ...Middleware) *Mux {
	param := &paramHandler{
		defaultHandler: h,
	}
	if mw != nil {
		param.defaultHandler = Link(param.defaultHandler, mw...)
		param.defaultHandlerName = functionName(h)
	}

	mux.node.addRoute("", 0, param, []Middleware{})
	return mux
}

// NotFoundHandler
// When a subject cannot be found, execute the 'NotFound'.
//
// "The difference between 'Default' and 'NotFound' is
// that the 'Default' handler will utilize middleware,
// whereas 'NotFound' won't use middleware."
func (mux *Mux) NotFoundHandler(h HandleFunc) *Mux {
	param := &paramHandler{
		notFoundHandler: h,
	}

	mux.node.addRoute("", 0, param, []Middleware{})
	return mux
}

func (mux *Mux) ErrorHandler(errHandlers ...Middleware) *Mux {
	mux.errorHandlers = append(mux.errorHandlers, errHandlers...)
	return mux
}

func (mux *Mux) EnableMessagePool() *Mux {
	mux.enableMessagePool = true
	return mux
}

// Endpoints get register handler function information
func (mux *Mux) Endpoints(action func(subject, handler string)) {
	for _, v := range mux.node.endpoint() {
		action(v[0], v[1])
	}
}
