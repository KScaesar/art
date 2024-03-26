package Artifex

import (
	"sync"
	"sync/atomic"

	"github.com/gookit/goutil/maputil"
)

type PubSubFactory[rMessage, sMessage any] interface {
	CreatePubSub() (*PubSub[rMessage, sMessage], error)
}

type PubSub[rMessage, sMessage any] struct {
	HandleRecv          HandleFunc[rMessage]      // Must
	AdapterRecv         func() (*rMessage, error) // Must
	AdapterSend         func(*sMessage) error     // Must
	AdapterStop         func(*sMessage) error     // Must
	Fixup               func() error
	FixupMaxRetrySecond int

	Identifier string
	AppData    maputil.Data
	Mutex      sync.RWMutex
	Lifecycle  Lifecycle

	isStop   atomic.Bool
	onceInit sync.Once
}

func (pubsub *PubSub[rMessage, sMessage]) init() error {
	var err error
	pubsub.onceInit.Do(func() {
		err = pubsub.Lifecycle.Execute()
		if err != nil {
			return
		}

		if pubsub.HandleRecv == nil || pubsub.AdapterStop == nil || pubsub.AdapterRecv == nil || pubsub.AdapterSend == nil {
			err = ErrorWrapWithMessage(ErrInvalidParameter, "pubsub")
			return
		}

		if pubsub.AppData == nil {
			pubsub.AppData = make(maputil.Data)
		}
	})
	return err
}

func (pubsub *PubSub[rMessage, sMessage]) Listen() error {
	pubsub.Mutex.Lock()
	err := pubsub.init()
	if err != nil {
		return err
	}

	if pubsub.Fixup == nil {
		return pubsub.listen()
	}
	return ReliableTask(pubsub.listen, pubsub.IsStop, pubsub.FixupMaxRetrySecond, pubsub.Fixup)
}

func (pubsub *PubSub[rMessage, sMessage]) listen() error {
	for !pubsub.isStop.Load() {
		message, err := pubsub.AdapterRecv()

		if pubsub.isStop.Load() {
			return nil
		}

		if err != nil {
			return err
		}

		pubsub.HandleRecv(message, nil)
	}
	return nil
}

func (pubsub *PubSub[rMessage, sMessage]) Send(message *sMessage) error {
	err := pubsub.init()
	if err != nil {
		return err
	}

	return pubsub.AdapterSend(message)
}

func (pubsub *PubSub[rMessage, sMessage]) IsStop() bool {
	return pubsub.isStop.Load()
}

func (pubsub *PubSub[rMessage, sMessage]) Stop() error {
	var empty *sMessage
	return pubsub.StopWithMessage(empty)
}

func (pubsub *PubSub[rMessage, sMessage]) StopWithMessage(message *sMessage) error {
	err := pubsub.init()
	if err != nil {
		return err
	}

	if pubsub.isStop.Load() {
		return nil
	}
	pubsub.isStop.Store(true)
	pubsub.Lifecycle.NotifyExit()
	err = pubsub.AdapterStop(message)
	if err != nil {
		return err
	}
	return nil
}

func (pubsub *PubSub[rMessage, sMessage]) PingPong(pp PingPong) error {
	err := pubsub.init()
	if err != nil {
		return err
	}

	err = pp.validate()
	if err != nil {
		return err
	}
	defer pubsub.Stop()

	if pubsub.Fixup == nil {
		return pp.Execute(pubsub.IsStop)
	}

	return ReliableTask(
		func() error {
			return pp.Execute(pubsub.IsStop)
		},
		pubsub.IsStop,
		pubsub.FixupMaxRetrySecond,
		pubsub.Fixup,
	)
}
