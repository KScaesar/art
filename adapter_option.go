package Artifex

import (
	"reflect"
)

func NewAdapterOption() (opt *AdapterOption) {
	pubsub := &Adapter{
		recvResult: make(chan error, 2),
		lifecycle:  new(Lifecycle),
		waitStop:   make(chan struct{}),
	}
	return &AdapterOption{
		adapter: pubsub,
	}
}

type AdapterOption struct {
	adapter         *Adapter
	decorateAdapter func(adp IAdapter) (app IAdapter)
}

func (opt *AdapterOption) Build() (adp IAdapter, err error) {
	pubsub := opt.adapter

	if pubsub.logger == nil {
		pubsub.logger = DefaultLogger()
	}

	if opt.decorateAdapter != nil {
		pubsub.application = opt.decorateAdapter(pubsub)
	} else {
		pubsub.application = pubsub
	}

	if !reflect.ValueOf(pubsub.hub).IsZero() {
		err = pubsub.hub.Join(pubsub.identifier, pubsub.application)
		if err != nil {
			return nil, err
		}
	}

	err = pubsub.lifecycle.initialize(pubsub.application)
	if err != nil {
		return nil, err
	}

	pubsub.pingpong()

	return pubsub.application, nil
}

func (opt *AdapterOption) DecorateAdapter(wrap func(adapter IAdapter) (application IAdapter)) *AdapterOption {
	opt.decorateAdapter = wrap
	return opt
}

func (opt *AdapterOption) Identifier(identifier string) *AdapterOption {
	pubsub := opt.adapter
	pubsub.identifier = identifier
	return opt
}

func (opt *AdapterOption) Logger(logger Logger) *AdapterOption {
	pubsub := opt.adapter
	pubsub.logger = logger
	return opt
}

func (opt *AdapterOption) AdapterHub(hub AdapterHub) *AdapterOption {
	pubsub := opt.adapter
	pubsub.hub = hub
	return opt
}

func (opt *AdapterOption) Lifecycle(setup func(life *Lifecycle)) *AdapterOption {
	if setup != nil {
		setup(opt.adapter.lifecycle)
	}
	return opt
}

func (opt *AdapterOption) IngressMux(mux *Mux) *AdapterOption {
	sub := opt.adapter
	sub.ingressMux = mux
	return opt
}

func (opt *AdapterOption) AdapterRecv(adapterRecv func(logger Logger) (message *Message, err error)) *AdapterOption {
	sub := opt.adapter
	sub.adapterRecv = adapterRecv
	return opt
}

func (opt *AdapterOption) EgressMux(mux *Mux) *AdapterOption {
	sub := opt.adapter
	sub.egressMux = mux
	return opt
}

func (opt *AdapterOption) AdapterSend(adapterSend func(logger Logger, message *Message) error) *AdapterOption {
	pub := opt.adapter
	pub.adapterSend = adapterSend
	return opt
}

func (opt *AdapterOption) AdapterStop(adapterStop func(logger Logger) error) *AdapterOption {
	pubsub := opt.adapter
	pubsub.adapterStop = adapterStop
	return opt
}

func (opt *AdapterOption) AdapterFixup(maxRetrySecond int, adapterFixup func(IAdapter) error) *AdapterOption {
	if maxRetrySecond < 0 {
		return opt
	}
	if maxRetrySecond == 0 {
		const RetryUntilAdapterStop = 0
		maxRetrySecond = RetryUntilAdapterStop
	}

	pubsub := opt.adapter
	pubsub.fixupMaxRetrySecond = maxRetrySecond
	pubsub.adapterFixup = adapterFixup
	return opt
}

// SendPing
//
// When SendPingWaitPong sends a ping message and waits for a corresponding pong message.
// SendPeriod = WaitSecond / 2
func (opt *AdapterOption) SendPing(sendPingSeconds int, waitPong WaitPingPong, sendPing func(IAdapter) error) *AdapterOption {
	sendSeconds := sendPingSeconds
	if sendSeconds < 0 {
		return opt
	}
	if sendSeconds == 0 {
		sendSeconds = 30
	}

	pubsub := opt.adapter
	pubsub.pp = func() error {
		return SendPingWaitPong(sendSeconds, func() error { return sendPing(pubsub) }, waitPong, pubsub.IsStopped)
	}
	return opt
}

// WaitPing
//
// When WaitPingSendPong waits for a ping message and response a corresponding pong message.
// SendPeriod = WaitSecond
func (opt *AdapterOption) WaitPing(waitPingSeconds int, waitPing WaitPingPong, sendPong func(IAdapter) error) *AdapterOption {
	waitSeconds := waitPingSeconds
	if waitSeconds < 0 {
		return opt
	}
	if waitSeconds == 0 {
		waitSeconds = 30
	}

	pubsub := opt.adapter
	pubsub.pp = func() error {
		return WaitPingSendPong(waitSeconds, waitPing, func() error { return sendPong(pubsub) }, pubsub.IsStopped)
	}
	return opt
}

func (opt *AdapterOption) RawInfra(infra any) *AdapterOption {
	pubsub := opt.adapter
	pubsub.rawInfra = infra
	return opt
}
