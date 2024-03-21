package rabbit

import (
	"context"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"talent.com/naming/common/adapter"
)

type Hub = adapter.Hub[RoutingKey, *AmqpMessage, *NullMsg]

func NewHub() *Hub {
	// TODO
	return adapter.NewHub[RoutingKey, *AmqpMessage, *NullMsg]()
}

type Session = adapter.Session[RoutingKey, *AmqpMessage, *NullMsg]

type Case1SessionFactory struct {
	Conn            *amqp.Connection
	ConnectionMutex sync.Mutex

	Mux *Mux

	AmqpUri string
	Ctx     context.Context

	QosPrefetchCount int

	ExchangeName string
	ExchangeType string

	BindingKey string
	RoutingKey RoutingKey

	QueueName string
	QueueTTL  time.Duration

	ConsumerName   string
	ConsumeAutoAck bool
}

func (f *Case1SessionFactory) CreateSession() (*Session, error) {
	f.ConnectionMutex.Lock()
	channel, err := f.Conn.Channel()
	if err != nil {
		f.ConnectionMutex.Unlock()
		return nil, err
	}
	f.ConnectionMutex.Unlock()

	consumer, err := f.createConsumer(channel)
	if err != nil {
		return nil, err
	}

	life := adapter.Lifecycle[RoutingKey, *AmqpMessage, *NullMsg]{
		SpawnHandlers: []func(sess *Session) error{
			f.SetupAdapterWithFixup(channel, consumer, adapter.DefaultLogger()),
		},
	}
	sess := &Session{
		Mux:        f.Mux,
		Identifier: f.ConsumerName,
		Lifecycle:  life,
	}
	return sess, nil
}

func (f *Case1SessionFactory) createConsumer(channel *amqp.Channel) (<-chan amqp.Delivery, error) {
	err := channel.Qos(f.QosPrefetchCount, 0, false)
	if err != nil {
		return nil, err
	}

	err = channel.ExchangeDeclare(
		f.ExchangeName,
		f.ExchangeType,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	var queueArgs amqp.Table
	if f.QueueTTL >= 0 {
		queueArgs = make(amqp.Table)
		queueArgs[amqp.QueueTTLArg] = f.QueueTTL.Milliseconds()
	}
	queue, err := channel.QueueDeclare(
		f.QueueName,
		true,
		false,
		false,
		false,
		queueArgs,
	)
	if err != nil {
		return nil, err
	}

	var bindQueueArgs amqp.Table
	if f.QueueTTL >= 0 {
		queueArgs = make(amqp.Table)
		queueArgs[amqp.QueueTTLArg] = f.QueueTTL.Milliseconds()
	}
	err = channel.QueueBind(
		queue.Name,
		f.BindingKey,
		f.ExchangeName,
		false,
		bindQueueArgs,
	)
	if err != nil {
		return nil, err
	}

	consumer, err := channel.Consume(
		queue.Name,
		f.ConsumerName,
		f.ConsumeAutoAck,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return consumer, nil
}

func (f *Case1SessionFactory) SetupAdapterWithFixup(channel *amqp.Channel, consumer <-chan amqp.Delivery, logger adapter.Logger) func(sess *Session) error {

	return func(sess *Session) error {
		channelNotify := channel.NotifyClose(make(chan *amqp.Error, 1))
		connectionNotify := f.Conn.NotifyClose(make(chan *amqp.Error, 1))

		sess.AdapterRecv = func() (*AmqpMessage, error) {
			select {
			case msg := <-consumer:
				return NewAmqpMessage(f.Ctx, msg), nil

			case Err := <-channelNotify:
				logger.Error("channel error: %v", Err)
				return nil, Err

			case Err := <-connectionNotify:
				logger.Error("connection error: %v", Err)
				return nil, Err
			}
		}

		sess.AdapterSend = func(_ *NullMsg) error { return nil }

		sess.AdapterStop = func(_ *NullMsg) {
			f.ConnectionMutex.Lock()
			defer f.ConnectionMutex.Unlock()
			f.Conn.Close()
			channel.Close()
			channel.Cancel(f.ConsumerName, true)
			return
		}

		cnt := 0
		sess.Fixup = func() error {
			f.ConnectionMutex.Lock()
			cnt++
			logger.Info("retry amqp %v times", cnt)

			freshChannel, chErr := f.Conn.Channel()
			if chErr != nil {
				freshConnection, connErr := amqp.Dial(f.AmqpUri)
				if connErr != nil {
					f.ConnectionMutex.Unlock()
					return connErr
				}

				f.Conn = freshConnection
				connectionNotify = f.Conn.NotifyClose(make(chan *amqp.Error, 1))
				f.ConnectionMutex.Unlock()
				return chErr
			}

			channel = freshChannel
			channelNotify = channel.NotifyClose(make(chan *amqp.Error, 1))
			freshConsumer, err := f.createConsumer(channel)
			if err != nil {
				f.ConnectionMutex.Unlock()
				return err
			}

			cnt = 0
			f.ConnectionMutex.Unlock()

			consumer = freshConsumer
			return nil
		}
		return nil
	}
}
