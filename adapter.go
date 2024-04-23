package Artifex

import (
	"sync/atomic"
)

func NewAdapterHub() *Hub[IAdapter] {
	return NewHub(func(adp IAdapter) {
		adp.Stop()
	})
}

type AdapterHub interface {
	Join(adapterId string, adp IAdapter) error
	RemoveOne(filter func(IAdapter) bool)
}

type IAdapter interface {
	Identifier() string
	Log() Logger
	SetLog(Logger)
	OnStop(terminates ...func(adp IAdapter))
	Stop() error
	IsStopped() bool         // IsStopped is used for polling
	WaitStop() chan struct{} // WaitStop is used for event push
}

type Adapter[Ingress, Egress any] struct {
	handleRecv  HandleFunc[Ingress]
	adapterRecv func(IAdapter) (*Ingress, error)
	adapterSend func(IAdapter, *Egress) error
	adapterStop func(IAdapter) error

	// WaitPingSendPong or SendPingWaitPong
	pp         func() error
	recvResult chan error

	fixupMaxRetrySecond int
	adapterFixup        func(IAdapter) error

	identifier  string
	logger      Logger
	application IAdapter
	hub         AdapterHub

	lifecycle *Lifecycle
	isStopped atomic.Bool
	waitStop  chan struct{}
}

func (adp *Adapter[Ingress, Egress]) Log() Logger {
	return adp.logger
}

func (adp *Adapter[Ingress, Egress]) SetLog(logger Logger) {
	adp.logger = logger
}

func (adp *Adapter[Ingress, Egress]) pingpong() {
	go func() {
		if adp.pp == nil {
			return
		}

		var Err error
		defer func() {
			if !adp.isStopped.Load() {
				adp.Stop()
			}
			if Err != nil {
				adp.logger.Error("Artifex Adapter pingpong: %v", Err.Error())
			}
			adp.recvResult <- Err
		}()

		if adp.adapterFixup == nil {
			Err = adp.pp()
			return
		}
		Err = ReliableTask(
			adp.pp,
			adp.IsStopped,
			adp.fixupMaxRetrySecond,
			func() error { return adp.adapterFixup(adp.application) },
		)
	}()
}

func (adp *Adapter[Ingress, Egress]) Identifier() string { return adp.identifier }

func (adp *Adapter[Ingress, Egress]) OnStop(terminates ...func(adp IAdapter)) {
	adp.lifecycle.OnStop(terminates...)
}

func (adp *Adapter[Ingress, Egress]) Listen() (err error) {
	if adp.isStopped.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex Adapter Listen")
	}

	go func() {
		var Err error
		defer func() {
			if !adp.isStopped.Load() {
				adp.Stop()
			}
			if Err != nil {
				adp.logger.Error("Artifex Adapter Listen: %v", Err.Error())
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
			func() error { return adp.adapterFixup(adp.application) },
		)
	}()

	return <-adp.recvResult
}

func (adp *Adapter[Ingress, Egress]) listen() error {
	for !adp.isStopped.Load() {
		ingress, err := adp.adapterRecv(adp.application)

		if adp.isStopped.Load() {
			return nil
		}

		if err != nil {
			return err
		}

		adp.handleRecv(ingress, nil)
	}
	return nil
}

func (adp *Adapter[Ingress, Egress]) Send(messages ...*Egress) error {
	if adp.isStopped.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex Adapter Send")
	}

	for _, egress := range messages {
		err := adp.adapterSend(adp.application, egress)
		if err != nil {
			return err
		}
	}

	return nil
}

func (adp *Adapter[Ingress, Egress]) Stop() error {
	if adp.isStopped.Swap(true) {
		return ErrorWrapWithMessage(ErrClosed, "Artifex Adapter Stop")
	}

	if adp.hub != nil {
		adp.hub.RemoveOne(func(adapter IAdapter) bool {
			return adapter == adp.application
		})
	}

	adp.lifecycle.asyncTerminate(adp.application)
	err := adp.adapterStop(adp.application)
	adp.lifecycle.wait()
	close(adp.waitStop)
	return err
}

func (adp *Adapter[Ingress, Egress]) IsStopped() bool { return adp.isStopped.Load() }

func (adp *Adapter[Ingress, Egress]) WaitStop() chan struct{} { return adp.waitStop }
