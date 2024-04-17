package Artifex

import (
	"sync"

	"github.com/gookit/goutil/maputil"
)

type IAdapter interface {
	Identifier() string
	Query(query func(id string, appData maputil.Data))
	Update(update func(id *string, appData maputil.Data))

	OnStop(terminates ...func(adp IAdapter))
	Stop() error
	IsStopped() bool         // IsStopped is used for polling
	WaitStop() chan struct{} // WaitStop is used for event push
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

	adpMutex   sync.RWMutex
	identifier string
	appData    maputil.Data

	lifecycle *Lifecycle
	isStopped bool
	waitStop  chan struct{}
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
			if !adp.isStopped {
				adp.Stop()
			}
			adp.recvResult <- Err
		}()

		if adp.adapterFixup == nil {
			Err = adp.pingpong(adp.IsStopped)
			return
		}
		Err = ReliableTask(
			func() error { return adp.pingpong(adp.IsStopped) },
			adp.IsStopped,
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

func (adp *Adapter[rMessage, sMessage]) OnStop(terminates ...func(adp IAdapter)) {
	adp.lifecycle.OnStop(terminates...)
}

func (adp *Adapter[rMessage, sMessage]) Listen() (err error) {
	if adp.isStopped {
		return ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
	}

	go func() {
		var Err error
		defer func() {
			if !adp.isStopped {
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
			adp.IsStopped,
			adp.fixupMaxRetrySecond,
			func() error { return adp.adapterFixup(adp) },
		)
	}()

	return <-adp.recvResult
}

func (adp *Adapter[rMessage, sMessage]) listen() error {
	for !adp.isStopped {
		message, err := adp.adapterRecv(adp)

		if adp.isStopped {
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
	if adp.isStopped {
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
	if adp.isStopped {
		return ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
	}

	adp.lifecycle.asyncTerminate(adp)
	adp.isStopped = true
	close(adp.waitStop)
	err := adp.adapterStop(adp, message)
	adp.lifecycle.wait()

	return err
}

func (adp *Adapter[rMessage, sMessage]) Stop() error {
	var empty *sMessage
	return adp.StopWithMessage(empty)
}

func (adp *Adapter[rMessage, sMessage]) IsStopped() bool {
	return adp.isStopped
}

func (adp *Adapter[rMessage, sMessage]) WaitStop() chan struct{} {
	return adp.waitStop
}
