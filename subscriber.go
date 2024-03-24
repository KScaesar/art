package Artifex

import (
	"context"
	"sync"
	"sync/atomic"
)

type SubscriberFactory[rMessage any] interface {
	CreateSubscriber() (*Subscriber[rMessage], error)
}

type Subscriber[rMessage any] struct {
	HandleRecv       HandleFunc[rMessage]     // Must
	AdapterRecv      func() (rMessage, error) // Must
	AdapterStop      func() error             // Must
	Fixup            func() error
	MaxElapsedMinute int

	Identifier string
	Context    context.Context

	isStop atomic.Bool
	isInit atomic.Bool
	mu     sync.Mutex
}

func (sub *Subscriber[rMessage]) init() error {
	if sub.isInit.Load() {
		return nil
	}

	if sub.HandleRecv == nil || sub.AdapterStop == nil || sub.AdapterRecv == nil {
		return ErrorWrapWithMessage(ErrInvalidParameter, "subscriber")
	}

	if sub.Context == nil {
		sub.Context = context.Background()
	}

	sub.isInit.Store(true)
	return nil
}

func (sub *Subscriber[rMessage]) Serve() error {
	if !sub.isInit.Load() {
		sub.mu.Lock()
		err := sub.init()
		if err != nil {
			sub.mu.Unlock()
			return err
		}
		sub.mu.Unlock()
	}

	result := make(chan error, 1)

	go func() {
		if sub.Fixup == nil {
			result <- sub.serve()
		}
		ReliableTask(sub.serve, sub.IsStop, sub.Fixup, sub.MaxElapsedMinute)
		result <- nil
	}()

	return <-result
}

func (sub *Subscriber[rMessage]) serve() error {
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
		sub.mu.Lock()
		err := sub.init()
		if err != nil {
			sub.mu.Unlock()
			return err
		}
		sub.mu.Unlock()
	}

	if sub.isStop.Load() {
		return nil
	}
	err := sub.AdapterStop()
	if err != nil {
		return err
	}
	sub.isStop.Store(true)
	return nil
}
