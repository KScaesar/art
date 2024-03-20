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

type Hub = Artifex.Artist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

func NewHub() *Hub {
	// TODO: option
	return Artifex.NewArtist[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]()
}

type Session = Artifex.Session[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]

type SessionFactory struct {
}

func (f *SessionFactory) CreateSession() (*Session, error) {

	// TODO: create infra obj

	sess := &Session{
		Identifier: "",
		Context:    context.Background(),
		Pingpong:   NewPigPong(),
		Lifecycle: Artifex.Lifecycle[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]{
			SpawnHandlers: []func(sess *Session) error{},
			ExitHandlers:  []func(sess *Session){},
		},
	}

	// TODO: must
	sess.Mux = nil
	recv := func() (*{{.RecvMessage}}, error) {
		return nil, nil
	}
	send := func(message *{{.SendMessage}}) error {
		return nil
	}
	stop := func(message *{{.SendMessage}}) {
		
	}

	sess.AdapterRecv = recv
	sess.AdapterSend = send
	sess.AdapterStop = stop
	return sess, nil
}

func NewPigPong() Artifex.PingPong[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}] {
	return Artifex.PingPong[{{.Subject}}, *{{.RecvMessage}}, *{{.SendMessage}}]{
		Enable:     false,
		WaitSecond: 0,
		// WaitSubject: "PingPong{{.Subject}}",
		SendFunc: func(session *Session) error {
			var pingpong *{{.SendMessage}}
			
			return session.Send(pingpong)
		},
		IsSendPingWaitPong: false,
		WaitFunc: func(message *{{.RecvMessage}}, route *Artifex.RouteParam) error {
			return nil
		},
	}
}
`
