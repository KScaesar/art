package Artifex

import (
	"github.com/gookit/goutil/maputil"
)

func NewPubSubOption[Ingress, Egress any]() (opt *AdapterOption[Ingress, Egress]) {
	return &AdapterOption[Ingress, Egress]{
		lifecycle: new(Lifecycle),
	}
}

func (opt *AdapterOption[Ingress, Egress]) BuildPubSub() (*Adapter[Ingress, Egress], error) {
	pubsub := &Adapter[Ingress, Egress]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		appData:             make(maputil.Data),
		isStopped:           false,
		waitStop:            make(chan struct{}),
	}

	// must
	pubsub.identifier = opt.identifier
	pubsub.handleRecv = opt.handleRecv
	pubsub.adapterRecv = opt.adapterRecv
	pubsub.adapterSend = opt.adapterSend
	pubsub.adapterStop = opt.adapterStop
	pubsub.lifecycle = opt.lifecycle
	return pubsub, pubsub.init()
}

func NewPublisherOption[Egress any]() (opt *AdapterOption[struct{}, Egress]) {
	return &AdapterOption[struct{}, Egress]{
		lifecycle: new(Lifecycle),
	}
}

func (opt *AdapterOption[Ingress, Egress]) BuildPublisher() (*Adapter[Ingress, Egress], error) {
	publisher := &Adapter[Ingress, Egress]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		appData:             make(maputil.Data),
		waitStop:            make(chan struct{}),
	}

	// must
	publisher.identifier = opt.identifier
	publisher.adapterSend = opt.adapterSend
	publisher.adapterStop = opt.adapterStop
	publisher.lifecycle = opt.lifecycle
	return publisher, publisher.init()
}

func NewSubscriberOption[Ingress any]() (opt *AdapterOption[Ingress, struct{}]) {
	return &AdapterOption[Ingress, struct{}]{
		lifecycle: new(Lifecycle),
	}
}

func (opt *AdapterOption[Ingress, Egress]) BuildSubscriber() (*Adapter[Ingress, Egress], error) {
	subscriber := &Adapter[Ingress, Egress]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		appData:             make(maputil.Data),
		waitStop:            make(chan struct{}),
	}

	// must
	subscriber.identifier = opt.identifier
	subscriber.handleRecv = opt.handleRecv
	subscriber.adapterRecv = opt.adapterRecv
	subscriber.adapterStop = opt.adapterStop
	subscriber.lifecycle = opt.lifecycle
	return subscriber, subscriber.init()
}

//

type AdapterOption[Ingress, Egress any] struct {
	identifier string

	handleRecv  HandleFunc[Ingress]
	adapterRecv func(IAdapter) (*Ingress, error)
	adapterSend func(IAdapter, *Egress) error
	adapterStop func(IAdapter) error

	fixupMaxRetrySecond int
	adapterFixup        func(IAdapter) error

	pingpong func(isStop func() bool) error

	lifecycle *Lifecycle
}

func (opt *AdapterOption[Ingress, Egress]) Identifier(identifier string) *AdapterOption[Ingress, Egress] {
	opt.identifier = identifier
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) HandleRecv(handleRecv HandleFunc[Ingress]) *AdapterOption[Ingress, Egress] {
	opt.handleRecv = handleRecv
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterRecv(adapterRecv func(adp IAdapter) (message *Ingress, err error)) *AdapterOption[Ingress, Egress] {
	opt.adapterRecv = adapterRecv
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterSend(adapterSend func(adp IAdapter, message *Egress) error) *AdapterOption[Ingress, Egress] {
	opt.adapterSend = adapterSend
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterStop(adapterStop func(adp IAdapter) error) *AdapterOption[Ingress, Egress] {
	opt.adapterStop = adapterStop
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterFixup(maxRetrySecond int, adapterFixup func(IAdapter) error) *AdapterOption[Ingress, Egress] {
	opt.fixupMaxRetrySecond = maxRetrySecond
	opt.adapterFixup = adapterFixup
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

	opt.pingpong = func(isStop func() bool) error { return SendPingWaitPong(sendPing, waitPong, isStop, second) }
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

	opt.pingpong = func(isStop func() bool) error { return WaitPingSendPong(waitPing, sendPong, isStop, second) }
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) Lifecycle(setup func(life *Lifecycle)) *AdapterOption[Ingress, Egress] {
	setup(opt.lifecycle)
	return opt
}
