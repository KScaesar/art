package ws

import (
	"context"

	"github.com/KScaesar/Artifex"
)

type Adapter = Artifex.Adapter[Topic, *WsMsg, *AppMsg]
type Session = Artifex.Session[Topic, *WsMsg, *AppMsg]

func NewSession(mux *ConsumeMux, factory *AdapterFactory) (*Session, error) {
	return Artifex.NewSession[Topic, *WsMsg, *AppMsg](mux, factory)
}

//

type Hub = Artifex.Artist[Topic, *WsMsg, *AppMsg]

func NewHub(mux *ConsumeMux) *Hub {
	return Artifex.NewArtist[Topic, *WsMsg, *AppMsg](mux)
}

//

type AdapterFactory struct {
}

func (f *AdapterFactory) CreateAdapter() (Adapter, error) {

	// TODO: create infra obj

	return Adapter{
		Recv: func(parent *Session) (*WsMsg, error) {
			return nil, nil
		},
		Send: func(message *AppMsg) error {
			return nil
		},
		Stop: func(message *AppMsg) {

		},
		Lifecycle: Artifex.Lifecycle[Topic, *WsMsg, *AppMsg]{
			SpawnHandlers: []func(sess *Session) error{},
			ExitHandlers:  []func(sess *Session){},
		},
		Identifier: "",
		Context:    context.Background(),
	}, nil
}
