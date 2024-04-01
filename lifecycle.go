package Artifex

import (
	"sync"
)

// Lifecycle define a management mechanism when init obj and terminate obj.
type Lifecycle struct {
	initMutex    sync.Mutex
	initHandlers []func(adapter IAdapter) error

	terminateMutex    sync.Mutex
	terminateHandlers []func(adapter IAdapter)
	wg                sync.WaitGroup
}

func (life *Lifecycle) AddInitialize(inits ...func(adp IAdapter) error) *Lifecycle {
	life.initMutex.Lock()
	defer life.initMutex.Unlock()
	life.initHandlers = append(life.initHandlers, inits...)
	return life
}

func (life *Lifecycle) AddTerminate(terminates ...func(adp IAdapter)) *Lifecycle {
	life.terminateMutex.Lock()
	defer life.terminateMutex.Unlock()
	life.terminateHandlers = append(life.terminateHandlers, terminates...)
	return life
}

func (life *Lifecycle) initialize(adp IAdapter) error {
	life.initMutex.Lock()
	defer life.initMutex.Unlock()

	for _, init := range life.initHandlers {
		err := init(adp)
		if err != nil {
			life.terminate(adp)
			return err
		}
	}
	return nil
}

func (life *Lifecycle) terminate(adp IAdapter) {
	life.terminateMutex.Lock()
	defer life.terminateMutex.Unlock()

	for _, terminate := range life.terminateHandlers {
		terminate(adp)
	}
	return
}

func (life *Lifecycle) asyncTerminate(adp IAdapter) {
	life.terminateMutex.Lock()
	defer life.terminateMutex.Unlock()

	for _, h := range life.terminateHandlers {
		terminate := h
		life.wg.Add(1)
		go func() {
			defer life.wg.Done()
			terminate(adp)
		}()
	}
	return
}

func (life *Lifecycle) wait() {
	life.wg.Wait()
}

func (life *Lifecycle) InitQty() int {
	life.initMutex.Lock()
	defer life.initMutex.Unlock()
	return len(life.initHandlers)
}

func (life *Lifecycle) TerminateQty() int {
	life.terminateMutex.Lock()
	defer life.terminateMutex.Unlock()
	return len(life.terminateHandlers)
}
