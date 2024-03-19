package Artifex

import (
	"sync"
	"sync/atomic"

	"golang.org/x/exp/constraints"
)

func NewArtist[Subject constraints.Ordered, rMessage, sMessage any](recvMux *Mux[Subject, rMessage]) *Artist[Subject, rMessage, sMessage] {
	return &Artist[Subject, rMessage, sMessage]{
		recvMux:  recvMux,
		sessions: make(map[*Session[Subject, rMessage, sMessage]]bool),
	}
}

// Artist can manage the lifecycle of multiple Sessions.
//
//   - concurrencyQty controls how many tasks can run simultaneously,
//     preventing resource usage or avoid frequent context switches.
//   - recvMux is a multiplexer used for handle messages.
//   - newAdapter is used to create a new adapter.
type Artist[Subject constraints.Ordered, rMessage, sMessage any] struct {
	mu             sync.RWMutex
	isStop         atomic.Bool
	recvMux        *Mux[Subject, rMessage]
	sessions       map[*Session[Subject, rMessage, sMessage]]bool
	concurrencyQty int // Option
}

func (hub *Artist[Subject, rMessage, sMessage]) Stop() {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if hub.isStop.Load() {
		return
	}
	hub.isStop.Store(true)

	for sess := range hub.sessions {
		session := sess
		session.Stop()
		delete(hub.sessions, session)
	}
}

func (hub *Artist[Subject, rMessage, sMessage]) Connect(newAdapter NewAdapterFunc[Subject, rMessage, sMessage]) (*Session[Subject, rMessage, sMessage], error) {
	if hub.isStop.Load() {
		return nil, ErrorWrapWithMessage(ErrClosed, "Artifex hub")
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()

	sess, err := NewSession(hub.recvMux, newAdapter)
	if err != nil {
		return nil, err
	}

	hub.sessions[sess] = true
	if err != nil {
		delete(hub.sessions, sess)
		return nil, err
	}

	go func() {
		defer func() {
			hub.mu.Lock()
			delete(hub.sessions, sess)
			hub.mu.Unlock()
		}()
		sess.Listen()
	}()

	return sess, nil
}

func (hub *Artist[Subject, rMessage, sMessage]) FindSession(filter func(*Session[Subject, rMessage, sMessage]) bool) (session *Session[Subject, rMessage, sMessage], found bool) {
	var target *Session[Subject, rMessage, sMessage]
	hub.DoSync(func(sess *Session[Subject, rMessage, sMessage]) (stop bool) {
		if filter(sess) {
			target = sess
			return true
		}
		return false
	})
	return target, target != nil
}

func (hub *Artist[Subject, rMessage, sMessage]) FindSessions(filter func(*Session[Subject, rMessage, sMessage]) bool) (sessions []*Session[Subject, rMessage, sMessage], found bool) {
	sessions = make([]*Session[Subject, rMessage, sMessage], 0)
	hub.DoSync(func(sess *Session[Subject, rMessage, sMessage]) bool {
		if filter(sess) {
			sessions = append(sessions, sess)
		}
		return false
	})
	return sessions, len(sessions) > 0
}

func (hub *Artist[Subject, rMessage, sMessage]) DoSync(action func(*Session[Subject, rMessage, sMessage]) (stop bool)) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for sess := range hub.sessions {
		stop := action(sess)
		if stop {
			break
		}
	}
}

func (hub *Artist[Subject, rMessage, sMessage]) DoAsync(action func(*Session[Subject, rMessage, sMessage])) {
	if hub.isStop.Load() {
		return
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	var bucket chan struct{}
	for sess := range hub.sessions {
		session := sess
		if hub.concurrencyQty <= 0 {
			go action(session)
			continue
		}

		bucket = make(chan struct{}, hub.concurrencyQty)
		bucket <- struct{}{}
		go func() {
			defer func() {
				<-bucket
			}()
			action(session)
		}()
	}
}

func (hub *Artist[Subject, rMessage, sMessage]) BroadcastFilter(msg sMessage, filter func(*Session[Subject, rMessage, sMessage]) bool) {
	hub.DoAsync(func(sess *Session[Subject, rMessage, sMessage]) {
		if filter(sess) {
			sess.Send(msg)
		}
	})
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

func (hub *Artist[Subject, rMessage, sMessage]) Count(filter func(*Session[Subject, rMessage, sMessage]) bool) int {
	cnt := 0
	hub.DoSync(func(sess *Session[Subject, rMessage, sMessage]) bool {
		if filter(sess) {
			cnt++
		}
		return false
	})
	return cnt
}

func (hub *Artist[Subject, rMessage, sMessage]) Total() int {
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
