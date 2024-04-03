package Artifex

import (
	"sync"

	"github.com/gookit/goutil/maputil"
)

func NewPubSub[rMessage, sMessage any](opt *AdapterOption[rMessage, sMessage]) (pubsub *Adapter[rMessage, sMessage], err error) {
	pubsub = &Adapter[rMessage, sMessage]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		lifecycle:           opt.lifecycle(),
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		waitStop:            make(chan struct{}),
	}
	pubsub.handleRecv = opt.handleRecv
	pubsub.adapterRecv = opt.adapterRecv
	pubsub.adapterSend = opt.adapterSend
	pubsub.adapterStop = opt.adapterStop
	return pubsub, pubsub.init()
}

func NewPublisher[sMessage any](opt *AdapterOption[struct{}, sMessage]) (publisher *Adapter[struct{}, sMessage], err error) {
	publisher = &Adapter[struct{}, sMessage]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		lifecycle:           opt.lifecycle(),
		adpMutex:            sync.RWMutex{},
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		waitStop:            make(chan struct{}),
	}
	publisher.handleRecv = nil
	publisher.adapterRecv = nil
	publisher.adapterSend = opt.adapterSend
	publisher.adapterStop = opt.adapterStop
	return publisher, publisher.init()
}

func NewSubscriber[rMessage any](opt *AdapterOption[rMessage, struct{}]) (subscriber *Adapter[rMessage, struct{}], err error) {
	subscriber = &Adapter[rMessage, struct{}]{
		pingpong:            opt.pingpong,
		recvResult:          make(chan error, 2),
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		lifecycle:           opt.lifecycle(),
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		waitStop:            make(chan struct{}),
	}
	subscriber.handleRecv = opt.handleRecv
	subscriber.adapterRecv = opt.adapterRecv
	subscriber.adapterSend = nil
	subscriber.adapterStop = opt.adapterStop
	return subscriber, subscriber.init()
}

type IAdapter interface {
	Identifier() string
	Query(query func(id string, appData maputil.Data))
	Update(update func(id *string, appData maputil.Data))
	AddTerminate(terminates ...func(adp IAdapter))

	Stop() error

	// IsStop is used for polling
	IsStop() bool

	// WaitStop is used for event push
	WaitStop() chan struct{}
}

type Adapter[rMessage, sMessage any] struct {
	handleRecv  HandleFunc[rMessage]
	adapterRecv func(IAdapter) (*rMessage, error)
	adapterSend func(IAdapter, *sMessage) error
	adapterStop func(IAdapter, *sMessage) error

	// WaitPingSendPong or SendPingWaitPong
	pingpong   func(isStop func() bool) error
	recvResult chan error

	fixupMaxRetrySecond int
	adapterFixup        func(IAdapter) error

	lifecycle *Lifecycle

	adpMutex   sync.RWMutex
	identifier string
	appData    maputil.Data

	isStop   bool
	waitStop chan struct{}
}

func (adp *Adapter[rMessage, sMessage]) init() error {
	err := adp.lifecycle.initialize(adp)
	if err != nil {
		return err
	}

	go func() {
		if adp.pingpong == nil {
			return
		}

		var Err error
		defer func() {
			if !adp.isStop {
				adp.Stop()
			}
			adp.recvResult <- Err
		}()

		if adp.adapterFixup == nil {
			Err = adp.pingpong(adp.IsStop)
			return
		}
		Err = ReliableTask(
			func() error { return adp.pingpong(adp.IsStop) },
			adp.IsStop,
			adp.fixupMaxRetrySecond,
			func() error { return adp.adapterFixup(adp) },
		)
	}()

	return nil
}

func (adp *Adapter[rMessage, sMessage]) Identifier() string {
	return adp.identifier
}

func (adp *Adapter[rMessage, sMessage]) Query(query func(id string, appData maputil.Data)) {
	adp.adpMutex.RLock()
	defer adp.adpMutex.RUnlock()
	query(adp.identifier, adp.appData)
}

func (adp *Adapter[rMessage, sMessage]) Update(update func(id *string, appData maputil.Data)) {
	adp.adpMutex.Lock()
	defer adp.adpMutex.Unlock()
	update(&adp.identifier, adp.appData)
}

func (adp *Adapter[rMessage, sMessage]) AddTerminate(terminates ...func(adp IAdapter)) {
	adp.lifecycle.AddTerminate(terminates...)
}

func (adp *Adapter[rMessage, sMessage]) Listen() (err error) {
	if adp.isStop {
		return ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
	}

	go func() {
		var Err error
		defer func() {
			if !adp.isStop {
				adp.Stop()
			}
			adp.recvResult <- Err
		}()

		if adp.adapterFixup == nil {
			Err = adp.listen()
			return
		}
		Err = ReliableTask(
			adp.listen,
			adp.IsStop,
			adp.fixupMaxRetrySecond,
			func() error { return adp.adapterFixup(adp) },
		)
	}()

	return <-adp.recvResult
}

func (adp *Adapter[rMessage, sMessage]) listen() error {
	for !adp.isStop {
		message, err := adp.adapterRecv(adp)

		if adp.isStop {
			return nil
		}

		if err != nil {
			return err
		}

		adp.handleRecv(message, nil)
	}
	return nil
}

func (adp *Adapter[rMessage, sMessage]) Send(messages ...*sMessage) error {
	if adp.isStop {
		return ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
	}

	for _, message := range messages {
		err := adp.adapterSend(adp, message)
		if err != nil {
			return err
		}
	}

	return nil
}

func (adp *Adapter[rMessage, sMessage]) StopWithMessage(message *sMessage) error {
	adp.adpMutex.Lock()
	defer adp.adpMutex.Unlock()
	if adp.isStop {
		return ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
	}

	adp.lifecycle.asyncTerminate(adp)
	err := adp.adapterStop(adp, message)
	adp.lifecycle.wait()

	// 確保所有關閉任務都執行結束
	// 才定義為 isStop=true
	adp.isStop = true
	close(adp.waitStop)
	return err
}

func (adp *Adapter[rMessage, sMessage]) Stop() error {
	var empty *sMessage
	return adp.StopWithMessage(empty)
}

func (adp *Adapter[rMessage, sMessage]) IsStop() bool {
	return adp.isStop
}

func (adp *Adapter[rMessage, sMessage]) WaitStop() chan struct{} {
	return adp.waitStop
}