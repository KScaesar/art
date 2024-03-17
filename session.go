package Artifex

import (
	"bytes"
	"context"
	"sync/atomic"

	"golang.org/x/exp/constraints"
)

type AdapterRecvFunc[Subject constraints.Ordered, rMessage, sMessage any] func(parent *Session[Subject, rMessage, sMessage]) (rMessage, error)
type AdapterSendFunc[sMessage any] func(session Logger, message sMessage) error
type AdapterStopFunc[sMessage any] func(session Logger, message sMessage)

type Adapter[Subject constraints.Ordered, rMessage, sMessage any] struct {
	Identifier string
	Context    context.Context
	Recv       AdapterRecvFunc[Subject, rMessage, sMessage]
	Send       AdapterSendFunc[sMessage]
	Stop       AdapterStopFunc[sMessage]
	Logger     Logger
}

func NewSession[S constraints.Ordered, rM, sM any](recvMux *MessageMux[S, rM], adapter Adapter[S, rM, sM]) (*Session[S, rM, sM], error) {
	if recvMux == nil || adapter.Stop == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "session adapter: mux or stop is empty")
	}

	if adapter.Send == nil && adapter.Recv == nil {
		return nil, ErrorWrapWithMessage(ErrInvalidParameter, "session adapter: send and recv are empty")
	}

	var sessId string
	if adapter.Identifier == "" {
		builder := bytes.NewBuffer(make([]byte, 0, 13))
		builder.WriteString(GenerateRandomCode(6))
		builder.WriteString("-")
		builder.WriteString(GenerateRandomCode(6))
		sessId = builder.String()
	}

	logger := DefaultLogger()
	if adapter.Logger != nil {
		logger = adapter.Logger
	}
	logger.WithSessionId(sessId)

	return &Session[S, rM, sM]{
		Identifier: sessId,
		keys:       make(map[string]interface{}),
		recvMux:    recvMux,
		recv:       adapter.Recv,
		send:       adapter.Send,
		stop:       adapter.Stop,
		logger:     logger,
	}, nil
}

type Session[Subject constraints.Ordered, rMessage, sMessage any] struct {
	Identifier     string
	keys           map[string]interface{}
	pingpong       func() error
	enablePingPong atomic.Bool
	isStop         atomic.Bool

	recvMux *MessageMux[Subject, rMessage]
	recv    AdapterRecvFunc[Subject, rMessage, sMessage]
	send    AdapterSendFunc[sMessage]
	stop    AdapterStopFunc[sMessage]
	logger  Logger
}

func (sess *Session[Subject, rMessage, sMessage]) Logger() Logger {
	return sess.logger
}

func (sess *Session[Subject, rMessage, sMessage]) Listen() error {
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

func (sess *Session[Subject, rMessage, sMessage]) listen() error {
	for !sess.isStop.Load() {
		if err := sess.Recv(); err != nil {
			return err
		}
	}
	return nil
}

func (sess *Session[Subject, rMessage, sMessage]) Recv() error {
	message, err := sess.recv(sess)
	if err != nil {
		sess.logger.Error("recv message fail: %v", err)
		return err
	}
	return sess.recvMux.HandleMessage(message)
}

func (sess *Session[Subject, rMessage, sMessage]) Send(message sMessage) error {
	return sess.send(sess.logger, message)
}

func (sess *Session[Subject, rMessage, sMessage]) Stop() {
	sess.StopWithMessage(nil)
}

func (sess *Session[Subject, rMessage, sMessage]) StopWithMessage(message sMessage) {
	if sess.isStop.Load() {
		return
	}
	sess.isStop.Store(true)
	sess.stop(sess.logger, message)
}

func (sess *Session[Subject, rMessage, sMessage]) IsStop() bool {
	return sess.isStop.Load()
}

func (sess *Session[Subject, rMessage, sMessage]) SendPingWaitPong(pongSubject Subject, pongWaitSecond int, ping, pong func(sess *Session[Subject, rMessage, sMessage]) error) {
	sess.enablePingPong.Store(true)
	waitPong := make(chan error, 1)

	sess.recvMux.RegisterHandler(pongSubject, func(message rMessage) error {
		waitPong <- pong(sess)
		return nil
	})

	sendPing := func() error { return ping(sess) }

	sess.pingpong = func() error {
		return SendPingWaitPong(sendPing, waitPong, sess.IsStop, pongWaitSecond)
	}
}

func (sess *Session[Subject, rMessage, sMessage]) WaitPingSendPong(pingSubject Subject, pingWaitSecond int, ping, pong func(sess *Session[Subject, rMessage, sMessage]) error) {
	sess.enablePingPong.Store(true)
	waitPing := make(chan error, 1)

	sess.recvMux.RegisterHandler(pingSubject, func(message rMessage) error {
		waitPing <- ping(sess)
		return nil
	})

	sendPong := func() error { return pong(sess) }

	sess.pingpong = func() error {
		return WaitPingSendPong(waitPing, sendPong, sess.IsStop, pingWaitSecond)
	}
}
