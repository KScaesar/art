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

func New{{.FileName}}Ingress(adp Artifex.IAdapter) *{{.FileName}}Ingress {
	msgId := Artifex.GenerateUlid()

	logger := adp.Log().WithKeyValue("msg_id", msgId)

	return &{{.FileName}}Ingress{
		MsgId:    msgId,
		Metadata: make(map[string]any),
		Logger:   logger,
	}
}

type {{.FileName}}Ingress struct {
	MsgId string
	Body  []byte

	Subject  string
	Metadata maputil.Data
	Logger   Artifex.Logger

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

func NewAdapterHub() *Artifex.Hub[Artifex.IAdapter] {
	hub := Artifex.NewAdapterHub(func(adp Artifex.IAdapter) {
		adp.Stop()
	})
	return hub
}

type {{.FileName}}Factory struct {
	NewAdapterId func() (string, error)

	NewMux        func() (*{{.FileName}}IngressMux, *{{.FileName}}EgressMux)
	NewIngressMux func() *{{.FileName}}IngressMux
	NewEgressMux  func() *{{.FileName}}EgressMux
	Hub           *Artifex.Hub[Artifex.IAdapter]
	Logger        Artifex.Logger

	SendPingSeconds int
	WaitPingSeconds int
	DecorateAdapter func(adp Artifex.IAdapter) (app Artifex.IAdapter)
	Lifecycle       func(lifecycle *Artifex.Lifecycle)
}

func (f *{{.FileName}}Factory) CreatePubSub() (pubsub {{.FileName}}PubSub, err error) {
	id, err := f.NewAdapterId()
	if err != nil {
		return nil, err
	}

	ingressMux, egressMux := f.NewMux()

	opt := Artifex.NewPubSubOption[{{.FileName}}Ingress, {{.FileName}}Egress]().
		Identifier(id).
		Logger(f.Logger).
		AdapterHub(f.Hub).
		DecorateAdapter(f.DecorateAdapter).
		Lifecycle(f.Lifecycle).
		HandleRecv(ingressMux.HandleMessage)

	var mu sync.Mutex
	waitNotify := make(chan error, 1)

	opt.SendPing(func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	}, waitNotify, f.SendPingSeconds*2)

	opt.WaitPing(waitNotify, f.WaitPingSeconds, func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.AdapterRecv(func(adp Artifex.IAdapter) (*{{.FileName}}Ingress, error) {
		return New{{.FileName}}Ingress(adp), nil
	})

	opt.AdapterSend(func(adp Artifex.IAdapter, message *{{.FileName}}Egress) error {
		mu.Lock()
		defer mu.Unlock()

		err := egressMux.HandleMessage(message, nil)
		logger := adp.Log().WithKeyValue("msg_id", message.MsgId())
		if err != nil {
			logger.Error("send %q: %v", message.Subject, err)
			return err
		}
		logger.Info("send %q", message.Subject)
		return nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) error {
		mu.Lock()
		defer mu.Unlock()

		var err error
		if err != nil {
			adp.Log().Error("stop: %v", err)
			return err
		}
		adp.Log().Info("stop")
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	adp, err := opt.Build()
	if err != nil {
		return
	}
	return adp.({{.FileName}}PubSub), err
}

func (f *{{.FileName}}Factory) CreatePublisher() (pub {{.FileName}}Publisher, err error) {
	id, err := f.NewAdapterId()
	if err != nil {
		return nil, err
	}

	egressMux := f.NewEgressMux()

	waitNotify := make(chan error, 1)
	opt := Artifex.NewPublisherOption[{{.FileName}}Egress]().
		Identifier(id).
		Logger(f.Logger).
		AdapterHub(f.Hub).
		DecorateAdapter(f.DecorateAdapter).
		Lifecycle(f.Lifecycle).
		SendPing(func() error { return nil }, waitNotify, f.SendPingSeconds*2)

	var mu sync.Mutex
	opt.AdapterSend(func(adp Artifex.IAdapter, message *{{.FileName}}Egress) error {
		mu.Lock()
		defer mu.Unlock()

		err := egressMux.HandleMessage(message, nil)
		logger := adp.Log().WithKeyValue("msg_id", message.MsgId())
		if err != nil {
			logger.Error("send %q: %v", message.Subject, err)
			return err
		}
		logger.Info("send %q", message.Subject)
		return nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) error {
		mu.Lock()
		defer mu.Unlock()

		var err error
		if err != nil {
			adp.Log().Error("stop: %v", err)
			return err
		}
		adp.Log().Info("stop")
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		return nil
	})

	adp, err := opt.Build()
	if err != nil {
		return
	}
	return adp.({{.FileName}}Publisher), err
}

func (f *{{.FileName}}Factory) CreateSubscriber() (sub {{.FileName}}Subscriber, err error) {
	id, err := f.NewAdapterId()
	if err != nil {
		return nil, err
	}

	ingressMux := f.NewIngressMux()

	waitNotify := make(chan error, 1)
	opt := Artifex.NewSubscriberOption[{{.FileName}}Ingress]().
		Identifier(id).
		Logger(f.Logger).
		AdapterHub(f.Hub).
		DecorateAdapter(f.DecorateAdapter).
		Lifecycle(f.Lifecycle).
		HandleRecv(ingressMux.HandleMessage).
		SendPing(func() error { return nil }, waitNotify, f.SendPingSeconds*2)

	opt.AdapterRecv(func(adp Artifex.IAdapter) (*{{.FileName}}Ingress, error) {
		return New{{.FileName}}Ingress(adp), nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) error {

		var err error
		if err != nil {
			adp.Log().Error("stop: %v", err)
			return err
		}
		adp.Log().Info("stop")
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		return nil
	})

	adp, err := opt.Build()
	if err != nil {
		return
	}
	return adp.({{.FileName}}Subscriber), err
}
`
