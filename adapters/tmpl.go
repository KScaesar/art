package main

const MsgTmpl = `
package {{.Package}}

import (
	"github.com/KScaesar/Artifex"
)

type {{.Subject}} = string

//

func New{{.RecvMessage}}() *{{.RecvMessage}} {
	return &{{.RecvMessage}}{}
}

type {{.RecvMessage}} struct {

}

type {{.RecvMessage}}HandleFunc = Artifex.HandleFunc[*{{.RecvMessage}}]
type {{.RecvMessage}}Middleware = Artifex.Middleware[*{{.RecvMessage}}]
type {{.RecvMessage}}Mux = Artifex.Mux[{{.Subject}}, *{{.RecvMessage}}]

func New{{.RecvMessage}}Mux() *{{.RecvMessage}}Mux {
	get{{.Subject}} := func(message *{{.RecvMessage}}) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("hello", {{.RecvMessage}}Handler())
	return mux
}

func {{.RecvMessage}}Handler() {{.RecvMessage}}HandleFunc {
	return func(message *{{.RecvMessage}}, _ *Artifex.RouteParam) error {
		return nil
	}
}

//

func New{{.SendMessage}}() *{{.SendMessage}} {
	return &{{.SendMessage}}{}
}

type {{.SendMessage}} struct {

}

type {{.SendMessage}}HandleFunc = Artifex.HandleFunc[*{{.SendMessage}}]
type {{.SendMessage}}Middleware = Artifex.Middleware[*{{.SendMessage}}]
type {{.SendMessage}}Mux = Artifex.Mux[{{.Subject}}, *{{.SendMessage}}]

func New{{.SendMessage}}Mux() *{{.SendMessage}}Mux {
	get{{.Subject}} := func(message *{{.SendMessage}}) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("world", {{.SendMessage}}Handler())
	return mux
}

func {{.SendMessage}}Handler() {{.SendMessage}}HandleFunc {
	return func(message *{{.SendMessage}}, _ *Artifex.RouteParam) error {
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

type {{.RecvMessage}}Hub = Artifex.Artist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

func New{{.RecvMessage}}Hub() *{{.RecvMessage}}Hub {
	return Artifex.NewArtist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]()
}

type {{.FileName}}Session = Artifex.Session[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

type {{.FileName}}SessionFactory struct {
	ApplyCase func() (*{{.FileName}}Session, error)
}

func (f *{{.FileName}}SessionFactory) CreateSession() (*{{.FileName}}Session, error) {
	return f.ApplyCase()
}

func (f *{{.FileName}}SessionFactory) UseCase1() (*{{.FileName}}Session, error) {
	var mu sync.Mutex

	// TODO

	life := Artifex.Lifecycle[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]{
		SpawnHandlers: []func(sess *{{.FileName}}Session) error{
			f.BuildAdapter(&mu),
			f.BuildAdapterWithPingPong(&mu),
			f.BuildAdapterWithFixup(&mu),
		},
		ExitHandlers: []func(sess *{{.FileName}}Session) error{},
	}
	sess := &{{.FileName}}Session{
		Mux:        nil,
		Identifier: "",
		Lifecycle:  life,
	}
	return sess, nil
}

func (f *{{.FileName}}SessionFactory) BuildAdapterWithPingPong(mu *sync.Mutex) func(sess *{{.FileName}}Session) error {

	return func(sess *{{.FileName}}Session) error {
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

		sess.AdapterStop = func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		return nil
	}
}

func (f *{{.FileName}}SessionFactory) BuildAdapter(mu *sync.Mutex) func(sess *{{.FileName}}Session) error {

	return func(sess *{{.FileName}}Session) error {
		sess.AdapterRecv = func() (*{{.RecvMessage}}, error) {

			return nil, nil
		}

		sess.AdapterSend = func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		sess.AdapterStop = func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		return nil
	}
}

func (f *{{.FileName}}SessionFactory) BuildAdapterWithFixup(mu *sync.Mutex) func(sess *{{.FileName}}Session) error {

	return func(sess *{{.FileName}}Session) error {
		sess.AdapterRecv = func() (*{{.RecvMessage}}, error) {

			return nil, nil
		}

		sess.AdapterSend = func(message *{{.SendMessage}}) error {
			mu.Lock()
			defer mu.Unlock()

			return nil
		}

		sess.AdapterStop = func(message *{{.SendMessage}}) error {
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
`
