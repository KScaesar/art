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

type {{.FileName}}Factory struct {
	NewMux func() (ingressMux *{{.FileName}}IngressMux, egressMux *{{.FileName}}EgressMux)

	PubSubHub *{{.FileName}}PubSubHub
	PubHub    *{{.FileName}}PublisherHub
	SubHub    *{{.FileName}}SubscriberHub
}

//

type {{.FileName}}PubSub = Artifex.Adapter[{{.FileName}}Ingress, {{.FileName}}Egress]
type {{.FileName}}PubSubHub = Artifex.Hub[*{{.FileName}}PubSub]

func New{{.FileName}}PubSubHub() *{{.FileName}}PubSubHub {
	stop := func(pubsub *{{.FileName}}PubSub) error {
		return pubsub.Stop()
	}
	return Artifex.NewHub(stop)
}

func (f *{{.FileName}}Factory) CreatePubSub() (*{{.FileName}}PubSub, error) {

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

	opt.WaitPing(func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	}, waitNotify, 30)

	opt.AdapterRecv(func(adp *{{.FileName}}PubSub) (*{{.FileName}}Ingress, error) {
		return New{{.FileName}}Ingress(), nil
	})

	opt.AdapterSend(func(adp *{{.FileName}}PubSub, egress *{{.FileName}}Egress) error {
		err := egressMux.HandleMessage(egress, nil)
		if err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.AdapterStop(func(adp *{{.FileName}}PubSub, egress *{{.FileName}}Egress) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	opt.AdapterFixup(0, func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	pubsub := Artifex.NewPubSub(opt)
	pubsub.
		AddSpawnHandler(func() error {
			err := f.PubSubHub.Join(pubsub.Identifier(), pubsub)
			if err != nil {
				return err
			}
			return nil
		}).
		AddExitHandler(func() error {
			f.PubSubHub.RemoveByKey(pubsub.Identifier())
			return nil
		})

	return pubsub, nil
}

//

type {{.FileName}}Publisher = Artifex.Adapter[struct{}, {{.FileName}}Egress]
type {{.FileName}}PublisherHub = Artifex.Hub[*{{.FileName}}Publisher]

func New{{.FileName}}PublisherHub() *{{.FileName}}PublisherHub {
	stop := func(publisher *{{.FileName}}Publisher) error {
		return publisher.Stop()
	}
	return Artifex.NewHub(stop)
}

func (f *{{.FileName}}Factory) CreatePublisher() (*{{.FileName}}Publisher, error) {

	_, egressMux := f.NewMux()

	waitNotify := make(chan error, 1)
	opt := Artifex.NewPublisherOption[{{.FileName}}Egress]().
		Identifier("").
		SendPing(func() error { return nil }, waitNotify, 30)

	opt.AdapterSend(func(adp *{{.FileName}}Publisher, egress *{{.FileName}}Egress) error {
		err := egressMux.HandleMessage(egress, nil)
		if err != nil {
			return err
		}
		return nil
	})

	opt.AdapterStop(func(adp *{{.FileName}}Publisher, _ *{{.FileName}}Egress) error {
		return nil
	})

	var mu sync.Mutex
	opt.AdapterFixup(0, func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	publisher := Artifex.NewPublisher(opt)
	publisher.
		AddSpawnHandler(func() error {
			err := f.PubHub.Join(publisher.Identifier(), publisher)
			if err != nil {
				return err
			}
			return nil
		}).
		AddExitHandler(func() error {
			f.PubHub.RemoveByKey(publisher.Identifier())
			return nil
		})

	return publisher, nil
}

//

type {{.FileName}}Subscriber = Artifex.Adapter[{{.FileName}}Ingress, struct{}]
type {{.FileName}}SubscriberHub = Artifex.Hub[*{{.FileName}}Subscriber]

func New{{.FileName}}SubscriberHub() *{{.FileName}}SubscriberHub {
	stop := func(subscriber *{{.FileName}}Subscriber) error {
		return subscriber.Stop()
	}
	return Artifex.NewHub(stop)
}

func (f *{{.FileName}}Factory) CreateSubscriber() (*{{.FileName}}Subscriber, error) {

	ingressMux, _ := f.NewMux()

	waitNotify := make(chan error, 1)
	opt := Artifex.NewSubscriberOption[{{.FileName}}Ingress]().
		Identifier("").
		HandleRecv(ingressMux.HandleMessage).
		SendPing(func() error { return nil }, waitNotify, 30)

	opt.AdapterRecv(func(adp *{{.FileName}}Subscriber) (*{{.FileName}}Ingress, error) {
		return New{{.FileName}}Ingress(), nil
	})

	opt.AdapterStop(func(adp *{{.FileName}}Subscriber, _ *struct{}) error {
		return nil
	})

	var mu sync.Mutex
	opt.AdapterFixup(0, func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	})

	subscriber := Artifex.NewSubscriber(opt)
	subscriber.
		AddSpawnHandler(func() error {
			err := f.SubHub.Join(subscriber.Identifier(), subscriber)
			if err != nil {
				return err
			}
			return nil
		}).
		AddExitHandler(func() error {
			f.SubHub.RemoveByKey(subscriber.Identifier())
			return nil
		})

	return subscriber, nil
}
`
