package Artifex

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func NewShutdown() Shutdown {
	notify := make(chan os.Signal, 2)
	signal.Notify(notify, syscall.SIGINT, syscall.SIGTERM)
	return Shutdown{
		done:      make(chan struct{}),
		sysSignal: notify,
		Logger:    DefaultLogger(),
	}
}

type Shutdown struct {
	done      chan struct{}
	cancel    context.CancelCauseFunc
	sysSignal chan os.Signal
	Logger    Logger

	stopQty     int
	names       []string
	stopActions []func()
}

func (s *Shutdown) SetStopAction(name string, action func()) *Shutdown {
	s.stopQty++
	s.names = append(s.names, name)
	s.stopActions = append(s.stopActions, action)
	return s
}

func (s *Shutdown) NotifyStop(cause error) {
	s.cancel(cause)
}

func (s *Shutdown) WaitAfter(waitSecond int) {
	s.WaitAfterWithContext(context.Background(), waitSecond)
}

func (s *Shutdown) WaitAfterWithContext(ctx context.Context, waitSecond int) {
	timer := time.NewTimer(time.Duration(waitSecond) * time.Second)
	defer timer.Stop()

	ctx, cancel := context.WithCancelCause(ctx)
	s.cancel = cancel
	go s.listen(ctx)

	select {
	case <-timer.C:
	case <-s.done:
	}
}

func (s *Shutdown) Wait() {
	s.WaitWithContext(context.Background())
}

func (s *Shutdown) WaitWithContext(ctx context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	s.cancel = cancel
	go s.listen(ctx)
	<-s.done
}

func (s *Shutdown) listen(ctx context.Context) {
	defer close(s.done)

	select {
	case sig := <-s.sysSignal:
		s.Logger.Info("receive signal: %v", sig)

	case <-ctx.Done():
		err := context.Cause(ctx)
		if errors.Is(err, context.Canceled) {
			s.Logger.Info("receive go context")
		} else {
			s.Logger.Error("receive go context: %v", err)
		}
	}

	s.Logger.Info("shutdown total service qty=%v", s.stopQty)
	wg := sync.WaitGroup{}
	for i := 0; i < s.stopQty; i++ {
		number := i + 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Logger.Info("number %v service %v shutdown start", number, s.names[number-1])
			s.stopActions[number-1]()
			s.Logger.Info("number %v service %v shutdown finish", number, s.names[number-1])
		}()
	}
	wg.Wait()
	s.Logger.Info("shutdown finish")
}
