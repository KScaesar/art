package main

const MsgTmpl = `
package {{.Package}}

import (
	"github.com/KScaesar/Artifex"
)

// TODO
type {{.Subject}} = string

type {{.RecvMessage}} struct {
	// TODO
}

type {{.SendMessage}} struct {
	// TODO
}

//

type {{.RecvMessage}}HandleFunc = Artifex.HandleFunc[*{{.RecvMessage}}]
type {{.RecvMessage}}Middleware = Artifex.Middleware[*{{.RecvMessage}}]
type {{.RecvMessage}}Mux = Artifex.Mux[{{.Subject}}, *{{.RecvMessage}}]

func New{{.RecvMessage}}Mux() *{{.RecvMessage}}Mux {
	get{{.Subject}} := func(message *{{.RecvMessage}}) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("hello", HelloHandler())
	return mux
}

// Example
func HelloHandler() {{.RecvMessage}}HandleFunc {
	return func(message *{{.RecvMessage}}, route *Artifex.RouteParam) error {
		return nil
	}
}

//

type {{.SendMessage}}HandleFunc = Artifex.HandleFunc[*{{.SendMessage}}]
type {{.SendMessage}}Middleware = Artifex.Middleware[*{{.SendMessage}}]
type {{.SendMessage}}Mux = Artifex.Mux[{{.Subject}}, *{{.SendMessage}}]

func New{{.SendMessage}}Mux() *{{.SendMessage}}Mux {
	get{{.Subject}} := func(message *{{.SendMessage}}) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("world", WorldHandler())
	return mux
}

// Example
func WorldHandler() {{.SendMessage}}HandleFunc {
	return func(message *{{.SendMessage}}, route *Artifex.RouteParam) error {
		return nil
	}
}
`

const SessionTmpl = `
package {{.Package}}

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

type Hub = Artifex.Artist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

func NewHub() *Hub {
	// TODO
	return Artifex.NewArtist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]()
}

type Session = Artifex.Session[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

type SessionFactory struct {
}

func (f *SessionFactory) CreateSession() (*Session, error) {
	var mu sync.Mutex
	life := Artifex.Lifecycle[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]{
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
	// TODO: create infra obj

	pp := Artifex.PingPong{
		Enable:             true,
		IsSendPingWaitPong: false,
		SendFunc: func() error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		},
		WaitNotify: make(chan error, 1),
		WaitSecond: 15,
	}

	return func(sess *Session) error {
		recv := func() (*{{.RecvMessage}}, error) {
			pp.WaitNotify <- nil
			return nil, nil
		}

		send := func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		stop := func(message *{{.SendMessage}}) {
			return
		}

		sess.AdapterRecv = recv
		sess.AdapterSend = send
		sess.AdapterStop = stop
		sess.PingPong = pp
		return nil
	}
}

func SetupAdapter(mu *sync.Mutex) func(sess *Session) error {
	// TODO: create infra obj

	return func(sess *Session) error {
		recv := func() (*{{.RecvMessage}}, error) {
			return nil, nil
		}

		send := func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		stop := func(message *{{.SendMessage}}) {
			return
		}

		sess.AdapterRecv = recv
		sess.AdapterSend = send
		sess.AdapterStop = stop
		return nil
	}
}

func SetupAdapterWithFixup(mu *sync.Mutex) func(sess *Session) error {
	// TODO: create infra obj

	return func(sess *Session) error {
		recv := func() (*{{.RecvMessage}}, error) {
			return nil, nil
		}

		send := func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		stop := func(message *{{.SendMessage}}) {
			return
		}

		fixup := func() error {
			return nil
		}

		sess.AdapterRecv = recv
		sess.AdapterSend = send
		sess.AdapterStop = stop
		sess.Fixup = fixup
		return nil
	}
}
`
