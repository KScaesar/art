package rabbit

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"

	"talent.com/naming/common/adapter"
)

// TODO
type RoutingKey = string

func NewAmqpMessage(ctx context.Context, rawMsg amqp.Delivery) *AmqpMessage {
	return &AmqpMessage{
		Ctx:        nil,
		RawMessage: rawMsg,
	}
}

type AmqpMessage struct {
	Ctx        context.Context
	RawMessage amqp.Delivery
}

type NullMsg struct{}

//

type HandleFunc = adapter.HandleFunc[*AmqpMessage]
type Middleware = adapter.Middleware[*AmqpMessage]
type Mux = adapter.Mux[RoutingKey, *AmqpMessage]

func NewMux() *Mux {
	getRoutingKey := func(message *AmqpMessage) (string, error) {
		// TODO
		return "", nil
	}

	mux := adapter.NewMux[RoutingKey](getRoutingKey)
	mux.Handler("hello", HelloHandler())
	return mux
}

// Example
func HelloHandler() HandleFunc {
	return func(message *AmqpMessage, _ *adapter.RouteParam) error {
		return nil
	}
}
