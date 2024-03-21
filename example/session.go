package main

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

type Hub = Artifex.Artist[Channel, *ConsumeMsg, *ProduceMsg]

func NewHub() *Hub {
	// TODO
	return Artifex.NewArtist[Channel, *ConsumeMsg, *ProduceMsg]()
}

type Session = Artifex.Session[Channel, *ConsumeMsg, *ProduceMsg]

type Case1SessionFactory struct {
}

func (f *Case1SessionFactory) CreateSession() (*Session, error) {
	var mu sync.Mutex

	life := Artifex.Lifecycle[Channel, *ConsumeMsg, *ProduceMsg]{
		SpawnHandlers: []func(sess *Session) error{
			f.CreateAdapter(&mu),
			f.CreateAdapterWithPingPong(&mu),
			f.CreateAdapterWithFixup(&mu),
		},
		ExitHandlers: []func(sess *Session) error{},
	}
	sess := &Session{
		Mux:        nil,
		Identifier: "",
		Lifecycle:  life,
	}
	return sess, nil
}

func (f *Case1SessionFactory) CreateAdapterWithPingPong(mu *sync.Mutex) func(sess *Session) error {

	return func(sess *Session) error {
		pp := Artifex.PingPong{
			Enable:             true,
			IsWaitPingSendPong: false,
			SendFunc: func() error {
				mu.Lock()
				defer mu.Unlock()
				return nil
			},
			WaitNotify: make(chan error, 1),
			WaitSecond: 15,
		}
		sess.PingPong = pp

		sess.AdapterRecv = func() (*ConsumeMsg, error) {
			pp.WaitNotify <- nil

			return nil, nil
		}

		sess.AdapterSend = func(message *ProduceMsg) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		sess.AdapterStop = func(message *ProduceMsg) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		return nil
	}
}

func (f *Case1SessionFactory) CreateAdapter(mu *sync.Mutex) func(sess *Session) error {

	return func(sess *Session) error {
		sess.AdapterRecv = func() (*ConsumeMsg, error) {

			return nil, nil
		}

		sess.AdapterSend = func(message *ProduceMsg) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		sess.AdapterStop = func(message *ProduceMsg) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		return nil
	}
}

func (f *Case1SessionFactory) CreateAdapterWithFixup(mu *sync.Mutex) func(sess *Session) error {

	return func(sess *Session) error {
		sess.AdapterRecv = func() (*ConsumeMsg, error) {

			return nil, nil
		}

		sess.AdapterSend = func(message *ProduceMsg) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		sess.AdapterStop = func(message *ProduceMsg) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		sess.Fixup = func() error {

			return nil
		}

		return nil
	}
}
