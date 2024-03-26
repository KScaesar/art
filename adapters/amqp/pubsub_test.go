package rabbit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"talent.com/naming/common/adapter"
)

func TestFactory_CreateSubscriber(t *testing.T) {
	record := func(consumerName string) func(message *Ingress, _ *adapter.RouteParam) error {
		return func(message *Ingress, _ *adapter.RouteParam) error {
			message.Logger.Info("record %v", consumerName)
			return nil
		}
	}
	mux := NewIngressMux().
		Transform(HandleDecodeMetadata()).
		PreMiddleware(HandleSetMessageId()).
		SetErrorHandler(PrintError())

	mux.Handler("packet.pay.order", record("red_point"))
	mux.Handler("packet.pay.mf.im", record("red_point"))
	mux.Handler("packet.kf.order", record("red_point"))
	mux.Handler("notify_single_user.eepay", record("eepay"))

	// nodeId, _ := os.Hostname()
	nodeId := adapter.GenerateRandomCode(4)

	url := "amqp://guest:guest@127.0.0.1:5672"
	connection, err := NewConnection(url)
	require.NoError(t, err)
	mu := &sync.Mutex{}
	factoryAll := []*Factory{
		{
			ConnectionMutex: mu,
			AmqpUri:         url,
			Connection:      &connection,
			SetupQos:        SetupQos(1),
			SetupEx:         SetupEx("test-ex", "topic"),
			SetupQueue:      SetupTemporaryQueue("test-q", 10*time.Second),
			SetupBind:       SetupBind("test-q", "test-ex", []string{"test"}),
			NewConsumer:     NewConsumer("test-q", "test-c", true),
			RecvMux:         mux,
		},
		{
			ConnectionMutex: mu,
			AmqpUri:         url,
			Connection:      &connection,
			SetupQos:        SetupQos(1),
			SetupEx:         SetupEx("black3", "topic"),
			SetupQueue:      SetupQueue("q_naming_packet"),
			SetupBind:       SetupBind("q_naming_packet", "black3", []string{"packet.pay.order", "packet.pay.mf.im", "packet.kf.order"}),
			NewConsumer:     NewConsumer("q_naming_packet", "red_point"+"."+nodeId, true),
			RecvMux:         mux,
		},
		{
			ConnectionMutex: mu,
			AmqpUri:         url,
			Connection:      &connection,
			SetupQos:        SetupQos(1),
			SetupEx:         SetupEx("black3", "topic"),
			SetupQueue:      SetupTemporaryQueue("EEPay.node_id="+nodeId, 5*time.Minute),
			SetupBind:       SetupTemporaryBind("EEPay.node_id="+nodeId, "black3", []string{"notify_single_user.*"}, 5*time.Minute),
			NewConsumer:     NewConsumer("EEPay.node_id="+nodeId, "eepay"+"."+nodeId, true),
			RecvMux:         mux,
		},
	}

	hub := NewSubscriberHub()
	for _, factory := range factoryAll {
		subscriber, err := factory.CreateSubscriber()
		if err != nil {
			hub.StopAll()
			return
		}
		require.NoError(t, err)

		go subscriber.Listen()

		err = hub.Join(subscriber.Identifier, subscriber)
		require.NoError(t, err)
	}

	shutdown := adapter.SetupShutdown(nil, func() (serviceName string) {
		hub.StopAll()
		return "amqp"
	})
	shutdown.Wait()
	// shutdown.WaitAfter(5)
}
