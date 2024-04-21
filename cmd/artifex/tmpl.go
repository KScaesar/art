package main

const MsgTmpl = `
package {{.Package}}

import (
	"context"

	"github.com/gookit/goutil/maputil"

	"github.com/KScaesar/Artifex"
)

type {{.Subject}} = string

//

func New{{.FileName}}Ingress() *{{.FileName}}Ingress {
	return &{{.FileName}}Ingress{
		Metadata: make(map[string]any),
	}
}

type {{.FileName}}Ingress struct {
	MsgId string
	Body  []byte

	Subject     string
	Metadata    maputil.Data
	ParentInfra any

	ctx context.Context
}

func (in *{{.FileName}}Ingress) Context() context.Context {
	if in.ctx == nil {
		in.ctx = context.Background()
	}
	return in.ctx
}

func (in *{{.FileName}}Ingress) SetContext(ctx context.Context) {
	in.ctx = ctx
}

type {{.FileName}}IngressHandleFunc = Artifex.HandleFunc[{{.FileName}}Ingress]
type {{.FileName}}IngressMiddleware = Artifex.Middleware[{{.FileName}}Ingress]
type {{.FileName}}IngressMux = Artifex.Mux[{{.FileName}}Ingress]

func New{{.FileName}}IngressMux() *{{.FileName}}IngressMux {
	get{{.Subject}} := func(message *{{.FileName}}Ingress) string {

		return message.Subject
	}

	mux := Artifex.NewMux("/", get{{.Subject}})
	return mux
}

func {{.FileName}}IngressSkip() {{.FileName}}IngressHandleFunc {
	return func(message *{{.FileName}}Ingress, route *Artifex.RouteParam) (err error) {
		return nil
	}
}

//

func New{{.FileName}}Egress(subject {{.Subject}}, message any) *{{.FileName}}Egress {
	return &{{.FileName}}Egress{
		Subject:  subject,
		Metadata: make(map[string]any),
		AppMsg:   message,
	}
}

type {{.FileName}}Egress struct {
	msgId string
	Body  []byte

	Subject  string
	Metadata maputil.Data
	AppMsg   any

	ctx context.Context
}

func (e *{{.FileName}}Egress) MsgId() string {
	if e.msgId == "" {
		e.msgId = Artifex.GenerateUlid()
	}
	return e.msgId
}

func (e *{{.FileName}}Egress) SetMsgId(msgId string) {
	e.msgId = msgId
}

func (e *{{.FileName}}Egress) Context() context.Context {
	if e.ctx == nil {
		e.ctx = context.Background()
	}
	return e.ctx
}

func (e *{{.FileName}}Egress) SetContext(ctx context.Context) {
	e.ctx = ctx
}

type {{.FileName}}EgressHandleFunc = Artifex.HandleFunc[{{.FileName}}Egress]
type {{.FileName}}EgressMiddleware = Artifex.Middleware[{{.FileName}}Egress]
type {{.FileName}}EgressMux = Artifex.Mux[{{.FileName}}Egress]

func New{{.FileName}}EgressMux() *{{.FileName}}EgressMux {
	get{{.Subject}} := func(message *{{.FileName}}Egress) string {

		return message.Subject
	}

	mux := Artifex.NewMux("/", get{{.Subject}})
	return mux
}

func {{.FileName}}EgressSkip() {{.FileName}}EgressHandleFunc {
	return func(message *{{.FileName}}Egress, route *Artifex.RouteParam) (err error) {
		return nil
	}
}

`

