package websocket

import (
	"github.com/gorilla/websocket"
)

type Message struct {
	MessageType int
	Payload     []byte

	Parent *websocket.Conn
}
