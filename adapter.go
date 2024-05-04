package art

import (
	"reflect"
	"sync/atomic"
)

type AdapterHub interface {
	Join(adapterId string, adp IAdapter) error
	RemoveOne(filter func(IAdapter) bool)
}

type Prosumer interface {
	Producer
	Consumer
}

type Producer interface {
	IAdapter
	Send(messages ...*Message) error
	RawSend(messages ...*Message) error
}

type Consumer interface {
	IAdapter
	Listen() (err error)
}

type IAdapter interface {
	Identifier() string
	Log() Logger
	SetLog(Logger)
	OnDisconnect(terminates ...func(adp IAdapter))
	Stop() error
	IsStopped() bool         // IsStopped is used for polling
	WaitStop() chan struct{} // WaitStop is used for event push
	RawInfra() any
}

type Adapter struct {
	ingressMux *Mux
	rawRecv    func(Logger) (*Message, error)

	egressMux *Mux
	rawSend   func(Logger, *Message) error

	rawStop func(Logger) error

	fixupMaxRetrySecond int
	rawFixup            func(IAdapter) error

	// WaitPingSendPong or SendPingWaitPong
	pp func() error

	identifier  string
	logger      Logger
	application IAdapter
	lifecycle   *Lifecycle
	rawInfra    any
	hub         AdapterHub
	recvResult  chan error
	isStopped   atomic.Bool
	waitStop    chan struct{}
}

func (adp *Adapter) Log() Logger {
	return adp.logger
}

func (adp *Adapter) SetLog(logger Logger) {
	adp.logger = logger
}

func (adp *Adapter) pingpong() {
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
				adp.logger.Error("art.Adapter pingpong: %v", Err)
			}
			adp.recvResult <- Err
		}()

		if adp.rawFixup == nil {
			Err = adp.pp()
			return
		}
		Err = ReliableTask(
			adp.pp,
			adp.IsStopped,
			adp.fixupMaxRetrySecond,
			func() error { return adp.rawFixup(adp.application) },
		)
	}()
}

func (adp *Adapter) Identifier() string { return adp.identifier }

func (adp *Adapter) OnDisconnect(terminates ...func(adp IAdapter)) {
	adp.lifecycle.OnDisconnect(terminates...)
}

func (adp *Adapter) Listen() (err error) {
	if adp.rawRecv == nil {
		return nil
	}

	if adp.isStopped.Load() {
		return ErrorWrapWithMessage(ErrClosed, "art.Adapter Listen")
	}

	go func() {
		var Err error
		defer func() {
			if !adp.isStopped.Load() {
				adp.Stop()
			}
			if Err != nil {
				adp.logger.Error("art.Adapter Listen: %v", Err)
			}
			adp.recvResult <- Err
		}()

		if adp.rawFixup == nil {
			Err = adp.listen()
			return
		}
		Err = ReliableTask(
			adp.listen,
			adp.IsStopped,
			adp.fixupMaxRetrySecond,
			func() error { return adp.rawFixup(adp.application) },
		)
	}()

	return <-adp.recvResult
}

func (adp *Adapter) listen() error {
	for !adp.isStopped.Load() {
		ingress, err := adp.rawRecv(adp.logger)

		if adp.isStopped.Load() {
			return nil
		}

		if err != nil {
			return err
		}

		err = adp.ingressMux.HandleMessage(ingress, adp.application)
		if err != nil {

		}
	}
	return nil
}

func (adp *Adapter) Send(messages ...*Message) error {
	if adp.isStopped.Load() {
		return ErrorWrapWithMessage(ErrClosed, "art.Adapter Send")
	}

	for _, egress := range messages {
		if adp.egressMux == nil {
			return nil
		}

		err := adp.egressMux.HandleMessage(egress, adp.application)
		if err != nil {
			return err
		}
	}
	return nil
}

func (adp *Adapter) RawSend(messages ...*Message) error {
	if adp.isStopped.Load() {
		return ErrorWrapWithMessage(ErrClosed, "art.Adapter RawSend")
	}

	for _, egress := range messages {
		if adp.rawSend == nil {
			return nil
		}

		err := adp.rawSend(adp.logger, egress)
		if err != nil {
			return err
		}
	}
	return nil
}

func (adp *Adapter) Stop() error {
	if adp.isStopped.Swap(true) {
		return ErrorWrapWithMessage(ErrClosed, "repeated execute stop")
	}

	err := adp.rawStop(adp.logger)

	if !reflect.ValueOf(adp.hub).IsNil() {
		adp.hub.RemoveOne(func(adapter IAdapter) bool { return adapter == adp.application })
	}

	adp.lifecycle.asyncTerminate(adp.application)
	adp.lifecycle.wait()

	close(adp.waitStop)
	return err
}

func (adp *Adapter) IsStopped() bool { return adp.isStopped.Load() }

func (adp *Adapter) WaitStop() chan struct{} { return adp.waitStop }

func (adp *Adapter) RawInfra() any {
	return adp.rawInfra
}
