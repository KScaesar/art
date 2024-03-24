package Artifex

import (
	"sync"
	"sync/atomic"
)

func NewAdapterHub[Adapter any](stop func(*Adapter) error) *AdapterHub[Adapter] {
	return &AdapterHub[Adapter]{
		adapters:    make(map[string]*Adapter),
		stopAdapter: stop,
	}
}

// AdapterHub can manage multiple adapters.
//
//	concurrencyQty controls how many tasks can run simultaneously,
//	preventing resource usage or avoid frequent context switches.
type AdapterHub[Adapter any] struct {
	adapters       map[string]*Adapter // Identifier:Adapter
	stopAdapter    func(*Adapter) error
	concurrencyQty int
	mu             sync.RWMutex
	isStop         atomic.Bool
}

func (hub *AdapterHub[Adapter]) JoinAdapter(name string, adapter *Adapter) error {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex hub")
	}
	_, found := hub.adapters[name]
	if found {
		return ErrorWrapWithMessage(ErrInvalidParameter, "duplicated adapter=%v", name)
	}
	hub.adapters[name] = adapter
	return nil
}

func (hub *AdapterHub[Adapter]) RemoveAdapter(name string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	adapter, found := hub.adapters[name]
	if !found {
		return
	}
	delete(hub.adapters, name)
	hub.stopAdapter(adapter)
}

func (hub *AdapterHub[Adapter]) UpdateAdapterName(oldName string, updateName func(*Adapter) (freshName string)) error {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if hub.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex hub")
	}

	adapter, found := hub.adapters[oldName]
	if !found {
		return ErrorWrapWithMessage(ErrNotFound, "adapter=%v not exist in hub", oldName)
	}
	freshName := updateName(adapter)
	hub.adapters[freshName] = adapter
	delete(hub.adapters, oldName)
	return nil
}

func (hub *AdapterHub[Adapter]) Stop() {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if hub.isStop.Load() {
		return
	}
	hub.isStop.Store(true)

	for name, adapter := range hub.adapters {
		delete(hub.adapters, name)
		hub.stopAdapter(adapter)
	}
}

func (hub *AdapterHub[Adapter]) DoSync(action func(*Adapter) (stop bool)) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for _, adapter := range hub.adapters {
		stop := action(adapter)
		if stop {
			break
		}
	}
}

func (hub *AdapterHub[Adapter]) DoAsync(action func(*Adapter)) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	var bucket chan struct{}
	for _, adp := range hub.adapters {
		adapter := adp
		if hub.concurrencyQty <= 0 {
			go action(adapter)
			continue
		}

		bucket = make(chan struct{}, hub.concurrencyQty)
		bucket <- struct{}{}
		go func() {
			defer func() {
				<-bucket
			}()
			action(adapter)
		}()
	}
}

func (hub *AdapterHub[Adapter]) FindAdapter(filter func(*Adapter) bool) (adapter *Adapter, found bool) {
	var target *Adapter
	hub.DoSync(func(adapter *Adapter) (stop bool) {
		if filter(adapter) {
			target = adapter
			return true
		}
		return false
	})
	return target, target != nil
}

func (hub *AdapterHub[Adapter]) FindAdapterByName(name string) (adapter *Adapter, found bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	adapter, found = hub.adapters[name]
	return
}

func (hub *AdapterHub[Adapter]) FindAdapters(filter func(*Adapter) bool) (adapters []*Adapter, found bool) {
	adapters = make([]*Adapter, 0)
	hub.DoSync(func(adapter *Adapter) bool {
		if filter(adapter) {
			adapters = append(adapters, adapter)
		}
		return false
	})
	return adapters, len(adapters) > 0
}

func (hub *AdapterHub[Adapter]) FindAdaptersByName(names []string) (adapters []*Adapter, found bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	adapters = make([]*Adapter, 0)
	for i := 0; i < len(names); i++ {
		adapter, ok := hub.adapters[names[i]]
		if ok {
			adapters = append(adapters, adapter)
		}
	}
	return adapters, len(adapters) > 0
}

func (hub *AdapterHub[Adapter]) Count(filter func(*Adapter) bool) int {
	cnt := 0
	hub.DoSync(func(adapter *Adapter) bool {
		if filter(adapter) {
			cnt++
		}
		return false
	})
	return cnt
}

func (hub *AdapterHub[Adapter]) Total() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.adapters)
}

func (hub *AdapterHub[Adapter]) SetConcurrencyQty(concurrencyQty int) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.concurrencyQty = concurrencyQty
}
