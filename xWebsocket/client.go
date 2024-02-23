package xWebsocket

import (
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"github.com/KScaesar/Artifex"
)

func NewClient(conn *websocket.Conn, mux *WebsocketMux, cryptoKey []byte, pongWaitSecond int) *Client {
	client := &Client{
		conn:           conn,
		Mux:            mux,
		cryptoKey:      cryptoKey,
		Logger:         Artifex.DefaultLogger(),
		pongHandler:    func(*Message) error { return nil },
		pongWaitSecond: pongWaitSecond,
	}
	return client
}

type Client struct {
	mu             sync.Mutex
	conn           *websocket.Conn
	Mux            *WebsocketMux
	isStop         atomic.Bool
	cryptoKey      []byte
	Logger         Artifex.Logger
	pongHandler    WebsocketHandler
	pongWaitSecond int

	Marshal Artifex.Marshal
}

func (client *Client) Listen(crypto bool) error {
	result := make(chan error, 2)
	go func() {
		result <- client.listen(crypto)
	}()
	go func() {
		result <- client.pingpong(client.pongWaitSecond)
	}()
	return <-result
}

func (client *Client) listen(crypto bool) (err error) {
	for !client.IsStop() {
		err = client.HandleResponse(crypto)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *Client) HandleResponse(crypto bool) error {
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

	message := NewMessage(bMessage, messageKind)
	logger.Info("receive websocket subject=%v", message.Subject)

	err = client.Mux.HandleMessageWithoutMutex(message)
	if err != nil {
		Err := Artifex.ErrorWrapWithMessage(err, "hande websocket subject=%v fail", message.Subject)
		logger.Error("%v", Err)
		return Err
	}
	logger.Debug("hande websocket subject=%v success", message.Subject)

	if message.ErrResponseResult != nil {
		return message.ErrResponseResult
	}

	return nil
}

func (client *Client) Invoke(message any, crypto bool) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	logger := client.Logger.WithCallDepth(1)

	bMessage, err := client.Marshal(message)
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

func (client *Client) Disconnect() error {
	if client.IsStop() {
		return nil
	}
	client.isStop.Store(true)
	return client.conn.Close()
}

func (client *Client) IsStop() bool {
	return client.isStop.Load()
}

func (client *Client) SetPongHandler(handler WebsocketHandler) {
	client.pongHandler = handler
}

func (client *Client) pingpong(pongWaitSecond int) error {
	sendPing := func() error {
		client.mu.Lock()
		defer client.mu.Unlock()
		return client.conn.WriteMessage(websocket.PingMessage, nil)
	}

	waitPong := make(chan error, 1)

	client.Mux.RegisterHandler("pong", func(dto *Message) error {
		waitPong <- client.pongHandler(dto)
		return nil
	})

	return Artifex.SendPingWaitPong(sendPing, waitPong, client.IsStop, pongWaitSecond)
}
