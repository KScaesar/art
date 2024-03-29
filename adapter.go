package Artifex

import (
	"sync"
	"sync/atomic"

	"github.com/gookit/goutil/maputil"
)

func NewPubSub[rMessage, sMessage any](opt *AdapterOption[rMessage, sMessage]) (pubsub *Adapter[rMessage, sMessage]) {
	return &Adapter[rMessage, sMessage]{
		handleRecv:  opt.handleRecv,
		adapterRecv: opt.adapterRecv,
		adapterSend: opt.adapterSend,
		adapterStop: opt.adapterStop,

		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,

		pingpong: opt.pingpong,

		identifier: opt.identifier,
		appData:    make(maputil.Data),

		result:        make(chan error, 2),
		notifyStopAll: make([]chan error, 0),
	}
}

func NewPublisher[sMessage any](opt *AdapterOption[struct{}, sMessage]) (publisher *Adapter[struct{}, sMessage]) {
	return &Adapter[struct{}, sMessage]{
		handleRecv:  nil,
		adapterRecv: nil,
		adapterSend: opt.adapterSend,
		adapterStop: opt.adapterStop,

		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,

		pingpong: opt.pingpong,

		identifier: opt.identifier,
		appData:    make(maputil.Data),

		result:        make(chan error, 2),
		notifyStopAll: make([]chan error, 0),
	}
}

func NewSubscriber[rMessage any](opt *AdapterOption[rMessage, struct{}]) (subscriber *Adapter[rMessage, struct{}]) {
	return &Adapter[rMessage, struct{}]{
		handleRecv:  opt.handleRecv,
		adapterRecv: opt.adapterRecv,
		adapterSend: nil,
		adapterStop: opt.adapterStop,

		fixupMaxRetrySecond: opt.fixupMaxRetrySecond,
		adapterFixup:        opt.adapterFixup,

		pingpong: opt.pingpong,

		identifier: opt.identifier,
		appData:    make(maputil.Data),

		result:        make(chan error, 2),
		notifyStopAll: make([]chan error, 0),
	}
}

type Adapter[rMessage, sMessage any] struct {
	handleRecv  HandleFunc[rMessage]
	adapterRecv func(*Adapter[rMessage, sMessage]) (*rMessage, error)
	adapterSend func(*Adapter[rMessage, sMessage], *sMessage) error
	adapterStop func(*Adapter[rMessage, sMessage], *sMessage) error

	fixupMaxRetrySecond int
	adapterFixup        func() error

	lifecycle Lifecycle
	pingpong  func(isStop func() bool) error

	identifier string
	appData    maputil.Data
	mutex      sync.RWMutex

	isStop   atomic.Bool
	onceInit sync.Once

	result        chan error
	notifyStopAll []chan error
}

func (adp *Adapter[rMessage, sMessage]) init() error {
	var err error
	adp.onceInit.Do(func() {
		err = adp.lifecycle.Execute()
		if err != nil {
			return
		}

		go adp.monitor()

		go adp.runPingPong()
	})
	return err
}

func (adp *Adapter[rMessage, sMessage]) Identifier() string {
	return adp.identifier
}

func (adp *Adapter[rMessage, sMessage]) Query(query func(identifier string, appData maputil.Data)) {
	adp.mutex.RLock()
	defer adp.mutex.RUnlock()
	query(adp.identifier, adp.appData)
}

func (adp *Adapter[rMessage, sMessage]) Update(update func(identifier *string, appData maputil.Data)) {
	adp.mutex.Lock()
	defer adp.mutex.Unlock()
	update(&adp.identifier, adp.appData)
}

func (adp *Adapter[rMessage, sMessage]) AddSpawnHandler(spawnHandlers ...func() error) *Adapter[rMessage, sMessage] {
	adp.mutex.Lock()
	defer adp.mutex.Unlock()
	adp.lifecycle.AddSpawnHandler(spawnHandlers...)
	return adp
}

func (adp *Adapter[rMessage, sMessage]) AddExitHandler(exitHandlers ...func() error) *Adapter[rMessage, sMessage] {
	adp.mutex.Lock()
	defer adp.mutex.Unlock()
	adp.lifecycle.AddExitHandler(exitHandlers...)
	return adp
}

func (adp *Adapter[rMessage, sMessage]) Listen() (err error) {
	err = adp.init()
	if err != nil {
		return err
	}

	if adp.adapterFixup == nil {
		err = adp.listen()
		adp.result <- err
		return
	}

	err = ReliableTask(adp.listen, adp.IsStop, adp.fixupMaxRetrySecond, adp.adapterFixup)
	adp.result <- err
	return err
}

func (adp *Adapter[rMessage, sMessage]) listen() error {
	for !adp.isStop.Load() {
		message, err := adp.adapterRecv(adp)

		if adp.isStop.Load() {
			return nil
		}

		if err != nil {
			return err
		}

		adp.handleRecv(message, nil)
	}
	return nil
}

func (adp *Adapter[rMessage, sMessage]) Send(message *sMessage) error {
	err := adp.init()
	if err != nil {
		return err
	}

	return adp.adapterSend(adp, message)
}

func (adp *Adapter[rMessage, sMessage]) IsStop() bool {
	return adp.isStop.Load()
}

func (adp *Adapter[rMessage, sMessage]) Stop() error {
	var empty *sMessage
	return adp.StopWithMessage(empty)
}

func (adp *Adapter[rMessage, sMessage]) StopWithMessage(message *sMessage) error {
	err := adp.init()
	if err != nil {
		return err
	}

	if adp.isStop.Load() {
		return nil
	}

	adp.isStop.Store(true)
	adp.lifecycle.NotifyExit()

	err = adp.adapterStop(adp, message)
	if err != nil {
		return err
	}
	return nil
}

// NotifyStop returns a channel for receiving the result of the Adapter.Listen.
// If the Adapter has been already Stopped,
// it returns a channel containing an error message indicating the Adapter is closed.
//
// Once notified, the channel will be closed immediately.
func (adp *Adapter[rMessage, sMessage]) NotifyStop() <-chan error {
	adp.mutex.Lock()
	defer adp.mutex.Unlock()

	ch := make(chan error, 1)
	if adp.isStop.Load() {
		ch <- ErrorWrapWithMessage(ErrClosed, "Artifex adapter")
		close(ch)
		return ch
	}

	adp.notifyStopAll = append(adp.notifyStopAll, ch)
	return ch
}

func (adp *Adapter[rMessage, sMessage]) monitor() {
	err := <-adp.result
	adp.Stop()

	adp.mutex.Lock()
	defer adp.mutex.Unlock()

	for _, notifyStop := range adp.notifyStopAll {
		notifyStop <- err
		close(notifyStop)
	}

	adp.notifyStopAll = make([]chan error, 0)
}

func (adp *Adapter[rMessage, sMessage]) runPingPong() {
	if adp.pingpong == nil {
		return
	}

	if adp.adapterFixup == nil {
		adp.result <- adp.pingpong(adp.IsStop)
		return
	}

	adp.result <- ReliableTask(
		func() error {
			return adp.pingpong(adp.IsStop)
		},
		adp.IsStop,
		adp.fixupMaxRetrySecond,
		adp.adapterFixup,
	)
}
