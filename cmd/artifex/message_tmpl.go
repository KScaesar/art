package main

const MessageTmpl = `
package {{.Package}}

import (
	"context"

	"github.com/KScaesar/Artifex"
)

func New{{.FileName}}Ingress(bBody []byte, metadata any, pingpong Artifex.WaitPingPong) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Bytes = bBody
	{{.FileName}}Metadata.SetCorrelationId(message.Metadata, metadata)
	message.RawInfra = nil
	message.UpdateContext(func(ctx context.Context) context.Context {
		return Artifex.CtxWithPingPong(ctx, pingpong)
	})
	return message
}

func NewBytes{{.FileName}}Egress(bMessage []byte) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Bytes = bMessage
	return message
}

func NewBody{{.FileName}}Egress(body any) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Body = body
	return message
}

func NewBody{{.FileName}}EgressWithSubject(subject string, body any) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Subject = subject
	message.Body = body
	return message
}

`
