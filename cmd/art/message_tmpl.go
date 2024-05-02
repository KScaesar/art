package main

const MessageTmpl = `
package {{.Package}}

import (
	"context"

	"github.com/KScaesar/art"
)

func New{{.FileName}}Ingress(bBody []byte, rawInfra any, pingpong art.WaitPingPong) *art.Message {
	message := art.GetMessage()

	message.Bytes = bBody
	{{.FileName}}Metadata.SetCorrelationId(message.Metadata, "")
	message.SetMsgId("")
	message.RawInfra = rawInfra
	message.UpdateContext(func(ctx context.Context) context.Context {
		return art.CtxWithPingPong(ctx, pingpong)
	})
	return message
}

func NewBytes{{.FileName}}Egress(bMessage []byte) *art.Message {
	message := art.GetMessage()

	message.Bytes = bMessage
	return message
}

func NewBody{{.FileName}}Egress(body any) *art.Message {
	message := art.GetMessage()

	message.Body = body
	return message
}

func NewBody{{.FileName}}EgressWithSubject(subject string, body any) *art.Message {
	message := art.GetMessage()

	message.Subject = subject
	message.Body = body
	return message
}

`
