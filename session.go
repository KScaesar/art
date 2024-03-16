package Artifex

import (
	"context"
	"sync"
	"sync/atomic"

	"golang.org/x/exp/constraints"
)

type RecvFunc[Subject constraints.Ordered, Message any] func(parent *Session[Subject, Message]) (Message, error)
type SendFunc func(msgObj any) error
type StopFunc func()

type SessionParam[S constraints.Ordered, M any] struct {
	Context context.Context
	Mux     *MessageMux[S, M]
	Recv    RecvFunc[S, M]
	Send    SendFunc
	Stop    StopFunc
	Logger  Logger
}

func NewSession[S constraints.Ordered, M any](param SessionParam[S, M]) (*Session[S, M], error) {
	if param.Mux == nil || param.Stop == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "Session param: mux or stop is empty")
	}

	if param.Send == nil && param.Recv == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "Session param: send and recv are empty")
	}

	ctx := context.Background()
	if param.Context != nil {
		ctx = param.Context
	}

	logger := DefaultLogger()
	if param.Logger != nil {
		logger = param.Logger
	}

	return &Session[S, M]{
		context: ctx,
		mux:     param.Mux,
		recv:    param.Recv,
		send:    param.Send,
		stop:    param.Stop,
		logger:  logger,
	}, nil
}

type Session[S constraints.Ordered, M any] struct {
	context context.Context

	mutex          sync.Mutex
	pingpong       func() error
	enablePingPong atomic.Bool
	isStop         atomic.Bool

	mux    *MessageMux[S, M]
	recv   RecvFunc[S, M]
	send   SendFunc
	stop   StopFunc
	logger Logger
}

func (sess *Session[S, M]) Logger() Logger {
	return sess.logger
}

func (sess *Session[S, M]) Listen() error {
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
	if err != nil {
		sess.Stop()
	}
	return err
}

func (sess *Session[S, M]) listen() error {
	for !sess.IsStop() {
		if err := sess.Recv(); err != nil {
			return err
		}
	}
	return nil
}

func (sess *Session[S, M]) Recv() error {
	message, err := sess.recv(sess)
	if err != nil {
		sess.logger.Error("recv message fail: %v", err)
		return err
	}
	return sess.mux.HandleMessage(message)
}

func (sess *Session[S, M]) Send(message any) error {
	sess.mutex.Lock()
	defer sess.mutex.Unlock()
	return sess.send(message)
}

func (sess *Session[S, M]) SendWithoutLock(message any) error {
	return sess.send(message)
}

func (sess *Session[S, M]) Stop() {
	if sess.IsStop() {
		return
	}
	sess.isStop.Store(true)
	sess.stop()
}

func (sess *Session[S, M]) IsStop() bool {
	return sess.isStop.Load()
}

func (sess *Session[S, M]) SendPingWaitPong(pongSubject S, pongWaitSecond int, ping, pong func(context.Context, Logger) error) {
	sess.enablePingPong.Store(true)
	waitPong := make(chan error, 1)

	sess.mux.RegisterHandler(pongSubject, func(message M) error {
		waitPong <- pong(sess.context, sess.logger)
		return nil
	})

	sendPing := func() error { return ping(sess.context, sess.logger) }

	sess.pingpong = func() error {
		return SendPingWaitPong(sendPing, waitPong, sess.IsStop, pongWaitSecond)
	}
}

func (sess *Session[S, M]) WaitPingSendPong(pingSubject S, pingWaitSecond int, ping, pong func(context.Context, Logger) error) {
	sess.enablePingPong.Store(true)
	waitPing := make(chan error, 1)

	sess.mux.RegisterHandler(pingSubject, func(message M) error {
		waitPing <- ping(sess.context, sess.logger)
		return nil
	})

	sendPong := func() error { return pong(sess.context, sess.logger) }

	sess.pingpong = func() error {
		return WaitPingSendPong(waitPing, sendPong, sess.IsStop, pingWaitSecond)
	}
}
