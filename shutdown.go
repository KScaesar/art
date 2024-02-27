package Artifex

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Shutdown struct {
	done   chan struct{}
	cancel context.CancelCauseFunc
}

func (s Shutdown) Stop(cause error) Shutdown {
	s.cancel(cause)
	return s
}

func (s Shutdown) Wait(waitSecond int) {
	timer := time.NewTimer(time.Duration(waitSecond) * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-s.done:
	}
}

func SetupShutdown(ctx1 context.Context, stopActions ...func() (serviceName string)) Shutdown {
	notify := make(chan os.Signal, 2)
	signal.Notify(notify, syscall.SIGINT, syscall.SIGTERM)

	if ctx1 == nil {
		ctx1 = context.Background()
	}
	ctx2, cancel := context.WithCancelCause(ctx1)

	done := make(chan struct{})
	logger := DefaultLogger().WithMessageId(GenerateRandomCode(4))

	go func() {
		defer close(done)

		select {
		case sig := <-notify:
			logger.Info("receive signal: %v", sig)

		case <-ctx2.Done():
			err := context.Cause(ctx2)
			logger.Error("receive context channel: %v", err)
		}

		total := len(stopActions)
		logger.Info("total service count=%v, shutdown start", total)
		wg := sync.WaitGroup{}
		for i, stop := range stopActions {
			number := i + 1
			stop := stop
			wg.Add(1)
			go func() {
				defer wg.Done()
				name := stop()
				logger.Info("number %v service %v shutdown finish", number, name)
			}()
		}
		wg.Wait()
		logger.Info("shutdown finish")
	}()

	return Shutdown{
		done:   done,
		cancel: cancel,
	}
}
