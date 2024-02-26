package xWebsocket

import (
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/constraints"

	"github.com/KScaesar/Artifex"
)

func NewClient[S constraints.Ordered, M any](
	conn *websocket.Conn,
	mux *Artifex.MessageMux[S, M],
	cryptoKey []byte,
	messageFactory func(bMessage []byte, mKind int) M,
	marshal Artifex.Marshal,
) *Client[S, M] {

	return &Client[S, M]{
		conn:           conn,
		mux:            mux,
		cryptoKey:      cryptoKey,
		Logger:         Artifex.DefaultLogger(),
		messageFactory: messageFactory,
		marshal:        marshal,
	}
}

type Client[S constraints.Ordered, M any] struct {
	mu             sync.Mutex
	conn           *websocket.Conn
	mux            *Artifex.MessageMux[S, M]
	isStop         atomic.Bool
	cryptoKey      []byte
	Logger         Artifex.Logger
	pingpong       func() error
	messageFactory func(bMessage []byte, mKind int) M
	marshal        Artifex.Marshal
}

func (client *Client[S, M]) Listen(crypto bool) error {
	result := make(chan error, 2)
	go func() {
		result <- client.listen(crypto)
	}()
	go func() {
		result <- client.pingpong()
	}()
	return <-result
}

func (client *Client[S, M]) listen(crypto bool) (err error) {
	for !client.IsStop() {
		err = client.HandleResponse(crypto)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *Client[S, M]) HandleResponse(crypto bool) error {
	logger := client.Logger

	messageKind, bMessage, err := client.conn.ReadMessage()
	if err != nil {
		Err := Artifex.ConvertErrNetwork(err)
		logger.Error("read websocket: %v", Err)
		return Err
	}

	if crypto {
		bMessage, err = Artifex.DecryptECB(client.cryptoKey, bMessage)
		if err != nil {
			Err := Artifex.ErrorWrapWithMessage(err, "decrypt websocket message")
			logger.Error("%v", Err)
			return Err
		}
	}

	message := client.messageFactory(bMessage, messageKind)

	return client.mux.HandleMessage(message)
}

func (client *Client[S, M]) Invoke(message any, crypto bool) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	logger := client.Logger.WithCallDepth(1)

	bMessage, err := client.marshal(message)
	if err != nil {
		logger.Error("%v", err)
		return err
	}

	if crypto {
		bMessage, err = Artifex.EncryptECB(client.cryptoKey, bMessage)
		if err != nil {
			Err := Artifex.ErrorWrapWithMessage(err, "encrypt websocket message")
			logger.Error("%v", Err)
			return Err
		}
	}

	err = client.conn.WriteMessage(websocket.BinaryMessage, bMessage)
	if err != nil {
		Err := Artifex.ConvertErrNetwork(err)
		logger.Error("%v", Err)
		return Err
	}

	return nil
}

func (client *Client[S, M]) Disconnect() error {
	if client.IsStop() {
		return nil
	}
	client.isStop.Store(true)
	return client.conn.Close()
}

func (client *Client[S, M]) IsStop() bool {
	return client.isStop.Load()
}

func (client *Client[S, M]) EnableSendPingWaitPong(pongSubject S, handler Artifex.MessageHandler[M], pongWaitSecond int, ping func(*Client[S, M]) error) {
	sendPing := func() error {
		client.mu.Lock()
		defer client.mu.Unlock()
		return ping(client)
	}

	waitPong := make(chan error, 1)

	client.mux.RegisterHandler(pongSubject, func(dto M) error {
		waitPong <- handler(dto)
		return nil
	})

	client.pingpong = func() error {
		return Artifex.SendPingWaitPong(sendPing, waitPong, client.IsStop, pongWaitSecond)
	}
}

func (client *Client[S, M]) EnableWaitPingSendPong(pingSubject S, handler Artifex.MessageHandler[M], pingWaitSecond int, pong func(client *Client[S, M]) error) {
	waitPing := make(chan error, 1)

	sendPong := func() error {
		client.mu.Lock()
		defer client.mu.Unlock()
		return pong(client)
	}

	client.mux.RegisterHandler(pingSubject, func(dto M) error {
		waitPing <- handler(dto)
		return nil
	})

	client.pingpong = func() error {
		return Artifex.WaitPingSendPong(waitPing, sendPong, client.IsStop, pingWaitSecond)
	}
}
