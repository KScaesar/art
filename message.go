package Artifex

import (
	"context"

	"github.com/gookit/goutil/maputil"
)

func NewMessageWithBytes(subject string, bBody []byte) *Message {
	return &Message{
		Subject:    subject,
		Bytes:      bBody,
		Metadata:   map[string]any{},
		RouteParam: map[string]any{},
	}
}

func NewMessageWithBody(subject string, body any) *Message {
	return &Message{
		Subject:    subject,
		Body:       body,
		Metadata:   map[string]any{},
		RouteParam: map[string]any{},
	}
}

type Message struct {
	Subject string

	// RouteParam are used to capture values from subject.
	// These parameters represent resources or identifiers.
	//
	// Example:
	//
	//	define mux subject = "/users/{id}"
	//	send or recv subject = "/users/1017"
	//
	//	route:
	//		key : value => id : 1017
	RouteParam maputil.Data

	identifier string
	Metadata   maputil.Data

	Bytes []byte
	Body  any

	Infra any

	ctx    context.Context
	Logger Logger
}

func (msg *Message) reset() {
	if msg.Bytes != nil {
		msg.Bytes = msg.Bytes[:0]
	}
	msg.Body = nil
	msg.Subject = ""
	msg.identifier = ""
	for key, _ := range msg.Metadata {
		delete(msg.Metadata, key)
	}
	for key, _ := range msg.RouteParam {
		delete(msg.RouteParam, key)
	}
	msg.Infra = nil
	msg.ctx = nil
	msg.Logger = nil
}

func (msg *Message) Log() Logger {
	return msg.Logger.WithKeyValue("msg_id", msg.identifier)
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

//

type HandleFunc func(message *Message, dep any) error

func (h HandleFunc) PreMiddleware() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			err := h(message, dep)
			if err != nil {
				return err
			}
			return next(message, dep)
		}
	}
}

func (h HandleFunc) PostMiddleware() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			err := next(message, dep)
			if err != nil {
				return err
			}
			return h(message, dep)
		}
	}
}

func (h HandleFunc) LinkMiddlewares(middlewares ...Middleware) HandleFunc {
	return LinkMiddlewares(h, middlewares...)
}

type Middleware func(next HandleFunc) HandleFunc

func (mw Middleware) HandleFunc() HandleFunc {
	return LinkMiddlewares(func(message *Message, dep any) error {
		return nil
	}, mw)
}

func LinkMiddlewares(handler HandleFunc, middlewares ...Middleware) func(*Message, any) error {
	n := len(middlewares)
	for i := n - 1; 0 <= i; i-- {
		decorator := middlewares[i]
		handler = decorator(handler)
	}
	return handler
}
