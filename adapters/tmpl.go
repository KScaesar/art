package main

const MsgTmpl = `
package {{.Package}}

import (
	"github.com/KScaesar/Artifex"
)

// TODO
type {{.Subject}} = string

func New{{.RecvMessage}}() *{{.RecvMessage}} {
	return &{{.RecvMessage}}{}
}

type {{.RecvMessage}} struct {
	// TODO
}

func New{{.SendMessage}}() *{{.SendMessage}} {
	return &{{.SendMessage}}{}
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

type Case1SessionFactory struct {

}

func (f *Case1SessionFactory) CreateSession() (*Session, error) {
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

		sess.AdapterRecv = func() (*{{.RecvMessage}}, error) {
			pp.WaitNotify <- nil
			return nil, nil
		}

		sess.AdapterSend = func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		sess.AdapterStop = func(message *{{.SendMessage}}) {
			return
		}

		return nil
	}
}

func SetupAdapter(mu *sync.Mutex) func(sess *Session) error {

	return func(sess *Session) error {
		sess.AdapterRecv = func() (*{{.RecvMessage}}, error) {
			return nil, nil
		}

		sess.AdapterSend = func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		sess.AdapterStop = func(message *{{.SendMessage}}) {
			return
		}

		return nil
	}
}

func SetupAdapterWithFixup(mu *sync.Mutex) func(sess *Session) error {

	return func(sess *Session) error {
		sess.AdapterRecv = func() (*{{.RecvMessage}}, error) {
			return nil, nil
		}

		sess.AdapterSend = func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}

		sess.AdapterStop = func(message *{{.SendMessage}}) {
			return
		}

		sess.Fixup = func() error {
			return nil
		}

		return nil
	}
}
`
