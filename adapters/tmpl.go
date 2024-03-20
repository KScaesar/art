package main

const MuxTmpl = `
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

type ConsumeHandleFunc = Artifex.HandleFunc[*{{.RecvMessage}}]
type ConsumeMiddleware = Artifex.Middleware[*{{.RecvMessage}}]
type ConsumeMux = Artifex.Mux[{{.Subject}}, *{{.RecvMessage}}]

func NewConsumeMux() *ConsumeMux {
	get{{.Subject}} := func(message *{{.RecvMessage}}) (string, error) {
		// TODO: must
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("hello", HelloHandler())
	return mux
}

// Example
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
		// TODO: must
		return "", nil
	}

	mux := Artifex.NewMux[{{.Subject}}](get{{.Subject}})
	mux.Handler("world", WorldHandler())
	return mux
}

// Example
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

func NewSession(mux *ConsumeMux, factory Artifex.AdapterFactory[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]) (*Session, error) {
	// TODO: option
	return Artifex.NewSession[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}](mux, factory)
}

//

type Hub = Artifex.Artist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

func NewHub(mux *ConsumeMux) *Hub {
	// TODO: option
	return Artifex.NewArtist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}](mux)
}

//

type AdapterFactory struct {
}

func (f *AdapterFactory) CreateAdapter() (Adapter, error) {

	// TODO: create infra obj

	return Adapter{
		Recv: func(parent *Session) (*{{.RecvMessage}}, error) {
			// TODO: must
			return nil, nil
		},
		Send: func(message *{{.SendMessage}}) error {
			// TODO: must
			return nil
		},
		Stop: func(message *{{.SendMessage}}) {
			// TODO: must
		},
		Lifecycle: Artifex.Lifecycle[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]{
			SpawnHandlers: []func(sess *Session) error{},
			ExitHandlers:  []func(sess *Session){},
		},
		Identifier: "",
		Context:    context.Background(),
		PingPong: Artifex.PingPong[Channel, *{{.RecvMessage}}, *{{.SendMessage}}]{
			Enable:     false,
			WaitSecond: 15,
			// WaitSubject: "PingPong{{.Subject}}",
			SendFunc: func(sess *Session) error {
				var pingpong *{{.SendMessage}}
				return sess.Send(pingpong)
			},
			IsSendPingWaitPong: false,
			WaitFunc: func(_ *{{.RecvMessage}}, _ *Artifex.RouteParam) error {
				return nil
			},
		},
	}, nil
}
`
