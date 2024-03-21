package leaf

import (
	"github.com/KScaesar/Artifex"
)

// TODO
type LeafId = string

func NewWsMessage() *WsMessage {
	return &WsMessage{}
}

type WsMessage struct {
	// TODO
}

func NewLeafMessage() *LeafMessage {
	return &LeafMessage{}
}

type LeafMessage struct {
	// TODO
}

//

type WsMessageHandleFunc = Artifex.HandleFunc[*WsMessage]
type WsMessageMiddleware = Artifex.Middleware[*WsMessage]
type WsMessageMux = Artifex.Mux[LeafId, *WsMessage]

func NewWsMessageMux() *WsMessageMux {
	getLeafId := func(message *WsMessage) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[LeafId](getLeafId)
	mux.Handler("hello", HelloHandler())
	return mux
}

// Example
func HelloHandler() WsMessageHandleFunc {
	return func(message *WsMessage, route *Artifex.RouteParam) error {
		return nil
	}
}

//

type LeafMessageHandleFunc = Artifex.HandleFunc[*LeafMessage]
type LeafMessageMiddleware = Artifex.Middleware[*LeafMessage]
type LeafMessageMux = Artifex.Mux[LeafId, *LeafMessage]

func NewLeafMessageMux() *LeafMessageMux {
	getLeafId := func(message *LeafMessage) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[LeafId](getLeafId)
	mux.Handler("world", WorldHandler())
	return mux
}

// Example
func WorldHandler() LeafMessageHandleFunc {
	return func(message *LeafMessage, route *Artifex.RouteParam) error {
		return nil
	}
}
