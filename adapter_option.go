package Artifex

import (
	"github.com/gookit/goutil/maputil"
)

func NewPubSubOption[rMessage, sMessage any]() (opt *AdapterOption[rMessage, sMessage]) {
	return &AdapterOption[rMessage, sMessage]{
		lifecycle: new(Lifecycle),
	}
}

func (opt *AdapterOption[rMessage, sMessage]) BuildPubSub() (*Adapter[rMessage, sMessage], error) {
	pubsub := &Adapter[rMessage, sMessage]{
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

func NewPublisherOption[sMessage any]() (opt *AdapterOption[struct{}, sMessage]) {
	return &AdapterOption[struct{}, sMessage]{
		lifecycle: new(Lifecycle),
	}
}

func (opt *AdapterOption[rMessage, sMessage]) BuildPublisher() (*Adapter[rMessage, sMessage], error) {
	publisher := &Adapter[rMessage, sMessage]{
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

func NewSubscriberOption[rMessage any]() (opt *AdapterOption[rMessage, struct{}]) {
	return &AdapterOption[rMessage, struct{}]{
		lifecycle: new(Lifecycle),
	}
}

func (opt *AdapterOption[rMessage, sMessage]) BuildSubscriber() (*Adapter[rMessage, sMessage], error) {
	subscriber := &Adapter[rMessage, sMessage]{
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

type AdapterOption[rMessage, sMessage any] struct {
	identifier string

	handleRecv  HandleFunc[rMessage]
	adapterRecv func(IAdapter) (*rMessage, error)
	adapterSend func(IAdapter, *sMessage) error
	adapterStop func(IAdapter) error

	fixupMaxRetrySecond int
	adapterFixup        func(IAdapter) error

	pingpong func(isStop func() bool) error

	lifecycle *Lifecycle
}

func (opt *AdapterOption[rMessage, sMessage]) Identifier(identifier string) *AdapterOption[rMessage, sMessage] {
	opt.identifier = identifier
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) HandleRecv(handleRecv HandleFunc[rMessage]) *AdapterOption[rMessage, sMessage] {
	opt.handleRecv = handleRecv
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterRecv(adapterRecv func(adp IAdapter) (message *rMessage, err error)) *AdapterOption[rMessage, sMessage] {
	opt.adapterRecv = adapterRecv
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterSend(adapterSend func(adp IAdapter, message *sMessage) error) *AdapterOption[rMessage, sMessage] {
	opt.adapterSend = adapterSend
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterStop(adapterStop func(adp IAdapter) error) *AdapterOption[rMessage, sMessage] {
	opt.adapterStop = adapterStop
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterFixup(maxRetrySecond int, adapterFixup func(IAdapter) error) *AdapterOption[rMessage, sMessage] {
	opt.fixupMaxRetrySecond = maxRetrySecond
	opt.adapterFixup = adapterFixup
	return opt
}

// SendPing
//
// When SendPingWaitPong sends a ping message and waits for a corresponding pong message.
// SendPeriod = WaitSecond / 2
func (opt *AdapterOption[rMessage, sMessage]) SendPing(sendPing func() error, waitPong chan error, waitPongSecond int) *AdapterOption[rMessage, sMessage] {
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
func (opt *AdapterOption[rMessage, sMessage]) WaitPing(waitPing chan error, waitPingSecond int, sendPong func() error) *AdapterOption[rMessage, sMessage] {
	second := waitPingSecond
	if second <= 0 {
		second = 30
	}

	opt.pingpong = func(isStop func() bool) error { return WaitPingSendPong(waitPing, sendPong, isStop, second) }
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) Lifecycle(setup func(life *Lifecycle)) *AdapterOption[rMessage, sMessage] {
	setup(opt.lifecycle)
	return opt
}
