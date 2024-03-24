package Artifex

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/gookit/goutil/maputil"
	"golang.org/x/exp/constraints"
)

type SessionFactory[Subject constraints.Ordered, rMessage, sMessage any] interface {
	CreateSession() (*Session[Subject, rMessage, sMessage], error)
}

type Session[Subject constraints.Ordered, rMessage, sMessage any] struct {
	Mux         *Mux[Subject, rMessage]  // Must
	AdapterRecv func() (rMessage, error) // Must
	AdapterSend func(sMessage) error     // Must
	AdapterStop func(sMessage) error     // Must

	Identifier string
	Context    context.Context
	AppData    maputil.Data
	Locker     sync.RWMutex

	PingPong PingPong

	// Lifecycle define a management mechanism when session creation and session end.
	Lifecycle Lifecycle

	// Use ReliableTask, when adapter encounters an error, it can Fixup error.
	// This makes sure that the Session keeps going without any problems until we decide to Stop it.
	Fixup func() error

	isStop    atomic.Bool
	isInit    atomic.Bool
	notifyAll []chan error
}

func (sess *Session[Subject, rMessage, sMessage]) init() (err error) {
	if sess.isInit.Load() {
		return nil
	}

	err = sess.Lifecycle.Execute()
	if err != nil {
		return err
	}

	if sess.Mux == nil {
		return ErrorWrapWithMessage(ErrInvalidParameter, "session: mux is nil")
	}

	if sess.AdapterStop == nil || sess.AdapterSend == nil || sess.AdapterRecv == nil {
		return ErrorWrapWithMessage(ErrInvalidParameter, "session: Adapter is nil")
	}

	err = sess.PingPong.validate()
	if err != nil {
		return err
	}

	if sess.AppData == nil {
		sess.AppData = make(maputil.Data)
	}

	if sess.Context == nil {
		sess.Context = context.Background()
	}

	sess.isInit.Store(true)
	return nil
}

func (sess *Session[Subject, rMessage, sMessage]) Listen() error {
	if sess.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex session")
	}
	defer sess.Stop()

	if !sess.isInit.Load() {
		sess.Locker.Lock()
		err := sess.init()
		if err != nil {
			sess.Locker.Unlock()
			return err
		}
		sess.Locker.Unlock()
	}

	result := make(chan error, 2)

	go func() {
		if sess.Fixup == nil {
			result <- sess.listen()
		}
		ReliableTask(sess.listen, sess.IsStop, sess.Fixup)
		result <- nil
	}()

	go func() {
		if !sess.PingPong.Enable {
			return
		}
		if sess.Fixup == nil {
			result <- sess.PingPong.Execute(sess.IsStop)
		}
		ReliableTask(
			func() error {
				return sess.PingPong.Execute(sess.IsStop)
			},
			sess.IsStop,
			sess.Fixup,
		)
	}()

	err := <-result
	go func() {
		sess.Locker.Lock()
		defer sess.Locker.Unlock()
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
		message, err := sess.AdapterRecv()

		if sess.isStop.Load() {
			return nil
		}

		if err != nil {
			return err
		}

		sess.Mux.HandleMessage(message, nil)
	}
	return nil
}

func (sess *Session[Subject, rMessage, sMessage]) Send(message sMessage) error {
	if !sess.isInit.Load() {
		sess.Locker.Lock()
		err := sess.init()
		if err != nil {
			sess.Locker.Unlock()
			return err
		}
		sess.Locker.Unlock()
	}

	if sess.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex session")
	}
	if sess.Fixup == nil {
		return sess.AdapterSend(message)
	}
	ReliableTask(
		func() error {
			return sess.AdapterSend(message)
		},
		sess.IsStop,
		sess.Fixup,
	)
	return nil
}

func (sess *Session[Subject, rMessage, sMessage]) Stop() {
	if !sess.isInit.Load() {
		sess.Locker.Lock()
		err := sess.init()
		if err != nil {
			sess.Locker.Unlock()
			return
		}
		sess.Locker.Unlock()
	}

	sess.Locker.Lock()
	defer sess.Locker.Unlock()

	if sess.isStop.Load() {
		return
	}
	sess.isStop.Store(true)
	var empty sMessage
	sess.AdapterStop(empty)
	sess.Lifecycle.NotifyExit()
}

func (sess *Session[Subject, rMessage, sMessage]) StopWithMessage(message sMessage) {
	if !sess.isInit.Load() {
		sess.Locker.Lock()
		err := sess.init()
		if err != nil {
			sess.Locker.Unlock()
			return
		}
		sess.Locker.Unlock()
	}

	sess.Locker.Lock()
	defer sess.Locker.Unlock()

	if sess.isStop.Load() {
		return
	}
	sess.isStop.Store(true)
	sess.AdapterStop(message)
}

func (sess *Session[Subject, rMessage, sMessage]) IsStop() bool {
	return sess.isStop.Load()
}

// NotifyStop returns a channel for receiving the result of the Session.Listen.
// If the Session is already Stop,
// it returns a channel containing an error message indicating the session is closed.
//
// Once notified, the channel will be closed immediately.
func (sess *Session[Subject, rMessage, sMessage]) NotifyStop() <-chan error {
	sess.Locker.Lock()
	defer sess.Locker.Unlock()

	if sess.notifyAll == nil {
		sess.notifyAll = make([]chan error, 0)
	}

	ch := make(chan error, 1)
	if sess.isStop.Load() {
		ch <- ErrorWrapWithMessage(ErrClosed, "Artifex session")
		close(ch)
		return ch
	}

	sess.notifyAll = append(sess.notifyAll, ch)
	return ch
}
