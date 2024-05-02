package art

import (
	"sync"
	"sync/atomic"
)

func NewHub() *Hub {
	return &Hub{
		waitStop: make(chan struct{}),
	}
}

type Hub struct {
	collections    sync.Map
	concurrencyQty atomic.Int32
	bucket         chan struct{}

	isStopped atomic.Bool
	waitStop  chan struct{}
}

func (hub *Hub) Join(key string, adp IAdapter) error {
	if hub.isStopped.Load() {
		return ErrorWrapWithMessage(ErrClosed, "art Hub Join")
	}

	loaded := true
	for loaded {
		_, loaded = hub.collections.LoadOrStore(key, adp)
		if loaded {
			hub.remove(key)
		}
	}
	return nil
}

func (hub *Hub) Shutdown() {
	if hub.isStopped.Swap(true) {
		return
	}

	hub.collections.Range(func(key, value any) bool {
		hub.remove(key.(string))
		return true
	})

	close(hub.waitStop)
}

func (hub *Hub) IsShutdown() bool {
	return hub.isStopped.Load()
}

func (hub *Hub) WaitShutdown() chan struct{} {
	return hub.waitStop
}

func (hub *Hub) UpdateByOldKey(oldKey string, update func(IAdapter) (freshKey string)) error {
	value, ok := hub.collections.LoadAndDelete(oldKey)
	if !ok {
		return ErrorWrapWithMessage(ErrNotFound, "key=%v not exist in hub", oldKey)
	}

	adp := value.(IAdapter)
	freshKey := update(adp)
	return hub.Join(freshKey, adp)
}

func (hub *Hub) FindByKey(key string) (adp IAdapter, found bool) {
	value, ok := hub.collections.Load(key)
	if ok {
		return value.(IAdapter), true
	}
	return adp, false
}

// If filter returns true, find target
func (hub *Hub) FindOne(filter func(IAdapter) bool) (adp IAdapter, found bool) {
	var target IAdapter
	hub.DoSync(func(adp IAdapter) (stop bool) {
		if filter(adp) {
			target = adp
			found = true
			return true
		}
		return false
	})
	return target, found
}

// If filter returns true, find target
func (hub *Hub) FindMulti(filter func(IAdapter) bool) (all []IAdapter, found bool) {
	all = make([]IAdapter, 0)
	hub.DoSync(func(adp IAdapter) bool {
		if filter(adp) {
			all = append(all, adp)
		}
		return false
	})
	return all, len(all) > 0
}

func (hub *Hub) FindMultiByKey(keys []string) (all []IAdapter, found bool) {
	all = make([]IAdapter, 0)
	for i := 0; i < len(keys); i++ {
		adp, ok := hub.FindByKey(keys[i])
		if ok {
			all = append(all, adp)
		}
	}
	return all, len(all) > 0
}

func (hub *Hub) RemoveByKey(key string) {
	hub.remove(key)
}

// If filter returns true, remove target
func (hub *Hub) RemoveOne(filter func(IAdapter) bool) {
	hub.collections.Range(func(key, value any) bool {
		adp := value.(IAdapter)
		if filter(adp) {
			hub.remove(key.(string))
			return false
		}
		return true
	})
}

// If filter returns true, remove target
func (hub *Hub) RemoveMulti(filter func(IAdapter) bool) {
	hub.collections.Range(func(key, value any) bool {
		adp := value.(IAdapter)
		if filter(adp) {
			hub.remove(key.(string))
		}
		return true
	})
}

func (hub *Hub) RemoveMultiByKey(keys []string) {
	for i := 0; i < len(keys); i++ {
		hub.remove(keys[i])
	}
}

func (hub *Hub) remove(key string) {
	adp, loaded := hub.collections.LoadAndDelete(key)
	if !loaded {
		return
	}
	adapter := adp.(IAdapter)
	if !adapter.IsStopped() {
		adapter.Stop()
	}
}

// DoSync
// If action returns stop=true, stops the iteration.
func (hub *Hub) DoSync(action func(IAdapter) (stop bool)) {
	hub.collections.Range(func(_, value any) bool {
		adp := value.(IAdapter)
		stop := action(adp)
		if stop {
			return false
		}
		return true
	})
}

func (hub *Hub) DoAsync(action func(IAdapter)) {
	hub.collections.Range(func(_, value any) bool {
		adp := value.(IAdapter)

		if hub.concurrencyQty.Load() <= 0 {
			go action(adp)
			return true
		}

		hub.bucket <- struct{}{}
		go func() {
			defer func() {
				<-hub.bucket
			}()
			action(adp)
		}()

		return true
	})
}

// Count
// If filter returns true, increase count
func (hub *Hub) Count(filter func(IAdapter) bool) int {
	cnt := 0
	hub.DoSync(func(adp IAdapter) bool {
		if filter(adp) {
			cnt++
		}
		return false
	})
	return cnt
}

func (hub *Hub) Total() int {
	filter := func(t IAdapter) bool { return true }
	return hub.Count(filter)
}

// SetConcurrencyQty
// concurrencyQty controls how many tasks can run simultaneously,
// preventing resource usage or avoid frequent context switches.
func (hub *Hub) SetConcurrencyQty(concurrencyQty int) {
	hub.concurrencyQty.Store(int32(concurrencyQty))
	hub.bucket = make(chan struct{}, hub.concurrencyQty.Load())
}