const PubSubTmpl = `
package {{.Package}}

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

//

type {{.FileName}}PubSub interface {
	Artifex.IAdapter
	Send(messages ...*{{.FileName}}Egress) error
	Listen() (err error)
}

type {{.FileName}}Publisher interface {
	Artifex.IAdapter
	Send(messages ...*{{.FileName}}Egress) error
}

type {{.FileName}}Subscriber interface {
	Artifex.IAdapter
	Listen() (err error)
}

//

type {{.FileName}}PubSubHub = Artifex.Hub[{{.FileName}}PubSub]

func New{{.FileName}}PubSubHub() *{{.FileName}}PubSubHub {
	stop := func(pubsub {{.FileName}}PubSub) {
		pubsub.Stop()
	}
	hub := Artifex.NewHub(stop)
	return hub
}

type {{.FileName}}PubHub = Artifex.Hub[{{.FileName}}Publisher]

func New{{.FileName}}PubHub() *{{.FileName}}PubHub {
	stop := func(publisher {{.FileName}}Publisher) {
		publisher.Stop()
	}
	hub := Artifex.NewHub(stop)
	return hub
}

type {{.FileName}}SubHub = Artifex.Hub[{{.FileName}}Subscriber]

func New{{.FileName}}SubHub() *{{.FileName}}SubHub {
	stop := func(subscriber {{.FileName}}Subscriber) {
		subscriber.Stop()
	}
	hub := Artifex.NewHub(stop)
	return hub
}

//

type {{.FileName}}Factory struct {
	NewMux    func() (*{{.FileName}}IngressMux, *{{.FileName}}EgressMux)
	PubSubHub *{{.FileName}}PubSubHub

	NewIngressMux func() *{{.FileName}}IngressMux
	SubHub        *{{.FileName}}SubHub

	NewEgressMux func() *{{.FileName}}EgressMux
	PubHub       *{{.FileName}}PubHub

	DecorateAdapter func(adp Artifex.IAdapter) (app Artifex.IAdapter)
	Lifecycle       func(lifecycle *Artifex.Lifecycle)
}

func (f *{{.FileName}}Factory) CreatePubSub() (pubsub {{.FileName}}PubSub, err error) {

	ingressMux, egressMux := f.NewMux()

	opt := Artifex.NewPubSubOption[{{.FileName}}Ingress, {{.FileName}}Egress]().
		Identifier("").
		DecorateAdapter(f.DecorateAdapter).
		HandleRecv(ingressMux.HandleMessage)

	var mu sync.Mutex
	waitNotify := make(chan error, 1)

	opt.SendPing(func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	}, waitNotify, 30)

	opt.WaitPing(waitNotify, 30, func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.AdapterRecv(func(adp Artifex.IAdapter) (*{{.FileName}}Ingress, error) {
		parent := adp.({{.FileName}}PubSub)
		_ = parent
		return New{{.FileName}}Ingress(), nil
	})

	opt.AdapterSend(func(adp Artifex.IAdapter, message *{{.FileName}}Egress) error {
		err := egressMux.HandleMessage(message, nil)
		if err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.Lifecycle(func(life *Artifex.Lifecycle) {
		life.OnOpen(func(adp Artifex.IAdapter) error {
			err := f.PubSubHub.Join(adp.Identifier(), adp.({{.FileName}}PubSub))
			if err != nil {
				return err
			}
			life.OnStop(func(adp Artifex.IAdapter) {
				go f.PubSubHub.RemoveOne(func(pubsub {{.FileName}}PubSub) bool { return pubsub == adp })
			})
			return nil
		})
		if f.Lifecycle != nil {
			f.Lifecycle(life)
		}
	})

	adp, err := opt.Build()
	if err != nil {
		return
	}
	return adp.({{.FileName}}PubSub), err
}

func (f *{{.FileName}}Factory) CreatePublisher() (pub {{.FileName}}Publisher, err error) {

	egressMux := f.NewEgressMux()

	waitNotify := make(chan error, 1)
	opt := Artifex.NewPublisherOption[{{.FileName}}Egress]().
		Identifier("").
		DecorateAdapter(f.DecorateAdapter).
		SendPing(func() error { return nil }, waitNotify, 30)

	var mu sync.Mutex
	opt.AdapterSend(func(adp Artifex.IAdapter, message *{{.FileName}}Egress) error {
		mu.Lock()
		defer mu.Unlock()

		err := egressMux.HandleMessage(message, nil)
		if err != nil {
			return err
		}
		return nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) error {
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		return nil
	})

	opt.Lifecycle(func(life *Artifex.Lifecycle) {
		life.OnOpen(func(adp Artifex.IAdapter) error {
			err := f.PubHub.Join(adp.Identifier(), adp.({{.FileName}}Publisher))
			if err != nil {
				return err
			}
			life.OnStop(func(adp Artifex.IAdapter) {
				go f.PubHub.RemoveOne(func(pub {{.FileName}}Publisher) bool { return pub == adp })
			})
			return nil
		})
		if f.Lifecycle != nil {
			f.Lifecycle(life)
		}
	})

	adp, err := opt.Build()
	if err != nil {
		return
	}
	return adp.({{.FileName}}Publisher), err
}

func (f *{{.FileName}}Factory) CreateSubscriber() (sub {{.FileName}}Subscriber, err error) {

	ingressMux := f.NewIngressMux()

	waitNotify := make(chan error, 1)
	opt := Artifex.NewSubscriberOption[{{.FileName}}Ingress]().
		Identifier("").
		DecorateAdapter(f.DecorateAdapter).
		HandleRecv(ingressMux.HandleMessage).
		SendPing(func() error { return nil }, waitNotify, 30)

	opt.AdapterRecv(func(adp Artifex.IAdapter) (*{{.FileName}}Ingress, error) {
		return New{{.FileName}}Ingress(), nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) error {
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		return nil
	})

	opt.Lifecycle(func(life *Artifex.Lifecycle) {
		life.OnOpen(func(adp Artifex.IAdapter) error {
			err := f.SubHub.Join(adp.Identifier(), adp.({{.FileName}}Subscriber))
			if err != nil {
				return err
			}
			life.OnStop(func(adp Artifex.IAdapter) {
				go f.SubHub.RemoveOne(func(sub {{.FileName}}Subscriber) bool { return sub == adp })
			})
			return nil
		})
		if f.Lifecycle != nil {
			f.Lifecycle(life)
		}
	})

	adp, err := opt.Build()
	if err != nil {
		return
	}
	return adp.({{.FileName}}Subscriber), err
}
`
