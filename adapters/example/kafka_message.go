package infra

import (
	"github.com/gookit/goutil/maputil"

	"github.com/KScaesar/Artifex"
)

type Topic = string

//

func NewKafkaIngress() *KafkaIngress {
	return &KafkaIngress{}
}

type KafkaIngress struct {
	MsgId      string
	IngressMsg []byte

	Topic       Topic
	ParentInfra any
}

type KafkaIngressHandleFunc = Artifex.HandleFunc[KafkaIngress]
type KafkaIngressMiddleware = Artifex.Middleware[KafkaIngress]
type KafkaIngressMux = Artifex.Mux[Topic, KafkaIngress]

func NewKafkaIngressMux() *KafkaIngressMux {
	getTopic := func(message *KafkaIngress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[Topic](getTopic)
	mux.Handler("ingress", KafkaIngressHandler())
	return mux
}

func KafkaIngressHandler() KafkaIngressHandleFunc {
	return func(message *KafkaIngress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func KafkaIngressHandleError(message *KafkaIngress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}

//

func NewKafkaEgress() *KafkaEgress {
	return &KafkaEgress{}
}

type KafkaEgress struct {
	msgId     string
	EgressMsg []byte

	Topic    Topic
	Metadata maputil.Data
	AppMsg   any
}

func (e *KafkaEgress) MsgId() string {
	if e.msgId == "" {
		return ""
	}
	return e.msgId
}

func (e *KafkaEgress) SetMsgId(msgId string) {
	e.msgId = msgId
}

type KafkaEgressHandleFunc = Artifex.HandleFunc[KafkaEgress]
type KafkaEgressMiddleware = Artifex.Middleware[KafkaEgress]
type KafkaEgressMux = Artifex.Mux[Topic, KafkaEgress]

func NewKafkaEgressMux() *KafkaEgressMux {
	getTopic := func(message *KafkaEgress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[Topic](getTopic)
	mux.Handler("egress", KafkaEgressHandler())
	return mux
}

func KafkaEgressHandler() KafkaEgressHandleFunc {
	return func(message *KafkaEgress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func KafkaEgressHandleError(message *KafkaIngress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}
