package main

const MuxTmpl = `
package {{.Package}}

import (
	"github.com/KScaesar/Artifex"
)

type {{.Subject}} = string

type {{.RecvMessage}} struct {
}

type {{.SendMessage}} struct {
}

//

type RecvHandleFunc = Artifex.HandleFunc[*{{.RecvMessage}}]
type RecvMiddleware = Artifex.Middleware[*{{.RecvMessage}}]
type RecvMux = Artifex.Mux[{{.Subject}}, *{{.RecvMessage}}]

func NewRecvMux() *RecvMux {
	get{{.Subject}} := func(message *{{.RecvMessage}}) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("hello", HelloHandler())
	return mux
}

func HelloHandler() RecvHandleFunc {
	return func(message *{{.RecvMessage}}, route *Artifex.RouteParam) error {
		return nil
	}
}

//

type SendHandleFunc = Artifex.HandleFunc[*{{.SendMessage}}]
type SendMiddleware = Artifex.Middleware[*{{.SendMessage}}]
type SendMux = Artifex.Mux[{{.Subject}}, *{{.SendMessage}}]

func NewSendMux() *SendMux {
	get{{.Subject}} := func(message *{{.SendMessage}}) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("world", WorldHandler())
	return mux
}

func WorldHandler() SendHandleFunc {
	return func(message *{{.SendMessage}}, route *Artifex.RouteParam) error {
		return nil
	}
}
`

const SessionTmpl = `
package {{.Package}}

import (
	"context"

	"github.com/KScaesar/Artifex"
)

type Adapter = Artifex.Adapter[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]
type Session = Artifex.Session[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

func NewSession(recvMux *RecvMux, factory *AdapterFactory) (*Session, error) {
	return Artifex.NewSession[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}](recvMux, factory)
}

//

type Hub = Artifex.Artist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

func NewHub(recvMux *RecvMux) *Hub {
	return Artifex.NewArtist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}](recvMux)
}

//

type AdapterFactory struct {
}

func (f *AdapterFactory) CreateAdapter() (Adapter, error) {

	// TODO: create infra obj

	return Adapter{
		Recv: func(parent *Session) (*{{.RecvMessage}}, error) {
			return nil, nil
		},
		Send: func(message *{{.SendMessage}}) error {
			return nil
		},
		Stop: func(message *{{.SendMessage}}) {

		},
		Lifecycle: Artifex.Lifecycle[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]{
			SpawnHandlers: []func(sess *Session) error{},
			ExitHandlers:  []func(sess *Session){},
		},
		Identifier: "",
		Context:    context.Background(),
	}, nil
}
`
