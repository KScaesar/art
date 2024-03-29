package main

const MsgTmpl = `
package {{.Package}}

import (
	"github.com/gookit/goutil/maputil"

	"github.com/KScaesar/Artifex"
)

type {{.Subject}} = string

//

func New{{.FileName}}Ingress() *{{.FileName}}Ingress {
	return &{{.FileName}}Ingress{}
}

type {{.FileName}}Ingress struct {
	MsgId       string
	IngressMsg []byte

	{{.Subject}} {{.Subject}}
	ParentInfra any
}

type {{.FileName}}IngressHandleFunc = Artifex.HandleFunc[{{.FileName}}Ingress]
type {{.FileName}}IngressMiddleware = Artifex.Middleware[{{.FileName}}Ingress]
type {{.FileName}}IngressMux = Artifex.Mux[{{.FileName}}Ingress]

func New{{.FileName}}IngressMux() *{{.FileName}}IngressMux {
	get{{.Subject}} := func(message *{{.FileName}}Ingress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux("/", get{{.Subject}})
	mux.Handler("ingress", {{.FileName}}IngressHandler())
	return mux
}

func {{.FileName}}IngressHandler() {{.FileName}}IngressHandleFunc {
	return func(message *{{.FileName}}Ingress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func {{.FileName}}IngressHandleError(message *{{.FileName}}Ingress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}

//

func New{{.FileName}}Egress() *{{.FileName}}Egress {
	return &{{.FileName}}Egress{}
}

type {{.FileName}}Egress struct {
	msgId     string
	EgressMsg []byte

	{{.Subject}} {{.Subject}}
	Metadata maputil.Data
	AppMsg   any
}

func (e *{{.FileName}}Egress) MsgId() string {
	if e.msgId == "" {
		return ""
	}
	return e.msgId
}

func (e *{{.FileName}}Egress) SetMsgId(msgId string) {
	e.msgId = msgId
}

type {{.FileName}}EgressHandleFunc = Artifex.HandleFunc[{{.FileName}}Egress]
type {{.FileName}}EgressMiddleware = Artifex.Middleware[{{.FileName}}Egress]
type {{.FileName}}EgressMux = Artifex.Mux[{{.FileName}}Egress]

func New{{.FileName}}EgressMux() *{{.FileName}}EgressMux {
	get{{.Subject}} := func(message *{{.FileName}}Egress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux("/", get{{.Subject}})
	mux.Handler("egress", {{.FileName}}EgressHandler())
	return mux
}

func {{.FileName}}EgressHandler() {{.FileName}}EgressHandleFunc {
	return func(message *{{.FileName}}Egress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func {{.FileName}}EgressHandleError(message *{{.FileName}}Ingress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}
`

const PubSubTmpl = `
package {{.Package}}

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

func Build{{.FileName}}Infrastructure() (any, error) {
	return nil, nil
}

func New{{.FileName}}PingPong() Artifex.PingPong {
	waitNotify := make(chan error, 1)

	return Artifex.PingPong{
		IsSendPingWaitPong: true,
		SendFunc: func() error {
			return nil
		},
		WaitNotify: waitNotify,
		WaitSecond: 30,
	}
}

type {{.FileName}}Factory struct {
	NewMux      func() (ingressMux *{{.FileName}}IngressMux, egressMux *{{.FileName}}EgressMux)
}

//

type {{.FileName}}PubSub    = Artifex.PubSub[{{.FileName}}Ingress, {{.FileName}}Egress]
type {{.FileName}}PubSubHub = Artifex.Hub[*{{.FileName}}PubSub]

func New{{.FileName}}PubSubHub() *{{.FileName}}PubSubHub {
	stop := func(adapter *{{.FileName}}PubSub) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

func (f *{{.FileName}}Factory) CreatePubSub() (*{{.FileName}}PubSub, error) {
	var mu sync.Mutex
	ingressMux, egressMux := f.NewMux()

	pubsub := &{{.FileName}}PubSub{
		HandleRecv: ingressMux.HandleMessage,
		Identifier: "",
	}

	pubsub.AdapterRecv = func() (*{{.FileName}}Ingress, error) {
		return New{{.FileName}}Ingress(), nil
	}

	pubsub.AdapterSend = func(message *{{.FileName}}Egress) error {
		err := egressMux.HandleMessage(message, nil)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		return nil
	}

	pubsub.AdapterStop = func(message *{{.FileName}}Egress) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	}

	pubsub.FixupMaxRetrySecond = 0
	pubsub.Fixup = func() error {
		return nil
	}

	life := Artifex.Lifecycle{}
	pubsub.Lifecycle = life

	pp := New{{.FileName}}PingPong()
	go func() {
		err := pubsub.PingPong(pp)
		if err != nil {
			_ = err
		}
	}()

	return pubsub, nil
}

//

type {{.FileName}}Publisher = Artifex.Publisher[{{.FileName}}Egress]
type {{.FileName}}PublisherHub = Artifex.Hub[*{{.FileName}}Publisher]

func New{{.FileName}}PublisherHub() *{{.FileName}}PublisherHub {
	stop := func(adapter *{{.FileName}}Publisher) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

func (f *{{.FileName}}Factory) CreatePublisher() (*{{.FileName}}Publisher, error) {
	_, egressMux := f.NewMux()

	pub := &{{.FileName}}Publisher{
		Identifier: "",
	}

	pub.AdapterSend = func(message *{{.FileName}}Egress) error {
		err := egressMux.HandleMessage(message, nil)
		if err != nil {
			return err
		}
		return nil
	}

	pub.AdapterStop = func() error {
		return nil
	}

	pub.FixupMaxRetrySecond = 0
	pub.Fixup = func() error {
		return nil
	}

	life := Artifex.Lifecycle{}
	pub.Lifecycle = life

	return pub, nil
}

//

type {{.FileName}}Subscriber = Artifex.Subscriber[{{.FileName}}Ingress]
type {{.FileName}}SubscriberHub = Artifex.Hub[*{{.FileName}}Subscriber]

func New{{.FileName}}SubscriberHub() *{{.FileName}}SubscriberHub {
	stop := func(adapter *{{.FileName}}Subscriber) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

func (f *{{.FileName}}Factory) CreateSubscriber() (*{{.FileName}}Subscriber, error) {
	ingressMux, _ := f.NewMux()

	sub := &{{.FileName}}Subscriber{
		HandleRecv: ingressMux.HandleMessage,
		Identifier: "",
	}

	sub.AdapterRecv = func() (*{{.FileName}}Ingress, error) {
		return New{{.FileName}}Ingress(), nil
	}

	sub.AdapterStop = func() error {
		return nil
	}

	sub.FixupMaxRetrySecond = 0
	sub.Fixup = func() error {
		return nil
	}

	life := Artifex.Lifecycle{}
	sub.Lifecycle = life

	return sub, nil
}
`
