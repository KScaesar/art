package Artifex

import (
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/constraints"
)

func NewGorillaSession[S constraints.Ordered, M any](
	conn *websocket.Conn,
	mux *MessageMux[S, M],
	cryptoKey []byte,
	messageConverter func(bMessage []byte, mKind int, parent *GorillaSession[S, M]) M,
	marshal Marshal,
) *GorillaSession[S, M] {
	return &GorillaSession[S, M]{
		conn:             conn,
		mux:              mux,
		cryptoKey:        cryptoKey,
		logger:           DefaultLogger(),
		messageConverter: messageConverter,
		marshal:          marshal,
	}
}

type GorillaSession[S constraints.Ordered, M any] struct {
	mutex            sync.Mutex
	conn             *websocket.Conn
	mux              *MessageMux[S, M]
	isStop           atomic.Bool
	cryptoKey        []byte
	logger           Logger
	pingpong         func() error
	messageConverter func(bMessage []byte, mKind int, parent *GorillaSession[S, M]) M
	marshal          Marshal
}

func (sess *GorillaSession[S, M]) Logger() Logger {
	return sess.logger
}

func (sess *GorillaSession[S, M]) SetLogger(logger Logger) {
	sess.logger = logger
}

func (sess *GorillaSession[S, M]) Connection() *websocket.Conn {
	return sess.conn
}

func (sess *GorillaSession[S, M]) Listen(crypto bool) error {
	result := make(chan error, 2)
	go func() {
		result <- sess.listen(crypto)
	}()
	go func() {
		result <- sess.pingpong()
	}()
	err := <-result
	if err != nil {
		sess.Disconnect()
	}
	return err
}

func (sess *GorillaSession[S, M]) listen(crypto bool) (err error) {
	for !sess.IsStop() {
		err = sess.ReceiveWithHandler(crypto, sess.mux.HandleMessage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sess *GorillaSession[S, M]) ReceiveWithHandler(crypto bool, handler MessageHandler[M]) error {
	logger := sess.logger

	if handler == nil {
		handler = sess.mux.HandleMessage
	}

	messageKind, bMessage, err := sess.conn.ReadMessage()
	if err != nil {
		Err := ConvertErrNetwork(err)
		logger.Error("read websocket: %v", Err)
		return Err
	}

	if crypto {
		bMessage, err = DecryptECB(sess.cryptoKey, bMessage)
		if err != nil {
			Err := ErrorWrapWithMessage(err, "decrypt websocket message")
			logger.Error("%v", Err)
			return Err
		}
	}

	message := sess.messageConverter(bMessage, messageKind, sess)

	return handler(message)
}

func (sess *GorillaSession[S, M]) Send(message any, crypto bool) error {
	sess.mutex.Lock()
	defer sess.mutex.Unlock()

	logger := sess.logger.WithCallDepth(1)

	bMessage, err := sess.marshal(message)
	if err != nil {
		logger.Error("%v", err)
		return err
	}

	if crypto {
		bMessage, err = EncryptECB(sess.cryptoKey, bMessage)
		if err != nil {
			Err := ErrorWrapWithMessage(err, "encrypt websocket message")
			logger.Error("%v", Err)
			return Err
		}
	}

	err = sess.conn.WriteMessage(websocket.BinaryMessage, bMessage)
	if err != nil {
		Err := ConvertErrNetwork(err)
		logger.Error("%v", Err)
		return Err
	}

	return nil
}

func (sess *GorillaSession[S, M]) WriteMessage(wsType int, bMessage []byte) error {
	sess.mutex.Lock()
	defer sess.mutex.Unlock()

	err := sess.conn.WriteMessage(wsType, bMessage)
	if err != nil {
		Err := ConvertErrNetwork(err)
		sess.logger.Error("%v", Err)
		return Err
	}
	return nil
}

func (sess *GorillaSession[S, M]) Disconnect() error {
	if sess.IsStop() {
		return nil
	}
	sess.isStop.Store(true)
	return sess.conn.Close()
}

func (sess *GorillaSession[S, M]) IsStop() bool {
	return sess.isStop.Load()
}

func (sess *GorillaSession[S, M]) EnableSendPingWaitPong(pongSubject S, handler MessageHandler[M], pongWaitSecond int, ping func(*GorillaSession[S, M]) error) {
	sendPing := func() error {
		return ping(sess)
	}

	waitPong := make(chan error, 1)

	sess.mux.RegisterHandler(pongSubject, func(dto M) error {
		waitPong <- handler(dto)
		return nil
	})

	sess.pingpong = func() error {
		return SendPingWaitPong(sendPing, waitPong, sess.IsStop, pongWaitSecond)
	}
}

func (sess *GorillaSession[S, M]) EnableWaitPingSendPong(pingSubject S, handler MessageHandler[M], pingWaitSecond int, pong func(client *GorillaSession[S, M]) error) {
	waitPing := make(chan error, 1)

	sendPong := func() error {
		return pong(sess)
	}

	sess.mux.RegisterHandler(pingSubject, func(dto M) error {
		waitPing <- handler(dto)
		return nil
	})

	sess.pingpong = func() error {
		return WaitPingSendPong(waitPing, sendPong, sess.IsStop, pingWaitSecond)
	}
}
