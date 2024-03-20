package Artifex

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/gookit/goutil/maputil"
	"golang.org/x/exp/constraints"
)

type AdapterFactory[Subject constraints.Ordered, rMessage, sMessage any] interface {
	CreateAdapter() (Adapter[Subject, rMessage, sMessage], error)
}

type Adapter[Subject constraints.Ordered, rMessage, sMessage any] struct {
	Recv func(parent *Session[Subject, rMessage, sMessage]) (rMessage, error) // Must
	Send func(message sMessage) error                                         // Must
	Stop func(message sMessage)                                               // Must

	// Lifecycle
	Lifecycle  Lifecycle[Subject, rMessage, sMessage]
	Identifier string                                // Option
	Context    context.Context                       // Option
	PingPong   PingPong[Subject, rMessage, sMessage] // Option
}

func NewSession[S constraints.Ordered, rM, sM any](recvMux *Mux[S, rM], factory AdapterFactory[S, rM, sM]) (sess *Session[S, rM, sM], err error) {
	if recvMux == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "session adapter: mux is nil")
	}

	adapter, err := factory.CreateAdapter()
	if err != nil {
		return nil, err
	}

	if adapter.Stop == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "session adapter: stop is nil")
	}
	var empty sM
	defer func() {
		if err != nil {
			adapter.Stop(empty)
		}
	}()

	if adapter.Send == nil && adapter.Recv == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "session adapter: send and recv are empty")
	}

	ctx := context.Background()
	if adapter.Context != nil {
		ctx = adapter.Context
	}

	session := &Session[S, rM, sM]{
		AppData: make(map[string]any),
		// pingpong:  func() error { return nil },
		notifyAll: make([]chan error, 0),

		recvMux:    recvMux,
		Identifier: adapter.Identifier,
		Context:    ctx,

		recv:      adapter.Recv,
		send:      adapter.Send,
		stop:      adapter.Stop,
		lifecycle: adapter.Lifecycle,
	}

	err = session.lifecycle.Spawn(session)
	if err != nil {
		return nil, err
	}

	go func() {
		notify := session.Notify()
		select {
		case <-notify:
			session.lifecycle.Exit(session)
		}
	}()

	return session, nil
}

type Session[Subject constraints.Ordered, rMessage, sMessage any] struct {
	mu        sync.RWMutex
	AppData   maputil.Data // AppData not concurrency safe
	pingpong  PingPong[Subject, rMessage, sMessage]
	isStop    atomic.Bool
	isListen  atomic.Bool
	notifyAll []chan error
	factory   AdapterFactory[Subject, rMessage, sMessage]

	recvMux    *Mux[Subject, rMessage]
	Identifier string
	Context    context.Context

	recv      func(parent *Session[Subject, rMessage, sMessage]) (rMessage, error) // Must
	send      func(message sMessage) error                                         // Must
	stop      func(message sMessage)                                               // Must
	lifecycle Lifecycle[Subject, rMessage, sMessage]
}

func (sess *Session[Subject, rMessage, sMessage]) SelfRepair() error {
	sess.mu.Lock()
	defer sess.mu.Unlock()

	adapter, err := sess.factory.CreateAdapter()
	if err != nil {
		return err
	}

	sess.recv = adapter.Recv
	sess.send = adapter.Send
	sess.stop = adapter.Stop
	sess.isStop.Store(false)
	return nil
}

func (sess *Session[Subject, rMessage, sMessage]) Listen() error {
	sess.mu.RLock()
	defer sess.mu.RUnlock()

	if sess.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex session")
	}

	if sess.isListen.Load() {
		return nil
	}
	sess.isListen.Store(true)

	result := make(chan error, 2)

	go func() {
		result <- sess.listen()
	}()

	go func() {
		result <- sess.pingpong.Run(sess)
	}()

	err := <-result
	sess.Stop()
	go func() {
		sess.mu.Lock()
		defer sess.mu.Unlock()
		for _, notify := range sess.notifyAll {
			notify <- err
			close(notify)
		}
		sess.notifyAll = make([]chan error, 0)
	}()
	return err
}

func (sess *Session[Subject, rMessage, sMessage]) listen() error {
	for !sess.isStop.Load() {
		err := sess.Recv()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sess *Session[Subject, rMessage, sMessage]) Recv() error {
	if sess.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex session")
	}

	message, err := sess.recv(sess)

	if sess.isStop.Load() {
		return nil
	}

	if err != nil {
		return err
	}

	return sess.recvMux.HandleMessage(message, nil)
}

func (sess *Session[Subject, rMessage, sMessage]) Send(message sMessage) error {
	if sess.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artist session")
	}
	return sess.send(message)
}

func (sess *Session[Subject, rMessage, sMessage]) Stop() {
	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.isStop.Load() {
		return
	}
	sess.isStop.Store(true)
	sess.isListen.Store(false)
	var empty sMessage
	sess.stop(empty)
}

func (sess *Session[Subject, rMessage, sMessage]) StopWithMessage(message sMessage) {
	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.isStop.Load() {
		return
	}
	sess.isStop.Store(true)
	sess.isListen.Store(false)
	sess.stop(message)
}

func (sess *Session[Subject, rMessage, sMessage]) IsStop() bool {
	return sess.isStop.Load()
}

// Notify returns a channel for receiving the result of the Session.Listen.
// If the session is already closed,
// it returns a channel containing an error message indicating the session is closed.
// Once notified, the channel will be closed immediately.
func (sess *Session[Subject, rMessage, sMessage]) Notify() <-chan error {
	sess.mu.Lock()
	defer sess.mu.Unlock()

	ch := make(chan error, 1)
	if sess.isStop.Load() {
		ch <- ErrorWrapWithMessage(ErrClosed, "Artist session")
		close(ch)
		return ch
	}

	sess.notifyAll = append(sess.notifyAll, ch)
	return ch
}
