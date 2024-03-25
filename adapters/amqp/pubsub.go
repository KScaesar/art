package rabbit

import (
	"errors"
	"sync"

	"github.com/KScaesar/Artifex"
	amqp "github.com/rabbitmq/amqp091-go"
)

func NewConnection(url string) (*amqp.Connection, error) {
	return amqp.Dial(url)
}

type Factory struct {
	// connection
	AmqpUri         string
	ConnectionMutex *sync.Mutex
	Connection      **amqp.Connection

	SetupQos   func(ch *amqp.Channel) error
	SetupEx    func(ch *amqp.Channel) error
	SetupQueue func(ch *amqp.Channel) error
	SetupBind  func(ch *amqp.Channel) error

	NewConsumer func(ch *amqp.Channel) (string, <-chan amqp.Delivery, error)
	RecvMux     *IngressMux

	SendMux *EgressMux

	Logger Artifex.Logger
}

//

func NewSubscriberHub() *Artifex.AdapterHub[Subscriber] {
	stop := func(Artifex *Subscriber) error {
		return Artifex.Stop()
	}
	return Artifex.NewAdapterHub(stop)
}

type Subscriber = Artifex.Subscriber[Ingress]

func (f *Factory) CreateSubscriber() (*Subscriber, error) {
	channel, err := (*f.Connection).Channel()
	if err != nil {
		return nil, err
	}
	if err := f.setupAmqp(channel); err != nil {
		return nil, err
	}
	consumerName, consumer, err := f.NewConsumer(channel)
	if err != nil {
		return nil, err
	}

	logger := f.Logger.
		WithKeyValue("amqp_id", Artifex.GenerateRandomCode(6)).
		WithKeyValue("amqp_sub", consumerName)
	logger.Info("create amqp subscriber success!")

	connCloseNotify := (*f.Connection).NotifyClose(make(chan *amqp.Error, 1))
	chCloseNotify := channel.NotifyClose(make(chan *amqp.Error, 1))
	consumerIsClose := false
	retryCnt := 0
	sub := &Subscriber{
		HandleRecv: f.RecvMux.HandleMessage,
		AdapterRecv: func() (*Ingress, error) {
			amqpMsg, ok := <-consumer
			if !ok {
				consumerIsClose = true
				err := errors.New("amqp consumer close")
				logger.Error("%v", err)
				return nil, err
			}
			logger.Info("receive msg: msgType=%q: key=%q: body=%q",
				amqpMsg.Type, amqpMsg.RoutingKey, string(amqpMsg.Body))
			return NewIngress(amqpMsg, logger), nil
		},
		AdapterStop: func() error {
			logger.Info("active stop")
			return channel.Close()
		},
		Fixup: func() error {
			select {
			case Err := <-connCloseNotify:
				if Err != nil {
					logger.Error("amqp connection close: %v", Err)
				}
			case Err := <-chCloseNotify:
				if Err != nil {
					logger.Error("amqp channel close: %v", Err)
				}
			default:
			}

			retryCnt++
			logger.Info("retry %v times", retryCnt)

			if (*f.Connection).IsClosed() {
				err := f.concurrencySafe(func() (err error) {
					if !(*f.Connection).IsClosed() {
						return nil
					}
					logger.Info("retry amqp connection start")
					connection, err := NewConnection(f.AmqpUri)
					if err != nil {
						logger.Error("retry amqp connection fail: %v", err)
						return err
					}
					logger.Info("retry amqp connection success")
					(*f.Connection) = connection
					connCloseNotify = (*f.Connection).NotifyClose(make(chan *amqp.Error, 1))
					return nil
				})
				if err != nil {
					return err
				}
			}

			if channel.IsClosed() {
				logger.Info("retry amqp channel start")
				channel, err = (*f.Connection).Channel()
				if err != nil {
					logger.Error("retry amqp channel fail: %v", err)
					return err
				}
				logger.Info("retry amqp channel success")
				chCloseNotify = channel.NotifyClose(make(chan *amqp.Error, 1))
			}

			if consumerIsClose {
				logger.Info("retry amqp consumer start")
				if err := f.setupAmqp(channel); err != nil {
					logger.Error("retry setup amqp fail: %v", err)
					return err
				}
				_, freshConsumer, err := f.NewConsumer(channel)
				if err != nil {
					logger.Error("retry amqp consumer fail: %v", err)
					return err
				}
				logger.Info("retry amqp consumer success")
				consumerIsClose = false
				consumer = freshConsumer
				retryCnt = 0
				return nil
			}

			logger.Info("'%v' retry ???")
			return nil
		},
		FixupMaxRetrySecond: 0,
		Identifier:          consumerName,
	}

	return sub, nil
}

//

func NewPublisherHub() *Artifex.AdapterHub[Publisher] {
	stop := func(Artifex *Publisher) error {
		return Artifex.Stop()
	}
	return Artifex.NewAdapterHub(stop)
}

type Publisher = Artifex.Publisher[Egress]

func (f *Factory) CreatePublisher() (*Publisher, error) {
	var mu sync.Mutex

	pub := &Publisher{
		AdapterSend: func(message *Egress) error {
			mu.Lock()
			defer mu.Unlock()
			return f.SendMux.HandleMessage(message, nil)
		},
		AdapterStop: func() error {
			return nil
		},
		Fixup: func() error {
			return nil
		},
		FixupMaxRetrySecond: 0,

		Identifier: "",
	}

	life := Artifex.Lifecycle{}
	pub.Lifecycle = life

	return pub, nil
}

//

func (f *Factory) setupAmqp(channel *amqp.Channel) error {
	if err := f.SetupQos(channel); err != nil {
		return err
	}
	if err := f.SetupEx(channel); err != nil {
		return err
	}
	if err := f.SetupQueue(channel); err != nil {
		return err
	}
	if err := f.SetupBind(channel); err != nil {
		return err
	}
	return nil
}

func (f *Factory) concurrencySafe(action func() error) error {
	f.ConnectionMutex.Lock()
	defer f.ConnectionMutex.Unlock()
	return action()
}
