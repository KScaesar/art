package xWebsocket

import (
	"github.com/KScaesar/Artifex"
)

func NewMessage(bMessage []byte, kind int) *Message {
	return &Message{
		ByteMessage: bMessage,
		MessageKind: kind,
	}
}

type Message struct {
	// input
	ByteMessage []byte
	MessageKind int

	Subject string

	// output
	// ErrorResponse 與正常訊息的語意是反向的
	// 如果 ErrorResponse handler 處理過程沒有問題, 則 handler 應該 return nil
	// 然後再取得處理後的結果
	ErrResponseResult error
}

func NewWebsocketMux(getSubject func(dto *Message) (string, error)) *WebsocketMux {
	return Artifex.NewMessageMux(getSubject)
}

type WebsocketMux = Artifex.MessageMux[string, *Message]

type WebsocketHandler = Artifex.MessageHandler[*Message]

type WebsocketDecorator = Artifex.MessageDecorator[*Message]
