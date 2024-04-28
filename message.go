package Artifex

import (
	"context"
	"sync"

	"github.com/gookit/goutil/maputil"
)

func GetMessage() *Message {
	return messagePool.Get()
}

func PutMessage(message *Message) {
	message.reset()
	messagePool.Put(message)
}

var messagePool = newPool(newMessage)

func newMessage() *Message {
	return &Message{
		RouteParam: map[string]any{},
		Metadata:   map[string]any{},
	}
}

type Message struct {
	Subject string

	Bytes []byte
	Body  any

	identifier string

	Mutex sync.Mutex

	// RouteParam are used to capture values from subject.
	// These parameters represent resources or identifiers.
	//
	// Example:
	//
	//	define mux subject = "/users/{id}"
	//	send or recv subject = "/users/1017"
	//
	//	get route param:
	//		key : value => id : 1017
	RouteParam maputil.Data

	Metadata maputil.Data

	RawInfra any

	ctx    context.Context
	Logger Logger
}

func (msg *Message) MsgId() string {
	if msg.identifier == "" {
		msg.identifier = GenerateUlid()
	}
	return msg.identifier
}

func (msg *Message) SetMsgId(msgId string) {
	msg.identifier = msgId
}

func (msg *Message) Context() context.Context {
	if msg.ctx == nil {
		msg.ctx = context.Background()
	}
	return msg.ctx
}

func (msg *Message) SetContext(ctx context.Context) {
	msg.ctx = ctx
}

func (msg *Message) SwapContext(swaps ...func(ctx1 context.Context) (ctx2 context.Context)) context.Context {
	for _, swap := range swaps {
		msg.ctx = swap(msg.Context())
	}
	return msg.ctx
}

func (msg *Message) reset() {
	msg.Subject = ""

	if msg.Bytes != nil {
		msg.Bytes = msg.Bytes[:0]
	}
	msg.Body = nil

	msg.identifier = ""

	for key := range msg.RouteParam {
		delete(msg.RouteParam, key)
	}
	for key := range msg.Metadata {
		delete(msg.Metadata, key)
	}

	msg.RawInfra = nil
	msg.ctx = nil
	msg.Logger = nil
}
