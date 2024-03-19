package main

import (
	"context"

	"github.com/KScaesar/Artifex"
)

type Adapter = Artifex.Adapter[Subject, *RecvMessage, *SendMessage]
type Session = Artifex.Session[Subject, *RecvMessage, *SendMessage]

func NewSession(recvMux *RecvMux, factory *AdapterFactory) (*Session, error) {
	return Artifex.NewSession[Subject, *RecvMessage, *SendMessage](recvMux, factory)
}

//

type Hub = Artifex.Artist[Subject, *RecvMessage, *SendMessage]

func NewHub(recvMux *RecvMux) *Hub {
	return Artifex.NewArtist[Subject, *RecvMessage, *SendMessage](recvMux)
}

//

type AdapterFactory struct {
}

func (f *AdapterFactory) CreateAdapter() (Adapter, error) {

	// TODO: create infra obj

	return Adapter{
		Recv: func(parent *Session) (*RecvMessage, error) {
			return nil, nil
		},
		Send: func(message *SendMessage) error {
			return nil
		},
		Stop: func(message *SendMessage) {

		},
		Lifecycle: Artifex.Lifecycle[Subject, *RecvMessage, *SendMessage]{
			SpawnHandlers: []func(sess *Session) error{},
			ExitHandlers:  []func(sess *Session){},
		},
		Identifier: "",
		Context:    context.Background(),
	}, nil
}
