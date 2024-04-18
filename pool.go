package Artifex

import (
	"sync"
)

func newPool[T any](newFn func() T) *pool[T] {
	return &pool[T]{
		syncPool: sync.Pool{
			New: func() interface{} { return newFn() },
		},
	}
}

type pool[T any] struct {
	syncPool sync.Pool
}

func (p *pool[T]) Get() T {
	return p.syncPool.Get().(T)
}

func (p *pool[T]) Put(value T) {
	p.syncPool.Put(value)
}
