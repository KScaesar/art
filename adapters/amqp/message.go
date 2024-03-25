package rabbit

import (
	"github.com/KScaesar/Artifex"
	"github.com/gookit/goutil/maputil"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RoutingKey = string

//

func NewIngress(amqpMsg amqp.Delivery, logger Artifex.Logger) *Ingress {
	return &Ingress{
		RoutingKey: amqpMsg.RoutingKey,
		IngressMsg: amqpMsg.Body,
		AmqpMsg:    amqpMsg,
		Logger:     logger,
	}
}

type Ingress struct {
	RoutingKey RoutingKey
	IngressMsg []byte

	AmqpMsg amqp.Delivery
	Logger  Artifex.Logger
}

type IngressHandleFunc = Artifex.HandleFunc[Ingress]
type IngressMiddleware = Artifex.Middleware[Ingress]
type IngressMux = Artifex.Mux[RoutingKey, Ingress]

func NewIngressMux() *IngressMux {
	getRoutingKey := func(message *Ingress) (string, error) {
		return message.AmqpMsg.RoutingKey, nil
	}

	mux := Artifex.NewMux[RoutingKey](getRoutingKey)
	return mux
}

//

func NewEgress() *Egress {
	return &Egress{}
}

type Egress struct {
	RoutingKey RoutingKey
	EgressMsg  []byte

	Metadata maputil.Data
	AppMsg   any
}

type EgressHandleFunc = Artifex.HandleFunc[Egress]
type EgressMiddleware = Artifex.Middleware[Egress]
type EgressMux = Artifex.Mux[RoutingKey, Egress]

func NewEgressMux() *EgressMux {
	getRoutingKey := func(message *Egress) (string, error) {
		// TODO
		return "", nil
	}

	mux := Artifex.NewMux[RoutingKey](getRoutingKey)
	mux.Handler("egress", EgressHandler())
	return mux
}

func EgressHandler() EgressHandleFunc {
	return func(message *Egress, _ *Artifex.RouteParam) error {
		return nil
	}
}

func EgressHandleError(message *Ingress, _ *Artifex.RouteParam, err error) error {
	if err != nil {
		return err
	}
	return nil
}
