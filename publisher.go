package Artifex

import (
	"sync"
	"sync/atomic"

	"github.com/gookit/goutil/maputil"
)

type PublisherFactory[sMessage any] interface {
	CreatePublisher() (*Publisher[sMessage], error)
}

type Publisher[sMessage any] struct {
	AdapterSend         func(*sMessage) error // Must
	AdapterStop         func() error          // Must
	Fixup               func() error
	FixupMaxRetrySecond int

	Identifier string
	AppData    maputil.Data
	Mutex      sync.RWMutex
	Lifecycle  Lifecycle

	isStop atomic.Bool
	isInit atomic.Bool
}

func (pub *Publisher[sMessage]) init() error {
	if pub.isInit.Load() {
		return nil
	}

	err := pub.Lifecycle.Execute()
	if err != nil {
		return err
	}

	if pub.AdapterStop == nil || pub.AdapterSend == nil {
		return ErrorWrapWithMessage(ErrInvalidParameter, "publisher")
	}

	if pub.AppData == nil {
		pub.AppData = make(maputil.Data)
	}

	pub.isInit.Store(true)
	return nil
}

func (pub *Publisher[sMessage]) Send(message *sMessage) error {
	if !pub.isInit.Load() {
		pub.Mutex.Lock()
		err := pub.init()
		if err != nil {
			pub.Mutex.Unlock()
			return err
		}
		pub.Mutex.Unlock()
	}

	return pub.AdapterSend(message)
}

func (pub *Publisher[sMessage]) IsStop() bool {
	return pub.isStop.Load()
}

func (pub *Publisher[sMessage]) Stop() error {
	if !pub.isInit.Load() {
		pub.Mutex.Lock()
		err := pub.init()
		if err != nil {
			pub.Mutex.Unlock()
			return err
		}
		pub.Mutex.Unlock()
	}

	if pub.isStop.Load() {
		return nil
	}
	err := pub.AdapterStop()
	if err != nil {
		return err
	}
	pub.isStop.Store(true)
	pub.Lifecycle.notifyExit()
	return nil
}

func (pub *Publisher[sMessage]) PingPong(pp PingPong) error {
	if !pub.isInit.Load() {
		pub.Mutex.Lock()
		err := pub.init()
		if err != nil {
			pub.Mutex.Unlock()
			return err
		}
		pub.Mutex.Unlock()
	}

	err := pp.validate()
	if err != nil {
		return err
	}
	defer pub.Stop()

	if pub.Fixup == nil {
		return pp.Execute(pub.IsStop)
	}

	return ReliableTask(
		func() error {
			return pp.Execute(pub.IsStop)
		},
		pub.IsStop,
		pub.FixupMaxRetrySecond,
		pub.Fixup,
	)
}
