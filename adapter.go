package Artifex

import (
	"sync"
)

func NewAdapterHub(stop func(adp IAdapter)) AdapterHub {
	return NewHub(stop)
}

type AdapterHub interface {
	Join(adapterId string, adp IAdapter) error
	RemoveOne(filter func(IAdapter) bool)
}

type IAdapter interface {
	Identifier() string
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

	mu          sync.RWMutex
	identifier  string
	application IAdapter
	hub         AdapterHub

	lifecycle *Lifecycle
	isStopped bool
	waitStop  chan struct{}
}

func (adp *Adapter[Ingress, Egress]) pingpong() {
	go func() {
		if adp.pp == nil {
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

func (adp *Adapter[Ingress, Egress]) Identifier() string {
	return adp.identifier
}

func (adp *Adapter[Ingress, Egress]) OnStop(terminates ...func(adp IAdapter)) {
	adp.lifecycle.OnStop(terminates...)
}

func (adp *Adapter[Ingress, Egress]) Listen() (err error) {
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
			func() error { return adp.adapterFixup(adp.application) },
		)
	}()

	return <-adp.recvResult
}

func (adp *Adapter[Ingress, Egress]) listen() error {
	for !adp.isStopped {
		ingress, err := adp.adapterRecv(adp.application)

		if adp.isStopped {
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
	if adp.isStopped {
		return ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
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
	adp.mu.Lock()

	if adp.isStopped {
		adp.mu.Unlock()
		return ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
	}

	if adp.hub != nil {
		adp.mu.Unlock()
		adp.hub.RemoveOne(func(adapter IAdapter) bool {
			return adapter == adp.application
		})
		adp.mu.Lock()
	}
	defer adp.mu.Unlock()

	adp.lifecycle.asyncTerminate(adp.application)
	adp.isStopped = true
	close(adp.waitStop)
	err := adp.adapterStop(adp.application)
	adp.lifecycle.wait()
	return err
}

func (adp *Adapter[Ingress, Egress]) IsStopped() bool {
	return adp.isStopped
}

func (adp *Adapter[Ingress, Egress]) WaitStop() chan struct{} {
	return adp.waitStop
}
