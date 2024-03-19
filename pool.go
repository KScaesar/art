package Artifex

import (
	"sync"

	"github.com/gookit/goutil/maputil"
)

var (
	routeParamPool = newPool(func() *RouteParam {
		return &RouteParam{make(maputil.Data)}
	})
)

type pool[T any] struct {
	syncPool sync.Pool
}

func newPool[T any](newFn func() T) *pool[T] {
	pool := &pool[T]{
		syncPool: sync.Pool{
			New: func() interface{} { return newFn() },
		},
	}
	return pool
}

func (p *pool[T]) Get() T {
	return p.syncPool.Get().(T)
}

func (p *pool[T]) Put(value T) {
	p.syncPool.Put(value)
}
