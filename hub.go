package Artifex

import (
	"sync"
	"sync/atomic"
)

func NewHub[T any](stopObj func(T) error) *Hub[T] {
	return &Hub[T]{
		collections: make(map[string]T),
		stopObj:     stopObj,
	}
}

// Hub
//
//	concurrencyQty controls how many tasks can run simultaneously,
//	preventing resource usage or avoid frequent context switches.
type Hub[T any] struct {
	collections    map[string]T // Identifier:T
	concurrencyQty int
	mu             sync.RWMutex

	stopObj func(T) error
	isStop  atomic.Bool
}

func (hub *Hub[T]) Join(key string, obj T) error {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex hub")
	}

	_, found := hub.collections[key]
	if found {
		hub.remove(key)
	}
	hub.collections[key] = obj
	return nil
}

func (hub *Hub[T]) Remove(key string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.remove(key)
}

func (hub *Hub[T]) remove(key string) {
	obj, found := hub.collections[key]
	if !found {
		return
	}
	delete(hub.collections, key)
	hub.stopObj(obj)
}

func (hub *Hub[T]) UpdateKeyAndObj(oldKey string, update func(T) (freshKey string)) error {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex hub")
	}

	obj, found := hub.collections[oldKey]
	if !found {
		return ErrorWrapWithMessage(ErrNotFound, "key=%v not exist in hub", oldKey)
	}
	freshKey := update(obj)
	hub.collections[freshKey] = obj
	delete(hub.collections, oldKey)
	return nil
}

func (hub *Hub[T]) UpdateByKey(key string, update func(T)) error {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if hub.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex hub")
	}

	obj, found := hub.collections[key]
	if !found {
		return ErrorWrapWithMessage(ErrNotFound, "key=%v not exist in hub", key)
	}
	update(obj)
	return nil
}

func (hub *Hub[T]) StopAll() {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.isStop.Load() {
		return
	}

	hub.isStop.Store(true)
	for key, obj := range hub.collections {
		delete(hub.collections, key)
		hub.stopObj(obj)
	}
}

func (hub *Hub[T]) DoSync(action func(T) (stop bool)) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if hub.isStop.Load() {
		return
	}

	for _, obj := range hub.collections {
		stop := action(obj)
		if stop {
			break
		}
	}
}

func (hub *Hub[T]) DoAsync(action func(T)) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if hub.isStop.Load() {
		return
	}

	var bucket chan struct{}
	for _, v := range hub.collections {
		obj := v
		if hub.concurrencyQty <= 0 {
			go action(obj)
			continue
		}

		bucket = make(chan struct{}, hub.concurrencyQty)
		bucket <- struct{}{}
		go func() {
			defer func() {
				<-bucket
			}()
			action(obj)
		}()
	}
}

func (hub *Hub[T]) Find(filter func(T) bool) (obj T, found bool) {
	var target T
	hub.DoSync(func(obj T) (stop bool) {
		if filter(obj) {
			target = obj
			return true
		}
		return false
	})
	return target, target != nil
}

func (hub *Hub[T]) FindByKey(key string) (obj T, found bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	obj, found = hub.collections[key]
	return
}

func (hub *Hub[T]) FindAll(filter func(T) bool) (all []T, found bool) {
	all = make([]T, 0)
	hub.DoSync(func(obj T) bool {
		if filter(obj) {
			all = append(all, obj)
		}
		return false
	})
	return all, len(all) > 0
}

func (hub *Hub[T]) FindAllByKey(keys []string) (all []T, found bool) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	all = make([]T, 0)
	for i := 0; i < len(keys); i++ {
		obj, ok := hub.collections[keys[i]]
		if ok {
			all = append(all, obj)
		}
	}
	return all, len(all) > 0
}

func (hub *Hub[T]) Count(filter func(T) bool) int {
	cnt := 0
	hub.DoSync(func(obj T) bool {
		if filter(obj) {
			cnt++
		}
		return false
	})
	return cnt
}

func (hub *Hub[T]) Total() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.collections)
}

func (hub *Hub[T]) SetConcurrencyQty(concurrencyQty int) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.concurrencyQty = concurrencyQty
}
