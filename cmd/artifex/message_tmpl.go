package main

const MessageTmpl = `
package {{.Package}}

import (
	"github.com/KScaesar/Artifex"
)

func New{{.FileName}}Ingress(bBody []byte, metadata any, pingpong Artifex.WaitPingPong) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Bytes = bBody
	{{.FileName}}Metadata.SetPingPong(message.Metadata, pingpong)
	message.RawInfra = nil
	return message
}

func New{{.FileName}}Egress(body any) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Body = body
	return message
}

func New{{.FileName}}EgressWithSubject(subject string, body any) *Artifex.Message {
	message := Artifex.GetMessage()

	message.Subject = subject
	message.Body = body
	return message
}

`
