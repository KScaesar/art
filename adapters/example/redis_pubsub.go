package main

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

func BuildRedisInfrastructure() (any, error) {
	return nil, nil
}

func NewRedisPingPong() (pp Artifex.PingPong, waitNotify chan error) {
	waitNotify = make(chan error, 1)
	return Artifex.PingPong{
		IsSendPingWaitPong: false,
		SendFunc: func() error {
			return nil
		},
		WaitNotify: waitNotify,
		WaitSecond: 30,
	}, waitNotify
}

type RedisFactory struct {
	RecvMux *RedisIngressMux
	SendMux *RedisEgressMux
}

//

func NewRedisPubSubHub() *Artifex.AdapterHub[RedisPubSub] {
	stop := func(adapter *RedisPubSub) error {
		return adapter.Stop()
	}
	return Artifex.NewAdapterHub(stop)
}

type RedisPubSub = Artifex.PubSub[RedisIngress, RedisEgress]

func (f *RedisFactory) CreatePubSub() (*RedisPubSub, error) {
	var mu sync.Mutex

	pp, waitNotify := NewRedisPingPong()
	_ = waitNotify

	pubsub := &RedisPubSub{
		HandleRecv: f.RecvMux.HandleMessage,
		AdapterRecv: func() (*RedisIngress, error) {
			return NewRedisIngress(), nil
		},
		AdapterSend: func(message *RedisEgress) error {
			mu.Lock()
			defer mu.Unlock()
			return f.SendMux.HandleMessage(message, nil)
		},
		AdapterStop: func(message *RedisEgress) error {
			mu.Lock()
			defer mu.Unlock()
			return nil
		},
		Fixup: func() error {
			return nil
		},
		FixupMaxRetrySecond: 0,

		Identifier: "",
	}

	life := Artifex.Lifecycle{}
	pubsub.Lifecycle = life

	go func() {
		err := pubsub.PingPong(pp)
		if err != nil {
			_ = err
		}
	}()

	return pubsub, nil
}

//

func NewRedisPublisherHub() *Artifex.AdapterHub[RedisPublisher] {
	stop := func(adapter *RedisPublisher) error {
		return adapter.Stop()
	}
	return Artifex.NewAdapterHub(stop)
}

type RedisPublisher = Artifex.Publisher[RedisEgress]

func (f *RedisFactory) CreatePublisher() (*RedisPublisher, error) {
	var mu sync.Mutex

	pub := &RedisPublisher{
		AdapterSend: func(message *RedisEgress) error {
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

func NewRedisSubscriberHub() *Artifex.AdapterHub[RedisSubscriber] {
	stop := func(adapter *RedisSubscriber) error {
		return adapter.Stop()
	}
	return Artifex.NewAdapterHub(stop)
}

type RedisSubscriber = Artifex.Subscriber[RedisIngress]

func (f *RedisFactory) CreateSubscriber() (*RedisSubscriber, error) {
	sub := &RedisSubscriber{
		HandleRecv: f.RecvMux.HandleMessage,
		AdapterRecv: func() (*RedisIngress, error) {
			return NewRedisIngress(), nil
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
	sub.Lifecycle = life

	return sub, nil
}
