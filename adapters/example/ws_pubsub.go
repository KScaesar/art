package main

import (
	"sync"

	"github.com/KScaesar/Artifex"
)

func BuildWsInfrastructure() (any, error) {
	return nil, nil
}

func NewWsPingPong() Artifex.PingPong {
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

type WsFactory struct {
	RecvMux *WsIngressMux
	SendMux *WsEgressMux
}

//

func NewWsPubSubHub() *Artifex.Hub[WsPubSub] {
	stop := func(adapter *WsPubSub) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

type WsPubSub = Artifex.PubSub[WsIngress, WsEgress]

func (f *WsFactory) CreatePubSub() (*WsPubSub, error) {
	var mu sync.Mutex

	pubsub := &WsPubSub{
		HandleRecv: f.RecvMux.HandleMessage,
		Identifier: "",
	}

	pubsub.AdapterRecv = func() (*WsIngress, error) {
		return NewWsIngress(), nil
	}

	pubsub.AdapterSend = func(message *WsEgress) error {
		mu.Lock()
		defer mu.Unlock()
		return f.SendMux.HandleMessage(message, nil)
	}

	pubsub.AdapterStop = func(message *WsEgress) error {
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

	pp := NewWsPingPong()
	go func() {
		err := pubsub.PingPong(pp)
		if err != nil {
			_ = err
		}
	}()

	return pubsub, nil
}

//

func NewWsPublisherHub() *Artifex.Hub[WsPublisher] {
	stop := func(adapter *WsPublisher) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

type WsPublisher = Artifex.Publisher[WsEgress]

func (f *WsFactory) CreatePublisher() (*WsPublisher, error) {
	var mu sync.Mutex

	pub := &WsPublisher{
		Identifier: "",
	}

	pub.AdapterSend = func(message *WsEgress) error {
		mu.Lock()
		defer mu.Unlock()
		return f.SendMux.HandleMessage(message, nil)
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

func NewWsSubscriberHub() *Artifex.Hub[WsSubscriber] {
	stop := func(adapter *WsSubscriber) error {
		return adapter.Stop()
	}
	return Artifex.NewHub(stop)
}

type WsSubscriber = Artifex.Subscriber[WsIngress]

func (f *WsFactory) CreateSubscriber() (*WsSubscriber, error) {

	sub := &WsSubscriber{
		HandleRecv: f.RecvMux.HandleMessage,
		Identifier: "",
	}

	sub.AdapterRecv = func() (*WsIngress, error) {
		return NewWsIngress(), nil
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
