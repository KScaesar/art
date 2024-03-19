package main

import (
	"github.com/KScaesar/Artifex"
)

type Topic = string

type SubMsg struct {
}

type PubMsg struct {
}

//

type ConsumeHandleFunc = Artifex.HandleFunc[*SubMsg]
type ConsumeMiddleware = Artifex.Middleware[*SubMsg]
type ConsumeMux = Artifex.Mux[Topic, *SubMsg]

func NewConsumeMux() *ConsumeMux {
	getTopic := func(message *SubMsg) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[Topic](getTopic)
	mux.Handler("hello", HelloHandler())
	return mux
}

func HelloHandler() ConsumeHandleFunc {
	return func(message *SubMsg, route *Artifex.RouteParam) error {
		return nil
	}
}

//

type ProduceHandleFunc = Artifex.HandleFunc[*PubMsg]
type ProduceMiddleware = Artifex.Middleware[*PubMsg]
type ProduceMux = Artifex.Mux[Topic, *PubMsg]

func NewProduceMux() *ProduceMux {
	getTopic := func(message *PubMsg) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[Topic](getTopic)
	mux.Handler("world", WorldHandler())
	return mux
}

func WorldHandler() ProduceHandleFunc {
	return func(message *PubMsg, route *Artifex.RouteParam) error {
		return nil
	}
}
