package websocket

import (
	"github.com/KScaesar/Artifex"
	"github.com/gookit/goutil/maputil"
	"github.com/gorilla/websocket"
)

type Message struct {
	// obj from gorilla/websocket.Conn.ReadMessage
	MessageType int
	Payload     []byte
	Conn        *websocket.Conn

	// obj from Artifex.Session
	sess *Session

	// obj from your application
	Keys maputil.Data
}

type Session = Artifex.Session[string, Message, Message]

type NewAdapter = Artifex.NewAdapterFunc[string, Message, Message]

type GetSubject = Artifex.NewSubjectFunc[Message]

type Mux = Artifex.Mux[string, Message]
