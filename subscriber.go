package Artifex

import (
	"sync"
	"sync/atomic"

	"github.com/gookit/goutil/maputil"
)

type SubscriberFactory[rMessage any] interface {
	CreateSubscriber() (*Subscriber[rMessage], error)
}

type Subscriber[rMessage any] struct {
	HandleRecv          HandleFunc[rMessage]      // Must
	AdapterRecv         func() (*rMessage, error) // Must
	AdapterStop         func() error              // Must
	Fixup               func() error
	FixupMaxRetrySecond int

	Identifier string
	AppData    maputil.Data
	Mutex      sync.RWMutex
	Lifecycle  Lifecycle

	isStop atomic.Bool
	isInit atomic.Bool
}

func (sub *Subscriber[rMessage]) init() error {
	if sub.isInit.Load() {
		return nil
	}

	err := sub.Lifecycle.Execute()
	if err != nil {
		return err
	}

	if sub.HandleRecv == nil || sub.AdapterStop == nil || sub.AdapterRecv == nil {
		return ErrorWrapWithMessage(ErrInvalidParameter, "subscriber")
	}

	if sub.AppData == nil {
		sub.AppData = make(maputil.Data)
	}

	sub.isInit.Store(true)
	return nil
}

func (sub *Subscriber[rMessage]) Listen() error {
	if !sub.isInit.Load() {
		sub.Mutex.Lock()
		err := sub.init()
		if err != nil {
			sub.Mutex.Unlock()
			return err
		}
		sub.Mutex.Unlock()
	}

	if sub.Fixup == nil {
		return sub.listen()
	}
	return ReliableTask(sub.listen, sub.IsStop, sub.FixupMaxRetrySecond, sub.Fixup)
}

func (sub *Subscriber[rMessage]) listen() error {
	for !sub.isStop.Load() {
		message, err := sub.AdapterRecv()

		if sub.isStop.Load() {
			return nil
		}

		if err != nil {
			return err
		}

		sub.HandleRecv(message, nil)
	}
	return nil
}

func (sub *Subscriber[rMessage]) IsStop() bool {
	return sub.isStop.Load()
}

func (sub *Subscriber[rMessage]) Stop() error {
	if !sub.isInit.Load() {
		sub.Mutex.Lock()
		err := sub.init()
		if err != nil {
			sub.Mutex.Unlock()
			return err
		}
		sub.Mutex.Unlock()
	}

	if sub.isStop.Load() {
		return nil
	}
	sub.isStop.Store(true)
	sub.Lifecycle.NotifyExit()
	err := sub.AdapterStop()
	if err != nil {
		return err
	}
	return nil
}

func (sub *Subscriber[rMessage]) PingPong(pp PingPong) error {
	if !sub.isInit.Load() {
		sub.Mutex.Lock()
		err := sub.init()
		if err != nil {
			sub.Mutex.Unlock()
			return err
		}
		sub.Mutex.Unlock()
	}

	err := pp.validate()
	if err != nil {
		return err
	}
	defer sub.Stop()

	if sub.Fixup == nil {
		return pp.Execute(sub.IsStop)
	}

	return ReliableTask(
		func() error {
			return pp.Execute(sub.IsStop)
		},
		sub.IsStop,
		sub.FixupMaxRetrySecond,
		sub.Fixup,
	)
}
