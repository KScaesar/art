package Artifex

import (
	"github.com/gookit/goutil/maputil"
)

func NewPubSubOption[Ingress, Egress any]() (opt *AdapterOption[Ingress, Egress]) {
	pubsub := &Adapter[Ingress, Egress]{
		recvResult: make(chan error, 2),
		appData:    make(maputil.Data),
		lifecycle:  new(Lifecycle),
		waitStop:   make(chan struct{}),
	}
	return &AdapterOption[Ingress, Egress]{
		adapter: pubsub,
	}
}

func NewPublisherOption[Egress any]() (opt *AdapterOption[struct{}, Egress]) {
	pub := &Adapter[struct{}, Egress]{
		recvResult: make(chan error, 2),
		appData:    make(maputil.Data),
		lifecycle:  new(Lifecycle),
		waitStop:   make(chan struct{}),
	}
	return &AdapterOption[struct{}, Egress]{
		adapter: pub,
	}
}

func NewSubscriberOption[Ingress any]() (opt *AdapterOption[Ingress, struct{}]) {
	sub := &Adapter[Ingress, struct{}]{
		recvResult: make(chan error, 2),
		appData:    make(maputil.Data),
		lifecycle:  new(Lifecycle),
		waitStop:   make(chan struct{}),
	}
	return &AdapterOption[Ingress, struct{}]{
		adapter: sub,
	}
}

type AdapterOption[Ingress, Egress any] struct {
	adapter *Adapter[Ingress, Egress]
}

func (opt *AdapterOption[Ingress, Egress]) Build() (*Adapter[Ingress, Egress], error) {
	pubsub := opt.adapter
	opt.adapter = nil
	return pubsub, pubsub.init()
}

func (opt *AdapterOption[Ingress, Egress]) Identifier(identifier string) *AdapterOption[Ingress, Egress] {
	pubsub := opt.adapter
	pubsub.identifier = identifier
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) HandleRecv(handleRecv HandleFunc[Ingress]) *AdapterOption[Ingress, Egress] {
	sub := opt.adapter
	sub.handleRecv = handleRecv
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterRecv(adapterRecv func(adp IAdapter) (message *Ingress, err error)) *AdapterOption[Ingress, Egress] {
	sub := opt.adapter
	sub.adapterRecv = adapterRecv
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterSend(adapterSend func(adp IAdapter, message *Egress) error) *AdapterOption[Ingress, Egress] {
	pub := opt.adapter
	pub.adapterSend = adapterSend
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterStop(adapterStop func(adp IAdapter) error) *AdapterOption[Ingress, Egress] {
	pubsub := opt.adapter
	pubsub.adapterStop = adapterStop
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterFixup(maxRetrySecond int, adapterFixup func(IAdapter) error) *AdapterOption[Ingress, Egress] {
	pubsub := opt.adapter
	pubsub.fixupMaxRetrySecond = maxRetrySecond
	pubsub.adapterFixup = adapterFixup
	return opt
}

// SendPing
//
// When SendPingWaitPong sends a ping message and waits for a corresponding pong message.
// SendPeriod = WaitSecond / 2
func (opt *AdapterOption[Ingress, Egress]) SendPing(sendPing func() error, waitPong chan error, waitPongSecond int) *AdapterOption[Ingress, Egress] {
	second := waitPongSecond
	if second <= 0 {
		second = 30
	}

	pubsub := opt.adapter
	pubsub.pingpong = func() error { return SendPingWaitPong(sendPing, waitPong, pubsub.IsStopped, second) }
	return opt
}

// WaitPing
//
// When WaitPingSendPong waits for a ping message and response a corresponding pong message.
// SendPeriod = WaitSecond
func (opt *AdapterOption[Ingress, Egress]) WaitPing(waitPing chan error, waitPingSecond int, sendPong func() error) *AdapterOption[Ingress, Egress] {
	second := waitPingSecond
	if second <= 0 {
		second = 30
	}

	pubsub := opt.adapter
	pubsub.pingpong = func() error { return WaitPingSendPong(waitPing, sendPong, pubsub.IsStopped, second) }
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) Lifecycle(setup func(life *Lifecycle)) *AdapterOption[Ingress, Egress] {
	setup(opt.adapter.lifecycle)
	return opt
}
