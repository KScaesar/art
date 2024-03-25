package main

import (
	"github.com/gookit/goutil/maputil"

	"github.com/KScaesar/Artifex"
)

type Channel = string

//

func NewRedisIngress() *RedisIngress {
	return &RedisIngress{}
}

type RedisIngress struct {
	Channel        Channel
	IngressByteMsg []byte

	ParentInfra any
}

type RedisIngressHandleFunc = Artifex.HandleFunc[RedisIngress]
type RedisIngressMiddleware = Artifex.Middleware[RedisIngress]
type RedisIngressMux = Artifex.Mux[Channel, RedisIngress]

func NewRedisIngressMux() *RedisIngressMux {
	getChannel := func(message *RedisIngress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[Channel](getChannel)
	mux.Handler("ingress", RedisIngressHandler())
	return mux
}

func RedisIngressHandler() RedisIngressHandleFunc {
	return func(message *RedisIngress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func RedisIngressHandleError(message *RedisIngress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}

//

func NewRedisEgress() *RedisEgress {
	return &RedisEgress{}
}

type RedisEgress struct {
	Channel       Channel
	EgressByteMsg []byte

	Metadata maputil.Data
	AppMsg   any
}

type RedisEgressHandleFunc = Artifex.HandleFunc[RedisEgress]
type RedisEgressMiddleware = Artifex.Middleware[RedisEgress]
type RedisEgressMux = Artifex.Mux[Channel, RedisEgress]

func NewRedisEgressMux() *RedisEgressMux {
	getChannel := func(message *RedisEgress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[Channel](getChannel)
	mux.Handler("egress", RedisEgressHandler())
	return mux
}

func RedisEgressHandler() RedisEgressHandleFunc {
	return func(message *RedisEgress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func RedisEgressHandleError(message *RedisIngress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}
