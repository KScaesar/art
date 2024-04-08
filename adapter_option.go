package Artifex

import (
	"sync"

	"github.com/gookit/goutil/maputil"
)

func NewPubSubOption[rMessage, sMessage any]() (opt *AdapterOption[rMessage, sMessage]) {
	return &AdapterOption[rMessage, sMessage]{}
}

func NewPublisherOption[sMessage any]() (opt *AdapterOption[struct{}, sMessage]) {
	return &AdapterOption[struct{}, sMessage]{}
}

func NewSubscriberOption[rMessage any]() (opt *AdapterOption[rMessage, struct{}]) {
	return &AdapterOption[rMessage, struct{}]{}
}

type AdapterOption[rMessage, sMessage any] struct {
	handleRecv  HandleFunc[rMessage]
	adapterRecv func(IAdapter) (*rMessage, error)
	adapterSend func(IAdapter, *sMessage) error
	adapterStop func(IAdapter, *sMessage) error

	fixupMaxRetrySecond int
	adapterFixup        func(IAdapter) error

	pingpong func(isStop func() bool) error

	identifier string

	newLifecycle func() *Lifecycle
}

func (opt *AdapterOption[rMessage, sMessage]) BuildPubSub() (*Adapter[rMessage, sMessage], error) {
	pubsub := &Adapter[rMessage, sMessage]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		lifecycle:           opt.lifecycle(),
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		waitStop:            make(chan struct{}),
	}

	// must
	pubsub.handleRecv = opt.handleRecv
	pubsub.adapterRecv = opt.adapterRecv
	pubsub.adapterSend = opt.adapterSend
	pubsub.adapterStop = opt.adapterStop
	return pubsub, pubsub.init()
}

func (opt *AdapterOption[rMessage, sMessage]) BuildPublisher() (*Adapter[rMessage, sMessage], error) {
	publisher := &Adapter[rMessage, sMessage]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		lifecycle:           opt.lifecycle(),
		adpMutex:            sync.RWMutex{},
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		waitStop:            make(chan struct{}),
	}

	// must
	publisher.handleRecv = nil
	publisher.adapterRecv = nil
	publisher.adapterSend = opt.adapterSend
	publisher.adapterStop = opt.adapterStop
	return publisher, publisher.init()
}

func (opt *AdapterOption[rMessage, sMessage]) BuildSubscriber() (*Adapter[rMessage, sMessage], error) {
	subscriber := &Adapter[rMessage, sMessage]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		lifecycle:           opt.lifecycle(),
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		waitStop:            make(chan struct{}),
	}

	// must
	subscriber.handleRecv = opt.handleRecv
	subscriber.adapterRecv = opt.adapterRecv
	subscriber.adapterSend = nil
	subscriber.adapterStop = opt.adapterStop
	return subscriber, subscriber.init()
}

func (opt *AdapterOption[rMessage, sMessage]) HandleRecv(handleRecv HandleFunc[rMessage]) *AdapterOption[rMessage, sMessage] {
	opt.handleRecv = handleRecv
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterRecv(adapterRecv func(adp IAdapter) (*rMessage, error)) *AdapterOption[rMessage, sMessage] {
	opt.adapterRecv = adapterRecv
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterSend(adapterSend func(adp IAdapter, egress *sMessage) error) *AdapterOption[rMessage, sMessage] {
	opt.adapterSend = adapterSend
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterStop(adapterStop func(adp IAdapter, egress *sMessage) error) *AdapterOption[rMessage, sMessage] {
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

func (opt *AdapterOption[rMessage, sMessage]) Identifier(identifier string) *AdapterOption[rMessage, sMessage] {
	opt.identifier = identifier
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) NewLifecycle(newFunc func() *Lifecycle) *AdapterOption[rMessage, sMessage] {
	opt.newLifecycle = newFunc
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) lifecycle() *Lifecycle {
	if opt.newLifecycle == nil {
		return &Lifecycle{}
	}
	return opt.newLifecycle()
}
