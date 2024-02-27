package Artifex

import (
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/constraints"
)

func NewWebsocketSession[S constraints.Ordered, M any](
	conn *websocket.Conn,
	mux *MessageMux[S, M],
	cryptoKey []byte,
	messageConverter func(bMessage []byte, mKind int, parent *WebsocketSession[S, M]) M,
	marshal Marshal,
) *WebsocketSession[S, M] {
	return &WebsocketSession[S, M]{
		conn:             conn,
		mux:              mux,
		cryptoKey:        cryptoKey,
		logger:           DefaultLogger(),
		messageConverter: messageConverter,
		marshal:          marshal,
	}
}

type WebsocketSession[S constraints.Ordered, M any] struct {
	Mutex            sync.Mutex
	conn             *websocket.Conn
	mux              *MessageMux[S, M]
	isStop           atomic.Bool
	cryptoKey        []byte
	logger           Logger
	pingpong         func() error
	messageConverter func(bMessage []byte, mKind int, parent *WebsocketSession[S, M]) M
	marshal          Marshal
}

func (sess *WebsocketSession[S, M]) Logger() Logger {
	return sess.logger
}

func (sess *WebsocketSession[S, M]) SetLogger(logger Logger) {
	sess.logger = logger
}

func (sess *WebsocketSession[S, M]) Connection() *websocket.Conn {
	return sess.conn
}

func (sess *WebsocketSession[S, M]) Listen(crypto bool) error {
	result := make(chan error, 2)
	go func() {
		result <- sess.listen(crypto)
	}()
	go func() {
		result <- sess.pingpong()
	}()
	return <-result
}

func (sess *WebsocketSession[S, M]) listen(crypto bool) (err error) {
	for !sess.IsStop() {
		err = sess.Receive(crypto)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sess *WebsocketSession[S, M]) Receive(crypto bool) error {
	logger := sess.logger

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

	return sess.mux.HandleMessage(message)
}

func (sess *WebsocketSession[S, M]) Send(message any, crypto bool) error {
	sess.Mutex.Lock()
	defer sess.Mutex.Unlock()

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

func (sess *WebsocketSession[S, M]) Disconnect() error {
	if sess.IsStop() {
		return nil
	}
	sess.isStop.Store(true)
	return sess.conn.Close()
}

func (sess *WebsocketSession[S, M]) IsStop() bool {
	return sess.isStop.Load()
}

func (sess *WebsocketSession[S, M]) EnableSendPingWaitPong(pongSubject S, handler MessageHandler[M], pongWaitSecond int, ping func(*WebsocketSession[S, M]) error) {
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

func (sess *WebsocketSession[S, M]) EnableWaitPingSendPong(pingSubject S, handler MessageHandler[M], pingWaitSecond int, pong func(client *WebsocketSession[S, M]) error) {
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
