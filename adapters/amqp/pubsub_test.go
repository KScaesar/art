package rabbit

import (
	"sync"
	"testing"
	"time"

	"github.com/KScaesar/Artifex"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestFactory_CreateSubscriber(t *testing.T) {
	record := func(caseName string) func(message *Ingress, _ *Artifex.RouteParam) error {
		return func(message *Ingress, _ *Artifex.RouteParam) error {
			message.Logger.Info("record %v", caseName)
			return nil
		}
	}
	mux := NewIngressMux()
	mux.Handler("test1", record("test1"))

	url := "amqp://guest:guest@127.0.0.1:5672"
	connection, err := NewConnection(url)
	require.NoError(t, err)

	mu := &sync.Mutex{}
	var factoryAll = []*Factory{
		{
			AmqpUri:         url,
			ConnectionMutex: mu,
			Connection:      &connection,
			SetupQos:        testSetupQos(1),
			SetupEx:         testSetupEx("test-ex", "topic"),
			SetupQueue:      testSetupTemporaryQueue("test-q", 10*time.Second),
			SetupBind:       testSetupTemporaryBind("test-q", "test-ex", []string{"test1"}, 10*time.Second),
			NewConsumer:     testNewConsumer("test-q", "test-consumer1", true),
			RecvMux:         mux,
			SendMux:         nil,
			Logger:          Artifex.DefaultLogger(),
		},
		{
			AmqpUri:         url,
			ConnectionMutex: mu,
			Connection:      &connection,
			SetupQos:        testSetupQos(1),
			SetupEx:         testSetupEx("test-ex", "topic"),
			SetupQueue:      testSetupTemporaryQueue("test-q", 10*time.Second),
			SetupBind:       testSetupTemporaryBind("test-q", "test-ex", []string{"test2"}, 10*time.Second),
			NewConsumer:     testNewConsumer("test-q", "test-consumer2", true),
			RecvMux:         mux,
			SendMux:         nil,
			Logger:          Artifex.DefaultLogger(),
		},
	}
	hub := NewSubscriberHub()
	for _, factory := range factoryAll {
		subscriber, err := factory.CreateSubscriber()
		if err != nil {
			hub.Stop()
			return
		}
		require.NoError(t, err)

		go subscriber.Listen()

		err = hub.JoinAdapter(subscriber.Identifier, subscriber)
		require.NoError(t, err)
	}

	shutdown := Artifex.SetupShutdown(nil, func() (serviceName string) {
		hub.Stop()
		return "amqp"
	})
	shutdown.Wait()
	// shutdown.WaitAfter(5)
}

func testSetupQos(prefetchCount int) func(channel *amqp.Channel) error {
	return func(channel *amqp.Channel) error {
		return channel.Qos(
			prefetchCount,
			0,
			false)
	}
}

func testSetupEx(name, kind string) func(channel *amqp.Channel) error {
	return func(channel *amqp.Channel) error {
		return channel.ExchangeDeclare(
			name,
			kind,
			true,
			false,
			false,
			false,
			nil,
		)
	}
}

func testSetupQueue(name string) func(channel *amqp.Channel) error {
	return func(channel *amqp.Channel) error {
		_, err := channel.QueueDeclare(
			name,
			true,
			false,
			false,
			false,
			nil,
		)
		return err
	}
}

func testSetupTemporaryQueue(name string, ttl time.Duration) func(channel *amqp.Channel) error {
	queueArgs := make(amqp.Table)
	queueArgs[amqp.QueueTTLArg] = ttl.Milliseconds()
	return func(channel *amqp.Channel) error {
		_, err := channel.QueueDeclare(
			name,
			false,
			false,
			false,
			false,
			queueArgs,
		)
		return err
	}
}

func testSetupBind(queueName, exName string, bindingKeys []string) func(channel *amqp.Channel) error {
	return func(channel *amqp.Channel) error {
		for _, key := range bindingKeys {
			err := channel.QueueBind(
				queueName,
				key,
				exName,
				false,
				nil,
			)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func testSetupTemporaryBind(queueName, exName string, bindingKeys []string, ttl time.Duration) func(channel *amqp.Channel) error {
	bindArgs := make(amqp.Table)
	bindArgs[amqp.QueueTTLArg] = ttl.Milliseconds()
	return func(channel *amqp.Channel) error {
		for _, key := range bindingKeys {
			err := channel.QueueBind(
				queueName,
				key,
				exName,
				false,
				bindArgs,
			)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func testNewConsumer(queueName, consumerName string, autoAck bool) func(*amqp.Channel) (string, <-chan amqp.Delivery, error) {
	return func(channel *amqp.Channel) (string, <-chan amqp.Delivery, error) {
		consumer, err := channel.Consume(
			queueName,
			consumerName,
			autoAck,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return "", nil, err
		}
		return consumerName, consumer, nil
	}
}
