package art

import (
	"sync"
)

// Lifecycle define a management mechanism when init obj and terminate obj.
type Lifecycle struct {
	initHandlers      []func(adapter IAdapter) error
	terminateHandlers []func(adapter IAdapter)
	wg                sync.WaitGroup
}

func (life *Lifecycle) OnConnect(inits ...func(adp IAdapter) error) *Lifecycle {
	for _, init := range inits {
		if init == nil {
			continue
		}
		life.initHandlers = append(life.initHandlers, init)
	}
	return life
}

func (life *Lifecycle) OnDisconnect(terminates ...func(adp IAdapter)) *Lifecycle {
	for _, terminate := range terminates {
		if terminate == nil {
			continue
		}
		life.terminateHandlers = append(life.terminateHandlers, terminate)
	}
	return life
}

func (life *Lifecycle) initialize(adp IAdapter) error {
	for _, init := range life.initHandlers {
		err := init(adp)
		if err != nil {
			life.syncTerminate(adp)
			return err
		}
	}
	return nil
}

func (life *Lifecycle) syncTerminate(adp IAdapter) {
	n := len(life.terminateHandlers)
	for i := n - 1; i >= 0; i-- {
		terminate := life.terminateHandlers[i]
		terminate(adp)
	}
	return
}

func (life *Lifecycle) asyncTerminate(adp IAdapter) {
	n := len(life.terminateHandlers)
	for i := n - 1; i >= 0; i-- {
		terminate := life.terminateHandlers[i]
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
