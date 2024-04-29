package main

const HandlerTmpl = `
package {{.Package}}

import (
	"github.com/KScaesar/Artifex"
)

type {{.FileName}}IngressMux = Artifex.Mux
type {{.FileName}}EgressMux = Artifex.Mux

func New{{.FileName}}IngressMux(pingpong bool) *{{.FileName}}IngressMux {
	in := Artifex.NewMux("/").
		Transform(func(message *Artifex.Message, dep any) error {
			return nil
		}).
		Handler("ping", func(message *Artifex.Message, dep any) error {
			{{.FileName}}Metadata.GetPingPong(message.Metadata).Ack()
			if pingpong {
				dep.(Artifex.IAdapter).Log().Debug("ack ping")
			}
			return nil
		})
	return in
}

func New{{.FileName}}EgressMux(pingpong bool) *{{.FileName}}EgressMux {
	out := Artifex.NewMux("/").
		Transform(func(message *Artifex.Message, dep any) error {
			return nil
		}).
		Handler("pong", func(message *Artifex.Message, dep any) error {
			if pingpong {
				dep.(Artifex.IAdapter).Log().Debug("send pong")
			}
			return nil
		})
	return out
}

`
