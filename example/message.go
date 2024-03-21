package main

import (
	"github.com/KScaesar/Artifex"
)

// TODO
type Channel = string

func NewConsumeMsg() *ConsumeMsg {
	return &ConsumeMsg{}
}

type ConsumeMsg struct {
	// TODO
}

func NewProduceMsg() *ProduceMsg {
	return &ProduceMsg{}
}

type ProduceMsg struct {
	// TODO
}

//

type ConsumeMsgHandleFunc = Artifex.HandleFunc[*ConsumeMsg]
type ConsumeMsgMiddleware = Artifex.Middleware[*ConsumeMsg]
type ConsumeMsgMux = Artifex.Mux[Channel, *ConsumeMsg]

func NewConsumeMsgMux() *ConsumeMsgMux {
	getChannel := func(message *ConsumeMsg) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[Channel](getChannel)
	mux.Handler("hello", HelloHandler())
	return mux
}

// Example
func HelloHandler() ConsumeMsgHandleFunc {
	return func(message *ConsumeMsg, route *Artifex.RouteParam) error {
		return nil
	}
}

//

type ProduceMsgHandleFunc = Artifex.HandleFunc[*ProduceMsg]
type ProduceMsgMiddleware = Artifex.Middleware[*ProduceMsg]
type ProduceMsgMux = Artifex.Mux[Channel, *ProduceMsg]

func NewProduceMsgMux() *ProduceMsgMux {
	getChannel := func(message *ProduceMsg) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[Channel](getChannel)
	mux.Handler("world", WorldHandler())
	return mux
}

// Example
func WorldHandler() ProduceMsgHandleFunc {
	return func(message *ProduceMsg, route *Artifex.RouteParam) error {
		return nil
	}
}
