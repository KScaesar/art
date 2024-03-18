package Artifex

import (
	"sync"
	"sync/atomic"

	"golang.org/x/exp/constraints"
)

func NewArtist[Subject constraints.Ordered, rMessage, sMessage any](recvMux *MessageMux[Subject, rMessage]) *Artist[Subject, rMessage, sMessage] {
	return &Artist[Subject, rMessage, sMessage]{
		recvMux:       recvMux,
		sessions:      make(map[*Session[Subject, rMessage, sMessage]]bool),
		spawnHandlers: make([]func(sess *Session[Subject, rMessage, sMessage]) error, 0),
		exitHandlers:  make([]func(sess *Session[Subject, rMessage, sMessage]), 0),
	}
}

type Artist[Subject constraints.Ordered, rMessage, sMessage any] struct {
	mu       sync.RWMutex
	isStop   atomic.Bool
	recvMux  *MessageMux[Subject, rMessage]
	sessions map[*Session[Subject, rMessage, sMessage]]bool

	concurrencyQty int                                                      // Option
	spawnHandlers  []func(sess *Session[Subject, rMessage, sMessage]) error // Option
	exitHandlers   []func(sess *Session[Subject, rMessage, sMessage])       // Option
}

func (hub *Artist[Subject, rMessage, sMessage]) Stop() {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if hub.isStop.Load() {
		return
	}
	hub.isStop.Store(true)

	wg := sync.WaitGroup{}
	for sess := range hub.sessions {
		session := sess
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.exit(session)
		}()
	}
	wg.Done()
}

func (hub *Artist[Subject, rMessage, sMessage]) Connect(newAdapter NewAdapterFunc[Subject, rMessage, sMessage]) (*Session[Subject, rMessage, sMessage], error) {
	if hub.isStop.Load() {
		return nil, ErrorWrapWithMessage(ErrClosed, "Artifex hub")
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	adapter, err := newAdapter()
	if err != nil {
		return nil, err
	}

	sess, err := NewSession(hub.recvMux, adapter)
	if err != nil {
		return nil, err
	}

	err = hub.spawn(sess)
	if err != nil {
		hub.exit(sess)
		return nil, err
	}

	go func() {
		defer func() {
			hub.mu.Lock()
			hub.exit(sess)
			hub.mu.Unlock()
		}()
		sess.Listen()
	}()

	return sess, nil
}

func (hub *Artist[Subject, rMessage, sMessage]) spawn(sess *Session[Subject, rMessage, sMessage]) error {
	hub.sessions[sess] = true
	for _, action := range hub.spawnHandlers {
		err := action(sess)
		if err != nil {
			return err
		}
	}
	return nil
}

func (hub *Artist[Subject, rMessage, sMessage]) exit(sess *Session[Subject, rMessage, sMessage]) {
	delete(hub.sessions, sess)
	defer sess.Stop()
	for _, action := range hub.exitHandlers {
		action(sess)
	}
}

func (hub *Artist[Subject, rMessage, sMessage]) DoAction(action func(*Session[Subject, rMessage, sMessage])) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for sess := range hub.sessions {
		action(sess)
	}
}

func (hub *Artist[Subject, rMessage, sMessage]) BroadcastFilter(msg sMessage, filter func(*Session[Subject, rMessage, sMessage]) bool) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	var bucket chan struct{}

	for sess := range hub.sessions {
		session := sess

		if !filter(session) {
			continue
		}

		if hub.concurrencyQty <= 0 {
			go session.Send(msg)
			continue
		}

		bucket = make(chan struct{}, hub.concurrencyQty)
		bucket <- struct{}{}
		go func() {
			defer func() {
				<-bucket
			}()
			session.Send(msg)
		}()
	}
}

func (hub *Artist[Subject, rMessage, sMessage]) Broadcast(msg sMessage) {
	hub.BroadcastFilter(msg, func(*Session[Subject, rMessage, sMessage]) bool {
		return true
	})
}

func (hub *Artist[Subject, rMessage, sMessage]) BroadcastOther(msg sMessage, self *Session[Subject, rMessage, sMessage]) {
	hub.BroadcastFilter(msg, func(other *Session[Subject, rMessage, sMessage]) bool {
		return other != self
	})
}

func (hub *Artist[Subject, rMessage, sMessage]) Len() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.sessions)
}

func (hub *Artist[Subject, rMessage, sMessage]) SetConcurrencyQty(concurrencyQty int) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.concurrencyQty = concurrencyQty
}

func (hub *Artist[Subject, rMessage, sMessage]) SetSpawnHandler(spawnHandler func(sess *Session[Subject, rMessage, sMessage]) error) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.spawnHandlers = append(hub.spawnHandlers, spawnHandler)
}

func (hub *Artist[Subject, rMessage, sMessage]) SetExitHandler(exitHandler func(sess *Session[Subject, rMessage, sMessage])) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.exitHandlers = append(hub.exitHandlers, exitHandler)
}
