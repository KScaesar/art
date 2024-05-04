package main

const AdapterTmpl = `
package {{.Package}}

import (
	"sync"

	"github.com/KScaesar/art"
)

type {{.FileName}}Adapter = art.Prosumer
type {{.FileName}}Producer = art.Producer
type {{.FileName}}Consumer = art.Consumer

//

type {{.FileName}}Factory struct {
	Hub    *art.Hub
	Logger art.Logger

	SendPingSeconds int
	WaitPingSeconds int

	MaxRetrySeconds int

	Authenticate func() (name string, err error)
	AdapterName  string

	IngressMux      *{{.FileName}}IngressMux
	EgressMux       *{{.FileName}}EgressMux
	DecorateAdapter func(adapter art.IAdapter) (application art.IAdapter)
	Lifecycle       func(lifecycle *art.Lifecycle)
}

func (f *{{.FileName}}Factory) CreateAdapter() (adapter {{.FileName}}Adapter, err error) {
	name, err := f.Authenticate()
	if err != nil {
		return nil, err
	}

	opt := art.NewAdapterOption().
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
	sendPing := func(adp art.IAdapter) error {
		return nil
	}
	waitPong := art.NewWaitPingPong()
	opt.SendPing(f.SendPingSeconds, waitPong, sendPing)

	// wait ping, send pong
	waitPing := art.NewWaitPingPong()
	sendPong := func(adp art.IAdapter) error {
		return nil
	}
	opt.WaitPing(f.WaitPingSeconds, waitPing, sendPong)

	opt.RawRecv(func(logger art.Logger) (*art.Message, error) {

		var err error
		if err != nil {
			logger.Error("recv: %v", err)
			return nil, err
		}
		return New{{.FileName}}Ingress(nil, nil, nil), nil
	})

	opt.RawSend(func(logger art.Logger, message *art.Message) (err error) {
		mu.Lock()
		defer mu.Unlock()
		return 
	})

	opt.RawStop(func(logger art.Logger) (err error) {
		mu.Lock()
		defer mu.Unlock()

		if err != nil {
			logger.Error("stop: %v", err)
			return
		}
		return nil
	})

	retry := 0
	opt.RawFixup(f.MaxRetrySeconds, func(adp art.IAdapter) (err error) {
		mu.Lock()
		defer mu.Unlock()

		retry++
		logger := adp.Log()
		logger.Info("retry %v times", retry)

		logger.Info("retry xxx start")
		if err != nil {
			logger.Error("retry xxx fail: %v", err)
			return err
		}
		logger.Info("retry xxx ok")

		retry = 0
		return nil
	})

	adp, err := opt.Build()
	if err != nil {
		return
	}
	return adp.({{.FileName}}Adapter), err
}
`
