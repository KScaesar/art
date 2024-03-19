package Artifex

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"

	"github.com/gookit/goutil/maputil"
	"golang.org/x/exp/constraints"
)

type AdapterRecvFunc[Subject constraints.Ordered, rMessage, sMessage any] func(parent *Session[Subject, rMessage, sMessage]) (rMessage, error)
type AdapterSendFunc[sMessage any] func(message sMessage) error
type AdapterStopFunc[sMessage any] func(message sMessage)

type NewAdapterFunc[Subject constraints.Ordered, rMessage, sMessage any] func() (Adapter[Subject, rMessage, sMessage], error)

type Adapter[Subject constraints.Ordered, rMessage, sMessage any] struct {
	Recv AdapterRecvFunc[Subject, rMessage, sMessage] // Must
	Send AdapterSendFunc[sMessage]                    // Must
	Stop AdapterStopFunc[sMessage]                    // Must

	Identifier string          // Option
	Context    context.Context // Option
	Logger     Logger          // Option
}

func NewSession[S constraints.Ordered, rM, sM any](recvMux *Mux[S, rM], adapter Adapter[S, rM, sM]) (*Session[S, rM, sM], error) {
	if recvMux == nil || adapter.Stop == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "session adapter: mux or stop is empty")
	}

	if adapter.Send == nil && adapter.Recv == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "session adapter: send and recv are empty")
	}

	var sessId string
	builder := bytes.NewBuffer(make([]byte, 0, 13))
	builder.WriteString(GenerateRandomCode(6))
	builder.WriteString("-")
	builder.WriteString(GenerateRandomCode(6))
	sessId = builder.String()

	ctx := context.Background()
	if adapter.Context != nil {
		ctx = adapter.Context
	}

	logger := DefaultLogger()
	if adapter.Logger != nil {
		logger = adapter.Logger
	}
	logger = logger.WithSessionId(sessId)

	return &Session[S, rM, sM]{
		Keys:      make(map[string]any),
		pingpong:  func() error { return nil },
		notifyAll: make([]chan error, 0),

		recvMux:    recvMux,
		Identifier: sessId,
		Context:    ctx,
		logger:     logger,

		recv: adapter.Recv,
		send: adapter.Send,
		stop: adapter.Stop,
	}, nil
}

type Session[Subject constraints.Ordered, rMessage, sMessage any] struct {
	mu             sync.RWMutex
	Keys           maputil.Data
	pingpong       func() error
	enablePingPong atomic.Bool
	isStop         atomic.Bool
	isListen       atomic.Bool
	notifyAll      []chan error

	recvMux    *Mux[Subject, rMessage]
	Identifier string
	Context    context.Context
	logger     Logger

	recv AdapterRecvFunc[Subject, rMessage, sMessage]
	send AdapterSendFunc[sMessage]
	stop AdapterStopFunc[sMessage]
}

func (sess *Session[Subject, rMessage, sMessage]) Logger() Logger {
	return sess.logger
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

	if sess.enablePingPong.Load() {
		go func() {
			result <- sess.pingpong()
		}()
	}

	err := <-result
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
		sess.logger.Error("recv message fail: %v", err)
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

// SendPingWaitPong sends a ping message and waits for a corresponding pong message.
// This function can be used in configuration by the Artist.SetSpawnHandler or Roamer.SpawnHandlers
func (sess *Session[Subject, rMessage, sMessage]) SendPingWaitPong(pongSubject Subject, pongWaitSecond int, ping, pong func(sess *Session[Subject, rMessage, sMessage]) error) {
	sess.enablePingPong.Store(true)
	waitPong := make(chan error, 1)

	sess.recvMux.Handler(pongSubject, func(message rMessage, _ *RouteParam) error {
		waitPong <- pong(sess)
		return nil
	})

	sendPing := func() error { return ping(sess) }

	sess.pingpong = func() error {
		return SendPingWaitPong(sendPing, waitPong, sess.IsStop, pongWaitSecond)
	}
}

// WaitPingSendPong waits for a ping message and response a corresponding pong message.
// This function can be used in configuration by the Artist.SetSpawnHandler or Roamer.SpawnHandlers
func (sess *Session[Subject, rMessage, sMessage]) WaitPingSendPong(pingSubject Subject, pingWaitSecond int, ping, pong func(sess *Session[Subject, rMessage, sMessage]) error) {
	sess.enablePingPong.Store(true)
	waitPing := make(chan error, 1)

	sess.recvMux.Handler(pingSubject, func(message rMessage, _ *RouteParam) error {
		waitPing <- ping(sess)
		return nil
	})

	sendPong := func() error { return pong(sess) }

	sess.pingpong = func() error {
		return WaitPingSendPong(waitPing, sendPong, sess.IsStop, pingWaitSecond)
	}
}
