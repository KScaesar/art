package main

const AdapterTmpl = `
package {{.Package}}

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

func New{{.FileName}}Ingress(bBody []byte, metadata any, pingpong chan error) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Bytes = bBody
	{{.FileName}}Metadata.SetPingPong(message.Metadata, pingpong)
	message.RawInfra = nil
	return message
}

func New{{.FileName}}Egress(body any) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Body = body
	return message
}

func New{{.FileName}}EgressWithSubject(subject string, body any) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Subject = subject
	message.Body = body
	return message
}

//

func New{{.FileName}}IngressMux(pingpong bool) *{{.FileName}}IngressMux {
	in := Artifex.NewMux("/").
		Transform(func(message *Artifex.Message, dep any) error {
			return nil
		}).
		Handler("pingpong", func(message *Artifex.Message, dep any) error {
			if pingpong {
				dep.(Artifex.IAdapter).Log().Info("ack pingpong")
			}
			{{.FileName}}Metadata.PingPong(message.Metadata) <- nil
			return nil
		})
	return in
}

func New{{.FileName}}EgressMux(pingpong bool) *{{.FileName}}EgressMux {
	out := Artifex.NewMux("/").
		Transform(func(message *Artifex.Message, dep any) error {
			return nil
		}).
		Handler("pingpong", func(message *Artifex.Message, dep any) error {
			if pingpong {
				dep.(Artifex.IAdapter).Log().Info("send pingpong")
			}
			return nil
		}).
		DefaultHandler(Artifex.UseSkipMessage())
	return out
}

type {{.FileName}}IngressMux = Artifex.Mux
type {{.FileName}}EgressMux = Artifex.Mux

//

type {{.FileName}}Adapter = Artifex.Prosumer
type {{.FileName}}Producer = Artifex.Producer
type {{.FileName}}Consumer = Artifex.Consumer

//

type {{.FileName}}Factory struct {
	Hub    *Artifex.Hub
	Logger Artifex.Logger

	SendPingSeconds int
	WaitPingSeconds int

	MaxRetrySeconds int

	Authenticate func() (name string, err error)
	AdapterName  string

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
		RawInfra(nil).
		Identifier(name).
		AdapterHub(f.Hub).
		Logger(f.Logger).
		IngressMux(f.IngressMux).
		EgressMux(f.EgressMux).
		DecorateAdapter(f.DecorateAdapter).
		Lifecycle(f.Lifecycle)

	var mu sync.Mutex

	// send pint, wait pong
	sendPing := func(adp Artifex.IAdapter) error {
		return nil
	}
	waitPong := make(chan error, 1)
	opt.SendPing(sendPing, waitPong, f.SendPingSeconds*2)

	// wait ping, send pong
	waitPing := make(chan error, 1)
	sendPong := func(adp Artifex.IAdapter) error {
		return nil
	}
	opt.WaitPing(waitPing, f.WaitPingSeconds, sendPong)

	opt.AdapterRecv(func(logger Artifex.Logger) (*Artifex.Message, error) {

		var err error
		if err != nil {
			logger.Error("recv: %v", err)
			return nil, err
		}
		return New{{.FileName}}Ingress(nil, nil, nil), nil
	})

	opt.AdapterSend(func(logger Artifex.Logger, message *Artifex.Message) (err error) {
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			logger.Error("send %q: %v", message.Subject, err)
			return
		}
		return 
	})

	opt.AdapterStop(func(logger Artifex.Logger) (err error) {
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			logger.Error("stop: %v", err)
			return
		}
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
