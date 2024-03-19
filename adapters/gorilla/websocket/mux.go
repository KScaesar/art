package ws

import (
	"github.com/KScaesar/Artifex"
)

type Topic = string

type WsMsg struct {
}

type AppMsg struct {
}

//

type ConsumeHandleFunc = Artifex.HandleFunc[*WsMsg]
type ConsumeMiddleware = Artifex.Middleware[*WsMsg]
type ConsumeMux = Artifex.Mux[Topic, *WsMsg]

func NewConsumeMux() *ConsumeMux {
	getTopic := func(message *WsMsg) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[Topic](getTopic)
	mux.Handler("hello", HelloHandler())
	return mux
}

func HelloHandler() ConsumeHandleFunc {
	return func(message *WsMsg, route *Artifex.RouteParam) error {
		return nil
	}
}

//

type ProduceHandleFunc = Artifex.HandleFunc[*AppMsg]
type ProduceMiddleware = Artifex.Middleware[*AppMsg]
type ProduceMux = Artifex.Mux[Topic, *AppMsg]

func NewProduceMux() *ProduceMux {
	getTopic := func(message *AppMsg) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[Topic](getTopic)
	mux.Handler("world", WorldHandler())
	return mux
}

func WorldHandler() ProduceHandleFunc {
	return func(message *AppMsg, route *Artifex.RouteParam) error {
		return nil
	}
}
