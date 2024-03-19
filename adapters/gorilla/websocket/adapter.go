package websocket

import (
	"log"
	"sync"

	"github.com/KScaesar/Artifex"
	"github.com/gookit/goutil/maputil"
	"github.com/gorilla/websocket"
)

type Session = Artifex.Session[string, *Message, *Message]

type Message struct {
	// obj from gorilla/websocket.Conn.ReadMessage
	MessageType int
	Payload     []byte
	Conn        *websocket.Conn

	// obj from Artifex.Session
	Sess *Session

	// obj from your application
	AppData maputil.Data
}

func NewMux() *Artifex.Mux[string, *Message] {
	getSubject := func(message *Message) (string, error) {
		return string(message.Payload), nil
	}
	return Artifex.NewMux[string](getSubject)
}

type HandeFunc = Artifex.HandleFunc[*Message]

type NewAdapter = Artifex.NewAdapterFunc[string, *Message, *Message]

func SimpleClient(url, clientName string) NewAdapter {
	return func() (adapter Artifex.Adapter[string, *Message, *Message], err error) {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			return
		}

		var mutex sync.Mutex
		logger := log.Default()

		return Artifex.Adapter[string, *Message, *Message]{
			Recv: func(parent *Session) (*Message, error) {
				messageType, p, err := conn.ReadMessage()
				if err != nil {
					return nil, err
				}
				return &Message{
					MessageType: messageType,
					Payload:     p,
					Conn:        conn,
					Sess:        parent,
					AppData:     nil,
				}, nil
			},
			Send: func(message *Message) error {
				const (
					ping = iota + 1
					notify
				)

				cmd := message.AppData.Int("command")

				mutex.Lock()
				defer mutex.Unlock()
				logger.Println("command=%v", cmd)

				switch cmd {
				case ping:
					return conn.WriteMessage(websocket.PingMessage, nil)

				case notify:
					return conn.WriteMessage(websocket.BinaryMessage, message.Payload)

				default:
					return conn.WriteMessage(websocket.TextMessage, message.Payload)

				}
			},
			Stop: func(message *Message) {
				mutex.Lock()
				defer mutex.Unlock()
				conn.Close()
			},

			Identifier: clientName,
			Context:    nil,
			Logger:     nil,
		}, nil
	}
}
