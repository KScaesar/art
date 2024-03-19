package main

import (
	"context"

	"github.com/KScaesar/Artifex"
)

type Adapter = Artifex.Adapter[Topic, *SubMsg, *PubMsg]
type Session = Artifex.Session[Topic, *SubMsg, *PubMsg]

func NewSession(mux *ConsumeMux, factory *AdapterFactory) (*Session, error) {
	return Artifex.NewSession[Topic, *SubMsg, *PubMsg](mux, factory)
}

//

type Hub = Artifex.Artist[Topic, *SubMsg, *PubMsg]

func NewHub(mux *ConsumeMux) *Hub {
	return Artifex.NewArtist[Topic, *SubMsg, *PubMsg](mux)
}

//

type AdapterFactory struct {
}

func (f *AdapterFactory) CreateAdapter() (Adapter, error) {

	// TODO: create infra obj

	return Adapter{
		Recv: func(parent *Session) (*SubMsg, error) {
			return nil, nil
		},
		Send: func(message *PubMsg) error {
			return nil
		},
		Stop: func(message *PubMsg) {

		},
		Lifecycle: Artifex.Lifecycle[Topic, *SubMsg, *PubMsg]{
			SpawnHandlers: []func(sess *Session) error{},
			ExitHandlers:  []func(sess *Session){},
		},
		Identifier: "",
		Context:    context.Background(),
	}, nil
}
