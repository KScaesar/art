package infra

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

func BuildKafkaInfrastructure() (any, error) {
	return nil, nil
}

func NewKafkaPingPong() Artifex.PingPong {
	waitNotify := make(chan error, 1)

	return Artifex.PingPong{
		IsSendPingWaitPong: true,
		SendFunc: func() error {
			return nil
		},
		WaitNotify: waitNotify,
		WaitSecond: 30,
	}
}

type KafkaFactory struct {
	RecvMux *KafkaIngressMux
	SendMux *KafkaEgressMux
}

//

func NewKafkaPubSubHub() *Artifex.Hub[KafkaPubSub] {
	stop := func(adapter *KafkaPubSub) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

type KafkaPubSub = Artifex.PubSub[KafkaIngress, KafkaEgress]

func (f *KafkaFactory) CreatePubSub() (*KafkaPubSub, error) {
	var mu sync.Mutex

	pubsub := &KafkaPubSub{
		HandleRecv: f.RecvMux.HandleMessage,
		Identifier: "",
	}

	pubsub.AdapterRecv = func() (*KafkaIngress, error) {
		return NewKafkaIngress(), nil
	}

	pubsub.AdapterSend = func(message *KafkaEgress) error {
		err := f.SendMux.HandleMessage(message, nil)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		return nil
	}

	pubsub.AdapterStop = func(message *KafkaEgress) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	}

	pubsub.FixupMaxRetrySecond = 0
	pubsub.Fixup = func() error {
		return nil
	}

	life := Artifex.Lifecycle{}
	pubsub.Lifecycle = life

	pp := NewKafkaPingPong()
	go func() {
		err := pubsub.PingPong(pp)
		if err != nil {
			_ = err
		}
	}()

	return pubsub, nil
}

//

func NewKafkaPublisherHub() *Artifex.Hub[KafkaPublisher] {
	stop := func(adapter *KafkaPublisher) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

type KafkaPublisher = Artifex.Publisher[KafkaEgress]

func (f *KafkaFactory) CreatePublisher() (*KafkaPublisher, error) {
	var mu sync.Mutex

	pub := &KafkaPublisher{
		Identifier: "",
	}

	pub.AdapterSend = func(message *KafkaEgress) error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	}

	pub.AdapterStop = func() error {
		mu.Lock()
		defer mu.Unlock()
		return nil
	}

	pub.FixupMaxRetrySecond = 0
	pub.Fixup = func() error {
		return nil
	}

	life := Artifex.Lifecycle{}
	pub.Lifecycle = life

	return pub, nil
}

//

func NewKafkaSubscriberHub() *Artifex.Hub[KafkaSubscriber] {
	stop := func(adapter *KafkaSubscriber) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

type KafkaSubscriber = Artifex.Subscriber[KafkaIngress]

func (f *KafkaFactory) CreateSubscriber() (*KafkaSubscriber, error) {

	sub := &KafkaSubscriber{
		HandleRecv: f.RecvMux.HandleMessage,
		Identifier: "",
	}

	sub.AdapterRecv = func() (*KafkaIngress, error) {
		return NewKafkaIngress(), nil
	}

	sub.AdapterStop = func() error {
		return nil
	}

	sub.FixupMaxRetrySecond = 0
	sub.Fixup = func() error {
		return nil
	}

	life := Artifex.Lifecycle{}
	sub.Lifecycle = life

	return sub, nil
}
