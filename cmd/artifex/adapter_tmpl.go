package main

const AdapterTmpl = `
package {{.Package}}

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

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

	// send ping, wait pong
	sendPing := func(adp Artifex.IAdapter) error {
		return nil
	}
	waitPong := Artifex.NewWaitPingPong()
	opt.SendPing(f.SendPingSeconds, waitPong, sendPing)

	// wait ping, send pong
	waitPing := Artifex.NewWaitPingPong()
	sendPong := func(adp Artifex.IAdapter) error {
		return nil
	}
	opt.WaitPing(f.WaitPingSeconds, waitPing, sendPong)

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
