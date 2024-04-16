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
	MsgId    string
	ByteBody []byte

	{{.Subject}} {{.Subject}}
	ParentInfra any
}

type {{.FileName}}IngressHandleFunc = Artifex.HandleFunc[{{.FileName}}Ingress]
type {{.FileName}}IngressMiddleware = Artifex.Middleware[{{.FileName}}Ingress]
type {{.FileName}}IngressMux = Artifex.Mux[{{.FileName}}Ingress]

func New{{.FileName}}IngressMux() *{{.FileName}}IngressMux {
	get{{.Subject}} := func(message *{{.FileName}}Ingress) string {
		// TODO
		return ""
	}

	mux := Artifex.NewMux("/", get{{.Subject}})
	return mux
}

func {{.FileName}}IngressSkip() {{.FileName}}IngressHandleFunc {
	return func(_ *{{.FileName}}Ingress, _ *Artifex.RouteParam) (err error) {
		return nil
	}
}

//

func New{{.FileName}}Egress() *{{.FileName}}Egress {
	return &{{.FileName}}Egress{}
}

type {{.FileName}}Egress struct {
	msgId      string
	ByteBody   []byte
	StringBody string

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
	get{{.Subject}} := func(message *{{.FileName}}Egress) string {
		// TODO
		return ""
	}

	mux := Artifex.NewMux("/", get{{.Subject}})
	return mux
}

func {{.FileName}}EgressSkip() {{.FileName}}EgressHandleFunc {
	return func(_ *{{.FileName}}Egress, _ *Artifex.RouteParam) (err error) {
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
	StopWithMessage(message *{{.FileName}}Egress) error
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
	return Artifex.NewHub(stop)
}

type {{.FileName}}PublisherHub = Artifex.Hub[{{.FileName}}Publisher]

func New{{.FileName}}PublisherHub() *{{.FileName}}PublisherHub {
	stop := func(publisher {{.FileName}}Publisher) {
		publisher.Stop()
	}
	return Artifex.NewHub(stop)
}

type {{.FileName}}SubscriberHub = Artifex.Hub[{{.FileName}}Subscriber]

func New{{.FileName}}SubscriberHub() *{{.FileName}}SubscriberHub {
	stop := func(subscriber {{.FileName}}Subscriber) {
		subscriber.Stop()
	}
	return Artifex.NewHub(stop)
}

//

type {{.FileName}}Factory struct {
	NewMux func() (ingressMux *{{.FileName}}IngressMux, egressMux *{{.FileName}}EgressMux)

	PubSubHub *{{.FileName}}PubSubHub
	PubHub    *{{.FileName}}PublisherHub
	SubHub    *{{.FileName}}SubscriberHub

	NewLifecycle func() (*Artifex.Lifecycle, error)
}

func (f *{{.FileName}}Factory) CreatePubSub() ({{.FileName}}PubSub, error) {

	ingressMux, egressMux := f.NewMux()

	opt := Artifex.NewPubSubOption[{{.FileName}}Ingress, {{.FileName}}Egress]().
		Identifier("").
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

	opt.AdapterSend(func(adp Artifex.IAdapter, egress *{{.FileName}}Egress) error {
		err := egressMux.HandleMessage(egress, nil)
		if err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter, egress *{{.FileName}}Egress) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	lifecycle, err := f.NewLifecycle()
	if err != nil {
		return nil, err
	}

	lifecycle.OnOpen(
		func(adp Artifex.IAdapter) error {
			err := f.PubSubHub.Join(adp.Identifier(), adp.({{.FileName}}PubSub))
			if err != nil {
				return err
			}
			lifecycle.OnStop(func(adp Artifex.IAdapter) {
				go f.PubSubHub.RemoveOne(func(pubsub {{.FileName}}PubSub) bool { return pubsub == adp })
			})
			return nil
		},
	)
	opt.NewLifecycle(func() *Artifex.Lifecycle { return lifecycle })

	return opt.BuildPubSub()
}

func (f *{{.FileName}}Factory) CreatePublisher() ({{.FileName}}Publisher, error) {

	_, egressMux := f.NewMux()

	waitNotify := make(chan error, 1)
	opt := Artifex.NewPublisherOption[{{.FileName}}Egress]().
		Identifier("").
		SendPing(func() error { return nil }, waitNotify, 30)

	var mu sync.Mutex
	opt.AdapterSend(func(adp Artifex.IAdapter, egress *{{.FileName}}Egress) error {
		mu.Lock()
		defer mu.Unlock()

		err := egressMux.HandleMessage(egress, nil)
		if err != nil {
			return err
		}
		return nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter, _ *{{.FileName}}Egress) error {
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		return nil
	})

	lifecycle, err := f.NewLifecycle()
	if err != nil {
		return nil, err
	}

	lifecycle.OnOpen(
		func(adp Artifex.IAdapter) error {
			err := f.PubHub.Join(adp.Identifier(), adp.({{.FileName}}Publisher))
			if err != nil {
				return err
			}
			lifecycle.OnStop(func(adp Artifex.IAdapter) {
				go f.PubHub.RemoveOne(func(pub {{.FileName}}Publisher) bool { return pub == adp })
			})
			return nil
		},
	)
	opt.NewLifecycle(func() *Artifex.Lifecycle { return lifecycle })

	return opt.BuildPublisher()
}

func (f *{{.FileName}}Factory) CreateSubscriber() ({{.FileName}}Subscriber, error) {

	ingressMux, _ := f.NewMux()

	waitNotify := make(chan error, 1)
	opt := Artifex.NewSubscriberOption[{{.FileName}}Ingress]().
		Identifier("").
		HandleRecv(ingressMux.HandleMessage).
		SendPing(func() error { return nil }, waitNotify, 30)

	opt.AdapterRecv(func(adp Artifex.IAdapter) (*{{.FileName}}Ingress, error) {
		return New{{.FileName}}Ingress(), nil
	})

	opt.AdapterStop(func(adp Artifex.IAdapter, _ *struct{}) error {
		return nil
	})

	opt.AdapterFixup(0, func(adp Artifex.IAdapter) error {
		return nil
	})

	lifecycle, err := f.NewLifecycle()
	if err != nil {
		return nil, err
	}

	lifecycle.OnOpen(
		func(adp Artifex.IAdapter) error {
			err := f.SubHub.Join(adp.Identifier(), adp.({{.FileName}}Subscriber))
			if err != nil {
				return err
			}
			lifecycle.OnStop(func(adp Artifex.IAdapter) {
				go f.SubHub.RemoveOne(func(sub {{.FileName}}Subscriber) bool { return sub == adp })
			})
			return nil
		},
	)
	opt.NewLifecycle(func() *Artifex.Lifecycle { return lifecycle })

	return opt.BuildSubscriber()
}
`
