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
		{{.Subject}}:  "",
		Metadata: make(map[string]any),
		Parent:   nil,
		Logger:   logger,
	}
}

type {{.FileName}}Ingress struct {
	MsgId string
	Body  []byte

	{{.Subject}}  {{.Subject}}
	Metadata maputil.Data
	Parent   any
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

		return message.{{.Subject}}
	}
	mux := Artifex.NewMux("/", get{{.Subject}})

	middleware := Artifex.MW[{{.FileName}}Ingress]{}
	mux.Middleware(middleware.Recover())
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

	middleware := Artifex.MW[{{.FileName}}Egress]{}
	mux.Middleware(middleware.Recover())
	return mux
}

func {{.FileName}}EgressSkip() {{.FileName}}EgressHandleFunc {
	return func(message *{{.FileName}}Egress, route *Artifex.RouteParam) (err error) {
		return nil
	}
}

`

const AdapterTmpl = `
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

type {{.FileName}}Factory struct {
	Hub             *Artifex.Hub[Artifex.IAdapter]
	Logger          Artifex.Logger
	SendPingSeconds int
	WaitPingSeconds int

	Authenticate   func() (name string, err error)
	AdapterName    string

	Container      DiContainer
}

func (f *{{.FileName}}Factory) CreatePubSub() (pubsub {{.FileName}}PubSub, err error) {
	id, err := f.Authenticate()
	if err != nil {
		return nil, err
	}

	ingressMux := f.Container.IngressMux()
	egressMux := f.Container.EgressMux()

	opt := Artifex.NewPubSubOption[{{.FileName}}Ingress, {{.FileName}}Egress]().
		Identifier(id).
		Logger(f.Logger).
		AdapterHub(f.Hub).
		DecorateAdapter(f.Container.DecorateAdapter).
		Lifecycle(f.Container.Lifecycle).
		HandleRecv(ingressMux.HandleMessage)

	var mu sync.Mutex
	waitNotify := make(chan error, 1)

	if f.SendPingSeconds > 0 {
		opt.SendPing(func() error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		}, waitNotify, f.SendPingSeconds*2)
	}

	if f.WaitPingSeconds > 0 {
		opt.WaitPing(waitNotify, f.WaitPingSeconds, func() error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		})
	}

	opt.AdapterRecv(func(adp Artifex.IAdapter) (*{{.FileName}}Ingress, error) {
		var err error
		defer func() {
			logger := adp.Log()
			if err != nil {
				logger.Error("recv : %v", err)
				return
			}
			logger.Info("recv %q", "")
		}()

		mu.Lock()
		defer mu.Unlock()

		return New{{.FileName}}Ingress(adp), nil
	})

	opt.AdapterSend(func(adp Artifex.IAdapter, message *{{.FileName}}Egress) (err error) {
		err = egressMux.HandleMessage(message, nil)

		defer func() {
			logger := adp.Log()
			if err != nil {
				logger.Error("send %q: %v", message.{{.Subject}}, err)
				return
			}
			logger.Info("send %q", message.{{.Subject}})
		}()

		mu.Lock()
		defer mu.Unlock()
		return 
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) (err error) {
		defer func() {
			logger := adp.Log()
			if err != nil {
				logger.Error("stop: %v", err)
				return
			}
			logger.Info("stop")
		}()

		mu.Lock()
		defer mu.Unlock()
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

	egressMux := f.Container.EgressMux()

	opt := Artifex.NewPublisherOption[{{.FileName}}Egress]().
		Identifier(f.AdapterName).
		Logger(f.Logger).
		AdapterHub(f.Hub).
		DecorateAdapter(f.Container.DecorateAdapter).
		Lifecycle(f.Container.Lifecycle)

	if f.SendPingSeconds > 0 {
		waitNotify := make(chan error, 1)
		opt.SendPing(func() error {

			return nil
		}, waitNotify, f.SendPingSeconds*2)
	}

	var mu sync.Mutex
	opt.AdapterSend(func(adp Artifex.IAdapter, message *{{.FileName}}Egress) (err error) {
		mu.Lock()
		defer mu.Unlock()

		err = egressMux.HandleMessage(message, nil)

		defer func() {
			logger := adp.Log()
			if err != nil {
				logger.Error("send %q: %v", message.{{.Subject}}, err)
				return
			}
			logger.Info("send %q", message.{{.Subject}})
		}()

		return 
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) (err error) {
		defer func() {
			logger := adp.Log()
			if err != nil {
				logger.Error("stop: %v", err)
				return
			}
			logger.Info("stop")
		}()

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

	ingressMux := f.Container.IngressMux()

	opt := Artifex.NewSubscriberOption[{{.FileName}}Ingress]().
		Identifier(f.AdapterName).
		Logger(f.Logger).
		AdapterHub(f.Hub).
		DecorateAdapter(f.Container.DecorateAdapter).
		Lifecycle(f.Container.Lifecycle).
		HandleRecv(ingressMux.HandleMessage)

	opt.AdapterRecv(func(adp Artifex.IAdapter) (*{{.FileName}}Ingress, error) {
		var err error
		defer func() {
			logger := adp.Log()
			if err != nil {
				logger.Error("recv: %v", err)
				return
			}
			logger.Info("recv %q", "")
		}()

		return New{{.FileName}}Ingress(adp), nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter) (err error) {
		defer func() {
			logger := adp.Log()
			if err != nil {
				logger.Error("stop: %v", err)
				return
			}
			logger.Info("stop")
		}()
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

const ContainerTmpl = `
package {{.Package}}

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

type Application struct {
	Artifex.IAdapter
}

func (app *Application) Initialize(ingressMux *{{.FileName}}IngressMux, egressMux *{{.FileName}}EgressMux) {

}

func (app *Application) Terminate() {

}

func NewAppContainer(newApp func() *Application) DiContainer {
	return &AppContainer{
		newApp:     newApp,
		ingressMux: NewIngressMux(),
		egressMux:  nil,
	}
}

type AppContainer struct {
	newApp func() *Application

	ingressMux *IngressMux

	mu        sync.Mutex
	egressMux *EgressMux
}

func (c *AppContainer) IngressMux() *IngressMux {
	mux := c.ingressMux

	return mux
}

func (c *AppContainer) EgressMux() *EgressMux {
	c.mu.Lock()
	mux := NewEgressMux()

	return mux
}

func (c *AppContainer) DecorateAdapter(adp Artifex.IAdapter) Artifex.IAdapter {
	app := c.newApp()
	app.IAdapter = adp
	return app
}

func (c *AppContainer) Lifecycle(lifecycle *Artifex.Lifecycle) {
	defer c.mu.Unlock()

	ingressMux := c.ingressMux
	egressMux := c.egressMux

	lifecycle.OnOpen(func(adp Artifex.IAdapter) error {
		adp.(*Application).Initialize(ingressMux, egressMux)
		return nil
	})

	lifecycle.OnStop(func(adp Artifex.IAdapter) {
		adp.(*Application).Terminate()
		return
	})
}

//

type DiContainer interface {
	IngressMux() *{{.FileName}}IngressMux
	EgressMux() *{{.FileName}}EgressMux
	DecorateAdapter(adp Artifex.IAdapter) (app Artifex.IAdapter)
	Lifecycle(lifecycle *Artifex.Lifecycle)
}

type PureFunction struct {
	IngressMux_f      func() *{{.FileName}}IngressMux
	EgressMux_f       func() *{{.FileName}}EgressMux
	DecorateAdapter_f func(adp Artifex.IAdapter) (app Artifex.IAdapter)
	Lifecycle_f       func(lifecycle *Artifex.Lifecycle)
}

func (c *PureFunction) IngressMux() *{{.FileName}}IngressMux {
	if c.IngressMux_f != nil {
		return c.IngressMux_f()
	}
	return nil
}

func (c *PureFunction) EgressMux() *{{.FileName}}EgressMux {
	if c.EgressMux_f != nil {
		return c.EgressMux_f()
	}
	return nil
}

func (c *PureFunction) DecorateAdapter(adp Artifex.IAdapter) (app Artifex.IAdapter) {
	if c.DecorateAdapter_f != nil {
		return c.DecorateAdapter_f(adp)
	}
	return adp
}

func (c *PureFunction) Lifecycle(lifecycle *Artifex.Lifecycle) {
	if c.Lifecycle_f != nil {
		c.Lifecycle_f(lifecycle)
	}
}

`
