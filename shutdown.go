package Artifex

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func Shutdown(ctx1 context.Context, stopActions ...func() (serviceName string)) (invokeStop func(cause error, wait bool)) {
	notify := make(chan os.Signal, 2)
	signal.Notify(notify, syscall.SIGINT, syscall.SIGTERM)

	if ctx1 == nil {
		ctx1 = context.Background()
	}
	ctx2, cancel := context.WithCancelCause(ctx1)

	done := make(chan struct{})
	logger := DefaultLogger().WithMessageId(GenerateRandomCode(4))

	go func() {
		select {
		case sig := <-notify:
			logger.Info("receive signal: %v", sig)

		case <-ctx2.Done():
			err := context.Cause(ctx2)
			if err != nil {
				logger.Error("receive context channel: %v", err)
			} else {
				logger.Info("receive context channel")
			}
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
		close(done)
	}()

	invokeStop = func(cause error, wait bool) {
		cancel(cause)
		if wait {
			<-done
		}
	}
	return invokeStop
}
