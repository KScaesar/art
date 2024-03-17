package Artifex

import (
	"sync"
	"sync/atomic"

	"golang.org/x/exp/constraints"
)

func NewArtist[Subject constraints.Ordered, rMessage, sMessage any](recvMux *MessageMux[Subject, rMessage]) *Artist[Subject, rMessage, sMessage] {
	return &Artist[Subject, rMessage, sMessage]{
		sessions:       make(map[*Session[Subject, rMessage, sMessage]]bool),
		enterHandlers:  make([]func(sess *Session[Subject, rMessage, sMessage]), 0),
		leaveHandlers:  make([]func(sess *Session[Subject, rMessage, sMessage]), 0),
		concurrencyQty: -1,
		recvMux:        recvMux,
	}
}

type Artist[Subject constraints.Ordered, rMessage, sMessage any] struct {
	mu             sync.RWMutex
	isStop         atomic.Bool
	sessions       map[*Session[Subject, rMessage, sMessage]]bool
	enterHandlers  []func(sess *Session[Subject, rMessage, sMessage])
	leaveHandlers  []func(sess *Session[Subject, rMessage, sMessage])
	concurrencyQty int
	recvMux        *MessageMux[Subject, rMessage]
}

func (hub *Artist[Subject, rMessage, sMessage]) Stop() {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.isStop.Store(true)
	wg := sync.WaitGroup{}
	for sess := range hub.sessions {
		session := sess
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.leave(session)
		}()
	}
	wg.Done()
}

func (hub *Artist[Subject, rMessage, sMessage]) Register(adapter Adapter[Subject, rMessage, sMessage]) (*Session[Subject, rMessage, sMessage], error) {
	if hub.isStop.Load() {
		return nil, ErrClosed
	}

	sess, err := NewSession(hub.recvMux, adapter)
	if err != nil {
		return nil, err
	}

	go func() {
		hub.enter(sess)
		defer hub.leave(sess)
		sess.Listen()
	}()

	return sess, nil
}

func (hub *Artist[Subject, rMessage, sMessage]) enter(sess *Session[Subject, rMessage, sMessage]) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.sessions[sess] = true

	for _, action := range hub.enterHandlers {
		action(sess)
	}
}

func (hub *Artist[Subject, rMessage, sMessage]) leave(sess *Session[Subject, rMessage, sMessage]) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	delete(hub.sessions, sess)

	for _, action := range hub.leaveHandlers {
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

func (hub *Artist[Subject, rMessage, sMessage]) SetEnterHandler(enterHandler func(sess *Session[Subject, rMessage, sMessage])) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.enterHandlers = append(hub.enterHandlers, enterHandler)
}

func (hub *Artist[Subject, rMessage, sMessage]) SetLeaveHandler(leaveHandler func(sess *Session[Subject, rMessage, sMessage])) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.leaveHandlers = append(hub.leaveHandlers, leaveHandler)
}
