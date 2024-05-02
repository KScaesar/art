package art

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
		Ctx:        context.Background(),
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

	Ctx context.Context
}

func (msg *Message) UpdateContext(updates ...func(ctx context.Context) context.Context) context.Context {
	for _, update := range updates {
		msg.Ctx = update(msg.Ctx)
	}
	return msg.Ctx
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

func (msg *Message) reset() {
	msg.Subject = ""
	msg.Bytes = nil
	msg.Body = nil
	msg.identifier = ""

	for key := range msg.RouteParam {
		delete(msg.RouteParam, key)
	}
	for key := range msg.Metadata {
		delete(msg.Metadata, key)
	}

	msg.RawInfra = nil
	msg.Ctx = context.Background()
}

func (msg *Message) Copy() *Message {
	message := GetMessage()

	message.Subject = msg.Subject
	message.Bytes = msg.Bytes
	message.Body = msg.Body
	message.identifier = msg.identifier

	for key, v := range msg.RouteParam {
		message.RouteParam.Set(key, v)
	}
	for key, v := range msg.Metadata {
		message.Metadata.Set(key, v)
	}

	message.RawInfra = message.RawInfra
	message.Ctx = message.Ctx
	return message
}
