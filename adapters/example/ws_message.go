package main

import (
	"github.com/gookit/goutil/maputil"

	"github.com/KScaesar/Artifex"
)

type LeafId = string

//

func NewWsIngress() *WsIngress {
	return &WsIngress{}
}

type WsIngress struct {
	LeafId     LeafId
	IngressMsg []byte

	ParentInfra any
}

type WsIngressHandleFunc = Artifex.HandleFunc[WsIngress]
type WsIngressMiddleware = Artifex.Middleware[WsIngress]
type WsIngressMux = Artifex.Mux[LeafId, WsIngress]

func NewWsIngressMux() *WsIngressMux {
	getLeafId := func(message *WsIngress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[LeafId](getLeafId)
	mux.Handler("ingress", WsIngressHandler())
	return mux
}

func WsIngressHandler() WsIngressHandleFunc {
	return func(message *WsIngress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func WsIngressHandleError(message *WsIngress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}

//

func NewWsEgress() *WsEgress {
	return &WsEgress{}
}

type WsEgress struct {
	LeafId    LeafId
	EgressMsg []byte

	Metadata maputil.Data
	AppMsg   any
}

type WsEgressHandleFunc = Artifex.HandleFunc[WsEgress]
type WsEgressMiddleware = Artifex.Middleware[WsEgress]
type WsEgressMux = Artifex.Mux[LeafId, WsEgress]

func NewWsEgressMux() *WsEgressMux {
	getLeafId := func(message *WsEgress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[LeafId](getLeafId)
	mux.Handler("egress", WsEgressHandler())
	return mux
}

func WsEgressHandler() WsEgressHandleFunc {
	return func(message *WsEgress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func WsEgressHandleError(message *WsIngress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}
