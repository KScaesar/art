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
	messageFactory func(bMessage []byte, mKind int, parent *WebsocketSession[S, M]) M,
	marshal Marshal,
) *WebsocketSession[S, M] {
	return &WebsocketSession[S, M]{
		conn:           conn,
		mux:            mux,
		cryptoKey:      cryptoKey,
		logger:         DefaultLogger(),
		messageFactory: messageFactory,
		marshal:        marshal,
	}
}

type WebsocketSession[S constraints.Ordered, M any] struct {
	mu             sync.Mutex
	conn           *websocket.Conn
	mux            *MessageMux[S, M]
	isStop         atomic.Bool
	cryptoKey      []byte
	logger         Logger
	pingpong       func() error
	messageFactory func(bMessage []byte, mKind int, parent *WebsocketSession[S, M]) M
	marshal        Marshal
}

func (session *WebsocketSession[S, M]) SetLogger(logger Logger) {
	session.logger = logger
}

func (session *WebsocketSession[S, M]) Connection() *websocket.Conn {
	return session.conn
}

func (session *WebsocketSession[S, M]) Listen(crypto bool) error {
	result := make(chan error, 2)
	go func() {
		result <- session.listen(crypto)
	}()
	go func() {
		result <- session.pingpong()
	}()
	return <-result
}

func (session *WebsocketSession[S, M]) listen(crypto bool) (err error) {
	for !session.IsStop() {
		err = session.HandleResponse(crypto)
		if err != nil {
			return err
		}
	}
	return nil
}

func (session *WebsocketSession[S, M]) HandleResponse(crypto bool) error {
	logger := session.logger

	messageKind, bMessage, err := session.conn.ReadMessage()
	if err != nil {
		Err := ConvertErrNetwork(err)
		logger.Error("read websocket: %v", Err)
		return Err
	}

	if crypto {
		bMessage, err = DecryptECB(session.cryptoKey, bMessage)
		if err != nil {
			Err := ErrorWrapWithMessage(err, "decrypt websocket message")
			logger.Error("%v", Err)
			return Err
		}
	}

	message := session.messageFactory(bMessage, messageKind, session)

	return session.mux.HandleMessage(message)
}

func (session *WebsocketSession[S, M]) Invoke(message any, crypto bool) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	logger := session.logger.WithCallDepth(1)

	bMessage, err := session.marshal(message)
	if err != nil {
		logger.Error("%v", err)
		return err
	}

	if crypto {
		bMessage, err = EncryptECB(session.cryptoKey, bMessage)
		if err != nil {
			Err := ErrorWrapWithMessage(err, "encrypt websocket message")
			logger.Error("%v", Err)
			return Err
		}
	}

	err = session.conn.WriteMessage(websocket.BinaryMessage, bMessage)
	if err != nil {
		Err := ConvertErrNetwork(err)
		logger.Error("%v", Err)
		return Err
	}

	return nil
}

func (session *WebsocketSession[S, M]) Disconnect() error {
	if session.IsStop() {
		return nil
	}
	session.isStop.Store(true)
	return session.conn.Close()
}

func (session *WebsocketSession[S, M]) IsStop() bool {
	return session.isStop.Load()
}

func (session *WebsocketSession[S, M]) EnableSendPingWaitPong(pongSubject S, handler MessageHandler[M], pongWaitSecond int, ping func(*WebsocketSession[S, M]) error) {
	sendPing := func() error {
		return ping(session)
	}

	waitPong := make(chan error, 1)

	session.mux.RegisterHandler(pongSubject, func(dto M) error {
		waitPong <- handler(dto)
		return nil
	})

	session.pingpong = func() error {
		return SendPingWaitPong(sendPing, waitPong, session.IsStop, pongWaitSecond)
	}
}

func (session *WebsocketSession[S, M]) EnableWaitPingSendPong(pingSubject S, handler MessageHandler[M], pingWaitSecond int, pong func(client *WebsocketSession[S, M]) error) {
	waitPing := make(chan error, 1)

	sendPong := func() error {
		return pong(session)
	}

	session.mux.RegisterHandler(pingSubject, func(dto M) error {
		waitPing <- handler(dto)
		return nil
	})

	session.pingpong = func() error {
		return WaitPingSendPong(waitPing, sendPong, session.IsStop, pingWaitSecond)
	}
}
