package Artifex

import (
	"sync"
	"sync/atomic"
)

func NewHub[T any](stopObj func(T)) *Hub[T] {
	return &Hub[T]{
		stopObj:  stopObj,
		waitStop: make(chan struct{}),
	}
}

type Hub[T any] struct {
	collections    sync.Map
	concurrencyQty atomic.Int32
	bucket         chan struct{}

	stopObj  func(T)
	isStop   atomic.Bool
	stopOnce sync.Once
	waitStop chan struct{}
}

func (hub *Hub[T]) Join(key string, obj T) error {
	if hub.isStop.Load() {
		return ErrorWrapWithMessage(ErrClosed, "Artifex hub")
	}

	loaded := true
	for loaded {
		_, loaded = hub.collections.LoadOrStore(key, obj)
		if loaded {
			hub.remove(key)
		}
	}
	return nil
}

func (hub *Hub[T]) UpdateByOldKey(oldKey string, update func(T) (freshKey string, err error)) error {
	obj, ok := hub.collections.Load(oldKey)
	if !ok {
		return ErrorWrapWithMessage(ErrNotFound, "key=%v not exist in hub", oldKey)
	}

	freshKey, err := update(obj.(T))
	if err != nil {
		return err
	}

	hub.collections.Delete(oldKey)
	_, ok = hub.collections.Load(freshKey)
	if !ok {
		hub.collections.Store(freshKey, obj)
	}
	return nil
}

func (hub *Hub[T]) FindByKey(key string) (obj T, found bool) {
	value, ok := hub.collections.Load(key)
	return value.(T), ok
}

// If filter returns true, find target
func (hub *Hub[T]) FindOne(filter func(T) bool) (obj T, found bool) {
	var target T
	hub.DoSync(func(obj T) (stop bool) {
		if filter(obj) {
			target = obj
			found = true
			return true
		}
		return false
	})
	return target, found
}

// If filter returns true, find target
func (hub *Hub[T]) FindMulti(filter func(T) bool) (all []T, found bool) {
	all = make([]T, 0)
	hub.DoSync(func(obj T) bool {
		if filter(obj) {
			all = append(all, obj)
		}
		return false
	})
	return all, len(all) > 0
}

func (hub *Hub[T]) FindMultiByKey(keys []string) (all []T, found bool) {
	all = make([]T, 0)
	for i := 0; i < len(keys); i++ {
		obj, ok := hub.FindByKey(keys[i])
		if ok {
			all = append(all, obj)
		}
	}
	return all, len(all) > 0
}

func (hub *Hub[T]) RemoveByKey(key string) {
	hub.remove(key)
}

// If filter returns true, remove target
func (hub *Hub[T]) RemoveOne(filter func(T) bool) {
	hub.collections.Range(func(key, value any) bool {
		obj := value.(T)
		if filter(obj) {
			hub.remove(key.(string))
			return false
		}
		return true
	})
}

// If filter returns true, remove target
func (hub *Hub[T]) RemoveMulti(filter func(T) bool) {
	hub.collections.Range(func(key, value any) bool {
		obj := value.(T)
		if filter(obj) {
			hub.remove(key.(string))
		}
		return true
	})
}

func (hub *Hub[T]) RemoveMultiByKey(keys []string) {
	for i := 0; i < len(keys); i++ {
		hub.remove(keys[i])
	}
}

func (hub *Hub[T]) remove(key string) {
	obj, loaded := hub.collections.LoadAndDelete(key)
	if !loaded {
		return
	}
	hub.stopObj(obj.(T))
}

func (hub *Hub[T]) StopAll() {
	if hub.isStop.Load() {
		return
	}
	hub.isStop.Store(true)

	hub.collections.Range(func(key, value any) bool {
		hub.remove(key.(string))
		return true
	})

	hub.stopOnce.Do(func() {
		close(hub.waitStop)
	})
}

func (hub *Hub[T]) WaitStopAll() chan struct{} {
	return hub.waitStop
}

// DoSync
// If action returns stop=true, stops the iteration.
func (hub *Hub[T]) DoSync(action func(T) (stop bool)) {
	if hub.isStop.Load() {
		return
	}

	hub.collections.Range(func(_, value any) bool {
		obj := value.(T)
		stop := action(obj)
		if stop {
			return false
		}
		return true
	})
}

func (hub *Hub[T]) DoAsync(action func(T)) {
	if hub.isStop.Load() {
		return
	}

	hub.collections.Range(func(_, value any) bool {
		obj := value.(T)

		if hub.concurrencyQty.Load() <= 0 {
			go action(obj)
			return true
		}

		hub.bucket <- struct{}{}
		go func() {
			defer func() {
				<-hub.bucket
			}()
			action(obj)
		}()

		return true
	})
}

// Count
// If filter returns true, increase count
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
	filter := func(t T) bool { return true }
	return hub.Count(filter)
}

// SetConcurrencyQty
// concurrencyQty controls how many tasks can run simultaneously,
// preventing resource usage or avoid frequent context switches.
func (hub *Hub[T]) SetConcurrencyQty(concurrencyQty int) {
	if hub.isStop.Load() {
		return
	}
	hub.concurrencyQty.Store(int32(concurrencyQty))
	hub.bucket = make(chan struct{}, hub.concurrencyQty.Load())
}
