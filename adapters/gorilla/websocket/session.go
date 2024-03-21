package leaf

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub = util.Hub[LeafId, *WsMessage, *LeafMessage]

func NewHub() *Hub {
	return util.NewHub[LeafId, *WsMessage, *LeafMessage]()
}

type Session = util.Session[LeafId, *WsMessage, *LeafMessage]

type SimpleClientFactory struct {
	Dialer        *websocket.Dialer
	Url           string
	RequestHeader http.Header

	Name          string
	PrintPingPong bool

	RecvMux *WsMux
	Logger  util.Logger
}

func (f *SimpleClientFactory) CreateSession() (*Session, error) {
	conn, _, Err := f.Dialer.Dial(f.Url, f.RequestHeader)
	if Err != nil {
		return nil, Err
	}

	var mu sync.Mutex
	life := util.Lifecycle[LeafId, *WsMessage, *LeafMessage]{
		SpawnHandlers: []func(sess *Session) error{
			WsClientWithPingPong(f, conn, &mu),
		},
		ExitHandlers: []func(sess *Session){},
	}
	session := &Session{
		Mux:        f.RecvMux,
		Identifier: f.Name,
		Lifecycle:  life,
	}
	return session, nil
}

func WsClientWithPingPong(f *SimpleClientFactory, conn *websocket.Conn, mu *sync.Mutex) func(sess *Session) error {
	logger := f.Logger
	if logger == nil {
		logger = util.DefaultLogger()
	}
	sessId := util.GenerateRandomCode(6)
	logger = logger.WithSessionId(sessId)

	return func(sess *Session) error {
		sess.AdapterRecv = func() (*WsMessage, error) {
			_, rawMessage, err := conn.ReadMessage()
			if err != nil {
				return nil, err
			}
			return NewWsMessage(rawMessage, sess, logger), nil
		}

		sess.AdapterSend = func(message *LeafMessage) error {
			bMessage, err := leaf.EncodeLeafMessage(message)
			if err != nil {
				return err
			}
			if message.LeafId != LeafId_Pong && message.LeafId != LeafId_Ping {
				logger.WithCallDepth(2).Debug("send leafId=%v to '%v'", message.LeafId, sess.Identifier)
			}
			mu.Lock()
			defer mu.Unlock()
			return conn.WriteMessage(websocket.BinaryMessage, bMessage)
		}

		sess.AdapterStop = func(message *LeafMessage) {
			var bMessage []byte
			if message != nil {
				bMessage, _ = leaf.EncodeLeafMessage(message)
			}
			logger.WithCallDepth(2).Debug("'%v' stop", sess.Identifier)
			mu.Lock()
			defer mu.Unlock()
			conn.WriteMessage(websocket.BinaryMessage, bMessage)
			conn.WriteMessage(websocket.CloseMessage, bMessage)
			conn.Close()
			return
		}

		pp := util.PingPong{
			Enable:             true,
			IsWaitPingSendPong: false,
			SendFunc: func() error {
				bMessage, err := leaf.EncodeLeafMessage(NewLeafMessage(LeafId_Ping, &leaf.Ping{}))
				if err != nil {
					return err
				}
				if f.PrintPingPong {
					logger.Debug("send ping")
				}
				mu.Lock()
				defer mu.Unlock()
				return conn.WriteMessage(websocket.BinaryMessage, bMessage)
			},
			WaitNotify: make(chan error, 1),
			WaitSecond: 15,
		}
		sess.PingPong = pp
		sess.Mux.Handler(LeafId_Pong, func(_ *WsMessage, _ *util.RouteParam) error {
			if f.PrintPingPong {
				logger.Debug("handle pong")
			}
			pp.WaitNotify <- nil
			return nil
		})

		return nil
	}
}
