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

	isStop   atomic.Bool
	onceInit sync.Once
}

func (pub *Publisher[sMessage]) init() error {
	var err error
	pub.onceInit.Do(func() {
		err = pub.Lifecycle.Execute()
		if err != nil {
			return
		}

		if pub.AdapterStop == nil || pub.AdapterSend == nil {
			err = ErrorWrapWithMessage(ErrInvalidParameter, "publisher")
			return
		}

		if pub.AppData == nil {
			pub.AppData = make(maputil.Data)
		}
	})
	return err
}

func (pub *Publisher[sMessage]) Send(message *sMessage) error {
	err := pub.init()
	if err != nil {
		return err
	}

	return pub.AdapterSend(message)
}

func (pub *Publisher[sMessage]) IsStop() bool {
	return pub.isStop.Load()
}

func (pub *Publisher[sMessage]) Stop() error {
	err := pub.init()
	if err != nil {
		return err
	}

	if pub.isStop.Load() {
		return nil
	}
	pub.isStop.Store(true)
	pub.Lifecycle.NotifyExit()
	err = pub.AdapterStop()
	if err != nil {
		return err
	}
	return nil
}

func (pub *Publisher[sMessage]) PingPong(pp PingPong) error {
	err := pub.init()
	if err != nil {
		return err
	}

	err = pp.validate()
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
