package Artifex

import (
	"context"
	"strconv"

	"github.com/gookit/goutil/maputil"
)

func NewNumberSubjectMessageWithBytes(number int, delimiter string, bBody []byte) *Message {
	return &Message{
		Subject:    strconv.Itoa(number) + delimiter,
		Bytes:      bBody,
		Metadata:   map[string]any{},
		RouteParam: map[string]any{},
	}
}

func NewNumberSubjectMessageWithBody(number int, delimiter string, body any) *Message {
	return &Message{
		Subject:    strconv.Itoa(number) + delimiter,
		Body:       body,
		Metadata:   map[string]any{},
		RouteParam: map[string]any{},
	}
}

func NewMessageWithBytes(subject string, bBody []byte) *Message {
	return &Message{
		Subject:    subject,
		Bytes:      bBody,
		RouteParam: map[string]any{},
		Metadata:   map[string]any{},
	}
}

func NewMessageWithBody(subject string, body any) *Message {
	return &Message{
		Subject:    subject,
		Body:       body,
		RouteParam: map[string]any{},
		Metadata:   map[string]any{},
	}
}

func NewMessage() *Message {
	return &Message{
		RouteParam: map[string]any{},
		Metadata:   map[string]any{},
	}
}

type Message struct {
	Subject    string
	Bytes      []byte
	Body       any
	identifier string

	// RFC "OPTIONAL" below

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

	RawInfra any // for ingress side

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
	msg.RawInfra = nil
	msg.ctx = nil
	msg.Logger = nil
}
