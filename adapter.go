package Artifex

import (
	"sync"
)

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
	pingpong   func() error
	recvResult chan error

	fixupMaxRetrySecond int
	adapterFixup        func(IAdapter) error

	mu         sync.RWMutex
	identifier string

	lifecycle *Lifecycle
	isStopped bool
	waitStop  chan struct{}
}

func (adp *Adapter[Ingress, Egress]) init() error {
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
			Err = adp.pingpong()
			return
		}
		Err = ReliableTask(
			adp.pingpong,
			adp.IsStopped,
			adp.fixupMaxRetrySecond,
			func() error { return adp.adapterFixup(adp) },
		)
	}()

	return nil
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
			func() error { return adp.adapterFixup(adp) },
		)
	}()

	return <-adp.recvResult
}

func (adp *Adapter[Ingress, Egress]) listen() error {
	for !adp.isStopped {
		ingress, err := adp.adapterRecv(adp)

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
		err := adp.adapterSend(adp, egress)
		if err != nil {
			return err
		}
	}

	return nil
}

func (adp *Adapter[Ingress, Egress]) Stop() error {
	adp.mu.Lock()
	defer adp.mu.Unlock()
	if adp.isStopped {
		return ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
	}

	adp.lifecycle.asyncTerminate(adp)
	adp.isStopped = true
	close(adp.waitStop)
	err := adp.adapterStop(adp)
	adp.lifecycle.wait()
	return err
}

func (adp *Adapter[Ingress, Egress]) IsStopped() bool {
	return adp.isStopped
}

func (adp *Adapter[Ingress, Egress]) WaitStop() chan struct{} {
	return adp.waitStop
}
