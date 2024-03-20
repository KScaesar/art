package websocket

import (
	"context"
	"sync"

	"github.com/KScaesar/Artifex"
	"github.com/gorilla/websocket"
)

type Hub = Artifex.Artist[Title, *WsInMsg, *WsOutMsg]

func NewHub() *Hub {
	// TODO: option
	return Artifex.NewArtist[Title, *WsInMsg, *WsOutMsg]()
}

type Session = Artifex.Session[Title, *WsInMsg, *WsOutMsg]

type SessionFactory struct {
	Dialer *websocket.Dialer
	Url    string
}

func (f *SessionFactory) CreateSession() (*Session, error) {
	conn, _, Err := f.Dialer.Dial(f.Url, nil)
	if Err != nil {
		return nil, Err
	}
	var mu sync.Mutex

	sess := &Session{
		Identifier: "",
		Context:    context.Background(),
		Pingpong:   NewPigPong(),
	}
	fixup := func() (err error) {
		mu.Lock()
		defer mu.Unlock()
		conn, _, err = f.Dialer.Dial(f.Url, nil)
		return err
	}

	sess.Mux = nil
	recv := func() (*WsInMsg, error) {
		messageType, bBody, err := conn.ReadMessage()
		if err != nil {
			return nil, err
		}

		title := ""
		switch messageType {
		case websocket.PingMessage:
			title = "ping"
		case websocket.TextMessage:
			title = string(bBody)
		}

		return &WsInMsg{
			Title: title,
			Body:  bBody,
			Conn:  conn,
			Sess:  sess,
		}, nil
	}
	send := func(message *WsOutMsg) error {
		mu.Lock()
		defer mu.Unlock()

		switch message.Title {
		case TitlePong:
			return conn.WriteMessage(websocket.PongMessage, nil)
		}
		return conn.WriteJSON(message.Obj)
	}
	stop := func(message *WsOutMsg) {
		conn.Close()
	}

	sess.AdapterRecv = recv
	sess.AdapterSend = send
	sess.AdapterStop = stop
	return sess, nil
}

func NewPigPong() Artifex.PingPong[Title, *WsInMsg, *WsOutMsg] {
	return Artifex.PingPong[Title, *WsInMsg, *WsOutMsg]{
		Enable:     true,
		WaitSecond: 0,
		// WaitSubject: "PingPongTitle",
		SendFunc: func(session *Session) error {
			pingpong := &WsOutMsg{
				Title: TitlePong,
			}
			return session.Send(pingpong)
		},
		IsSendPingWaitPong: false,
		WaitFunc: func(message *WsInMsg, route *Artifex.RouteParam) error {
			return nil
		},
	}
}
