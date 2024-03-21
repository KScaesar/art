package leaf

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

type Hub = Artifex.Artist[LeafId, *WsMessage, *LeafMessage]

func NewHub() *Hub {
	// TODO
	return Artifex.NewArtist[LeafId, *WsMessage, *LeafMessage]()
}

type Session = Artifex.Session[LeafId, *WsMessage, *LeafMessage]

type Case1SessionFactory struct {
}

func (f *Case1SessionFactory) CreateSession() (*Session, error) {
	var mu sync.Mutex
	life := Artifex.Lifecycle[LeafId, *WsMessage, *LeafMessage]{
		SpawnHandlers: []func(sess *Session) error{
			SetupAdapter(&mu),
			SetupAdapterWithPingPong(&mu),
			SetupAdapterWithFixup(&mu),
		},
		ExitHandlers: []func(sess *Session){},
	}
	sess := &Session{
		Mux:        nil,
		Identifier: "",
		Lifecycle:  life,
	}
	return sess, nil
}

func SetupAdapterWithPingPong(mu *sync.Mutex) func(sess *Session) error {

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

		sess.AdapterRecv = func() (*WsMessage, error) {
			pp.WaitNotify <- nil
			return nil, nil
		}

		sess.AdapterSend = func(message *LeafMessage) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		sess.AdapterStop = func(message *LeafMessage) {
			return
		}

		return nil
	}
}

func SetupAdapter(mu *sync.Mutex) func(sess *Session) error {

	return func(sess *Session) error {
		sess.AdapterRecv = func() (*WsMessage, error) {
			return nil, nil
		}

		sess.AdapterSend = func(message *LeafMessage) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		sess.AdapterStop = func(message *LeafMessage) {
			return
		}

		return nil
	}
}

func SetupAdapterWithFixup(mu *sync.Mutex) func(sess *Session) error {

	return func(sess *Session) error {
		sess.AdapterRecv = func() (*WsMessage, error) {
			return nil, nil
		}

		sess.AdapterSend = func(message *LeafMessage) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		sess.AdapterStop = func(message *LeafMessage) {
			return
		}

		sess.Fixup = func() error {
			return nil
		}

		return nil
	}
}
