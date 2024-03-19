package main

import (
	"github.com/KScaesar/Artifex"
)

type Subject = string

type RecvMessage struct {
}

type SendMessage struct {
}

//

type RecvHandleFunc = Artifex.HandleFunc[*RecvMessage]
type RecvMiddleware = Artifex.Middleware[*RecvMessage]
type RecvMux = Artifex.Mux[Subject, *RecvMessage]

func NewRecvMux() *RecvMux {
	getSubject := func(message *RecvMessage) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[Subject](getSubject)
	mux.Handler("hello", HelloHandler())
	return mux
}

func HelloHandler() RecvHandleFunc {
	return func(message *RecvMessage, route *Artifex.RouteParam) error {
		return nil
	}
}

//

type SendHandleFunc = Artifex.HandleFunc[*SendMessage]
type SendMiddleware = Artifex.Middleware[*SendMessage]
type SendMux = Artifex.Mux[Subject, *SendMessage]

func NewSendMux() *SendMux {
	getSubject := func(message *SendMessage) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[Subject](getSubject)
	mux.Handler("world", WorldHandler())
	return mux
}

func WorldHandler() SendHandleFunc {
	return func(message *SendMessage, route *Artifex.RouteParam) error {
		return nil
	}
}
