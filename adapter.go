package Artifex

import (
	"sync"

	"github.com/gookit/goutil/maputil"
)

func NewPubSub[rMessage, sMessage any](opt *AdapterOption[rMessage, sMessage]) (pubsub *Adapter[rMessage, sMessage], err error) {
	pubsub = &Adapter[rMessage, sMessage]{
		pingpong:            opt.pingpong,
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		result:              make(chan error, 2),
		mqStopAll:           make([]chan error, 0),
		lifecycle:           opt.lifecycle(),
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
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		result:              make(chan error, 2),
		mqStopAll:           make([]chan error, 0),
		lifecycle:           opt.lifecycle(),
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
		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,
		identifier:          opt.identifier,
		appData:             make(maputil.Data),
		result:              make(chan error, 2),
		mqStopAll:           make([]chan error, 0),
		lifecycle:           opt.lifecycle(),
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

	// RegisterStop returns a channel for receiving the result of the Adapter
	// If the Adapter has been already Stopped,
	// it returns a channel containing an error message indicating the Adapter is closed.
	//
	// Once notified, the channel will be closed immediately.
	RegisterStop() <-chan error
	IsStop() bool
	Stop() error
}

type Adapter[rMessage, sMessage any] struct {
	handleRecv  HandleFunc[rMessage]
	adapterRecv func(IAdapter) (*rMessage, error)
	adapterSend func(IAdapter, *sMessage) error
	adapterStop func(IAdapter, *sMessage) error

	pingpong func(isStop func() bool) error

	fixupMaxRetrySecond int
	adapterFixup        func(IAdapter) error

	lifecycle *Lifecycle

	adpMutex   sync.RWMutex
	identifier string
	appData    maputil.Data

	result    chan error
	mqStopAll []chan error
	isStop    bool
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
		if adp.adapterFixup == nil {
			adp.result <- adp.pingpong(adp.IsStop)
			return
		}
		adp.result <- ReliableTask(
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
		if adp.adapterFixup == nil {
			adp.result <- adp.listen()
			return
		}
		adp.result <- ReliableTask(
			adp.listen,
			adp.IsStop,
			adp.fixupMaxRetrySecond,
			func() error { return adp.adapterFixup(adp) },
		)
	}()

	err = <-adp.result
	adp.Stop()
	go func() {
		adp.adpMutex.Lock()
		defer adp.adpMutex.Unlock()

		for _, notifyStop := range adp.mqStopAll {
			notifyStop <- err
			close(notifyStop)
		}
		adp.mqStopAll = make([]chan error, 0)
	}()
	return err
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
	adp.isStop = true

	adp.lifecycle.asyncTerminate(adp)
	err := adp.adapterStop(adp, message)
	adp.lifecycle.wait()
	return err
}

// RegisterStop returns a channel for receiving the result of the Adapter
// If the Adapter has been already Stopped,
// it returns a channel containing an error message indicating the Adapter is closed.
//
// Once notified, the channel will be closed immediately.
func (adp *Adapter[rMessage, sMessage]) RegisterStop() <-chan error {
	adp.adpMutex.Lock()
	defer adp.adpMutex.Unlock()

	ch := make(chan error, 1)
	if adp.isStop {
		ch <- ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
		close(ch)
		return ch
	}

	adp.mqStopAll = append(adp.mqStopAll, ch)
	return ch
}

func (adp *Adapter[rMessage, sMessage]) IsStop() bool {
	return adp.isStop
}

func (adp *Adapter[rMessage, sMessage]) Stop() error {
	var empty *sMessage
	return adp.StopWithMessage(empty)
}
