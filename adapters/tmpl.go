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

type ConsumeHandleFunc = Artifex.HandleFunc[*{{.RecvMessage}}]
type ConsumeMiddleware = Artifex.Middleware[*{{.RecvMessage}}]
type ConsumeMux = Artifex.Mux[{{.Subject}}, *{{.RecvMessage}}]

func NewConsumeMux() *ConsumeMux {
	get{{.Subject}} := func(message *{{.RecvMessage}}) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("hello", HelloHandler())
	return mux
}

func HelloHandler() ConsumeHandleFunc {
	return func(message *{{.RecvMessage}}, route *Artifex.RouteParam) error {
		return nil
	}
}

//

type ProduceHandleFunc = Artifex.HandleFunc[*{{.SendMessage}}]
type ProduceMiddleware = Artifex.Middleware[*{{.SendMessage}}]
type ProduceMux = Artifex.Mux[{{.Subject}}, *{{.SendMessage}}]

func NewProduceMux() *ProduceMux {
	get{{.Subject}} := func(message *{{.SendMessage}}) (string, error) {
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("world", WorldHandler())
	return mux
}

func WorldHandler() ProduceHandleFunc {
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

func NewSession(mux *ConsumeMux, factory *AdapterFactory) (*Session, error) {
	return Artifex.NewSession[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}](mux, factory)
}

//

type Hub = Artifex.Artist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

func NewHub(mux *ConsumeMux) *Hub {
	return Artifex.NewArtist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}](mux)
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
