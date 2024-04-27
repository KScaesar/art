package main

const AdapterTmpl = `
package {{.Package}}

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

func New{{.FileName}}Ingress(infra any, metadata any, bBody []byte) *{{.FileName}}Ingress {
	message := Artifex.NewMessage()

	message.RawInfra = infra
	message.Metadata.Set("metadata", metadata)
	message.Bytes = bBody
	return message
}

type {{.FileName}}Ingress = Artifex.Message

//

func New{{.FileName}}IngressMux() *{{.FileName}}IngressMux {
	mux := Artifex.NewMux("/")

	mux.Transform(func(message *{{.FileName}}Ingress, dep any) error {
		meta := message.Metadata.Str("metadata")
		if meta == "pingpong" {
			message.Subject = "pingpong"
		}
		return nil
	})
	return mux
}

type {{.FileName}}IngressMux = Artifex.Mux

//

func New{{.FileName}}Egress() *{{.FileName}}Egress {
	message := Artifex.NewMessage()

	return message
}

type {{.FileName}}Egress = Artifex.Message

//

func New{{.FileName}}EgressMux() *{{.FileName}}EgressMux {
	mux := Artifex.NewMux("/")

	mux.PostMiddleware(func(message *{{.FileName}}Egress, dep any) error {
		adp := dep.(Artifex.IAdapter)
		conn := adp.RawInfra().(any)
		_ = conn
		return nil
	})
	return mux
}

type {{.FileName}}EgressMux = Artifex.Mux

//

type {{.FileName}}Adapter = Artifex.Prosumer
type {{.FileName}}Producer = Artifex.Producer
type {{.FileName}}Consumer = Artifex.Consumer

//

type {{.FileName}}Factory struct {
	Hub             *Artifex.Hub
	Logger          Artifex.Logger

	SendPingSeconds int
	WaitPingSeconds int
	PrintPingPong   bool

	MaxRetrySeconds int

	Authenticate func() (name string, err error)
	AdapterName  string

	DecodeTransform func() (bBody []byte, err error)

	IngressMux      *{{.FileName}}IngressMux
	EgressMux       *{{.FileName}}EgressMux
	DecorateAdapter func(adapter Artifex.IAdapter) (application Artifex.IAdapter)
	Lifecycle       func(lifecycle *Artifex.Lifecycle)
}

func (f *{{.FileName}}Factory) CreateAdapter() (adapter {{.FileName}}Adapter, err error) {
	name, err := f.Authenticate()
	if err != nil {
		return nil, err
	}

	opt := Artifex.NewAdapterOption().
		Identifier(name).
		AdapterHub(f.Hub).
		Logger(f.Logger).
		IngressMux(f.IngressMux).
		EgressMux(f.EgressMux).
		DecorateAdapter(f.DecorateAdapter).
		Lifecycle(f.Lifecycle).
		RawInfra(nil)

	var mu sync.Mutex

	// send pint, wait pong
	sendPing := func(adp Artifex.IAdapter) error {
		if f.PrintPingPong {
			adp.(Artifex.IAdapter).Log().Info("send ping")
		}
		return nil
	}
	waitPong := make(chan error, 1)
	opt.SendPing(sendPing, waitPong, f.SendPingSeconds*2)

	// wait ping, send pong
	waitPing := make(chan error, 1)
	sendPong := func(adp Artifex.IAdapter) error {
		if f.PrintPingPong {
			adp.(Artifex.IAdapter).Log().Info("send pong")
		}

		return nil
	}
	opt.WaitPing(waitPing, f.WaitPingSeconds, sendPong)

	opt.AdapterRecv(func(logger Artifex.Logger) (*Artifex.Message, error) {

		var err error
		if err != nil {
			return nil, err
		}
		return Artifex.NewMessageWithBytes("",nil), nil
	})

	opt.AdapterSend(func(logger Artifex.Logger, message *Artifex.Message) (err error) {
		logger = logger.WithKeyValue("msg_id", message.MsgId())

		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			logger.Error("send %q: %v", message.Subject, err)
			return
		}
		logger.Info("send %q", message.Subject)
		return 
	})

	opt.AdapterStop(func(logger Artifex.Logger) (err error) {
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			logger.Error("stop: %v", err)
			return
		}
		logger.Info("stop")
		return nil
	})

	retry := 0
	opt.AdapterFixup(f.MaxRetrySeconds, func(adp Artifex.IAdapter) error {
		mu.Lock()
		defer mu.Unlock()
		logger := adp.Log()

		retry++
		logger.Info("retry %v times start", retry)
		if err != nil {
			logger.Error("retry: %v", err)
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
	return adp.({{.FileName}}Adapter), err
}
`
