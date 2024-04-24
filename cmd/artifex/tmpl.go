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

func New{{.FileName}}Ingress(logger Artifex.Logger) *{{.FileName}}Ingress {

	return &{{.FileName}}Ingress{
		{{.Subject}}:  "",
		Metadata: make(map[string]any),
		Logger: logger,
	}
}

type {{.FileName}}Ingress struct {
	msgId string
	Body  []byte

	{{.Subject}}  {{.Subject}}
	Metadata maputil.Data
	Logger   Artifex.Logger

	ctx context.Context
}

func (in *{{.FileName}}Ingress) MsgId() string {
	if in.msgId == "" {
		in.msgId = Artifex.GenerateUlid()
	}
	return in.msgId
}

func (in *{{.FileName}}Ingress) SetMsgId(msgId string) {
	in.msgId = msgId
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
	mux.SetHandleError(middleware.PrintError(get{{.Subject}}))
	return mux
}

func {{.FileName}}IngressSkip() {{.FileName}}IngressHandleFunc {
	return func(dep any, message *{{.FileName}}Ingress, route *Artifex.RouteParam) (err error) {
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
	mux.SetHandleError(middleware.PrintError(get{{.Subject}}))
	return mux
}

func {{.FileName}}EgressSkip() {{.FileName}}EgressHandleFunc {
	return func(dep any, message *{{.FileName}}Egress, route *Artifex.RouteParam) (err error) {
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

	Authenticate func() (name string, err error)
	AdapterName  string

	IngressMux      func() *IngressMux
	EgressMux       func() *EgressMux
	DecorateAdapter func(adapter Artifex.IAdapter) (application Artifex.IAdapter)
	Lifecycle       func(lifecycle *Artifex.Lifecycle)
}

func (f *{{.FileName}}Factory) CreateAdapter() (adapter Artifex.IAdapter, err error) {
	name, err := f.Authenticate()
	if err != nil {
		return nil, err
	}

	opt := Artifex.NewPubSubOption[{{.FileName}}Ingress, {{.FileName}}Egress]().
		Identifier(name).
		AdapterHub(f.Hub).
		Logger(f.Logger).
		DecorateAdapter(f.DecorateAdapter).
		Lifecycle(f.Lifecycle)

	ingressMux := f.IngressMux()
	egressMux := f.EgressMux()
	var mu sync.Mutex

	if f.SendPingSeconds > 0 {
		waitPong := make(chan error, 1)
		sendPing := func(adp Artifex.IAdapter) error {
			adp.(Artifex.IAdapter).Log().Info("send ping")

			return nil
		}
		opt.SendPing(sendPing, waitPong, f.SendPingSeconds*2)
	}

	if f.WaitPingSeconds > 0 {
		waitPing := make(chan error, 1)
		sendPong := func(adp Artifex.IAdapter) error {
			adp.(Artifex.IAdapter).Log().Info("send pong")

			return nil
		}
		opt.WaitPing(waitPing, f.WaitPingSeconds, sendPong)
	}

	opt.IngressMux(ingressMux)
	opt.AdapterRecv(func(logger Artifex.Logger) (*{{.FileName}}Ingress, error) {

		var err error
		if err != nil {
			logger.Error("recv fail: %v", err)
			return nil, err
		}
		logger.Info("recv ok")
		return New{{.FileName}}Ingress(logger), nil
	})

	opt.EgressMux(egressMux)
	opt.AdapterSend(func(logger Artifex.Logger, message *{{.FileName}}Egress) (err error) {
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			logger.Error("send %q fail: %v", message.Subject, err)
			return
		}
		logger.Info("send %q ok", message.Subject)
		return 
	})

	opt.AdapterStop(func(logger Artifex.Logger) (err error) {
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			logger.Error("stop fail: %v", err)
			return
		}
		logger.Info("stop ok")
		return nil
	})

	retry := 0
	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		mu.Lock()
		defer mu.Unlock()
		logger := adp.Log()

		retry++
		logger.Info("retry %v times start", retry)
		if err != nil {
			logger.Error("retry fail: %v", err)
			return err
		}
		retry = 0
		logger.Info("retry ok")
		return nil
	})

	adp, err := opt.Build()
	if err != nil {
		return
	}
	return adp, err
}
`
