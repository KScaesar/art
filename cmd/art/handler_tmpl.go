package main

const HandlerTmpl = `
package {{.Package}}

import (
	"github.com/KScaesar/art"
)

type {{.FileName}}IngressMux = art.Mux
type {{.FileName}}EgressMux = art.Mux

func New{{.FileName}}IngressMux(pingpong bool) *{{.FileName}}IngressMux {
	in := art.NewMux("/").
		Transform(func(message *art.Message, dep any) error {
			return nil
		}).
		Handler("pong", func(message *art.Message, dep any) error {
			art.CtxGetPingPong(message.Ctx).Ack()
			if pingpong {
				dep.(art.IAdapter).Log().Debug("ack pong")
			}
			return nil
		})
	return in
}

func New{{.FileName}}EgressMux(pingpong bool) *{{.FileName}}EgressMux {
	out := art.NewMux("/").
		Transform(func(message *art.Message, dep any) error {
			return nil
		}).
		Handler("ping", func(message *art.Message, dep any) error {
			if pingpong {
				dep.(art.IAdapter).Log().Debug("send ping")
			}
			return nil
		})
	return out
}

`
