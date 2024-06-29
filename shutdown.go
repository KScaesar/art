package art

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func NewShutdownWithoutTimeout() *Shutdown {
	return NewShutdown(0)
}

func NewShutdown(waitSeconds int) *Shutdown {
	osSig := make(chan os.Signal, 2)
	signal.Notify(osSig, syscall.SIGINT, syscall.SIGTERM)
	return &Shutdown{
		waitSeconds: waitSeconds,
		done:        make(chan struct{}),
		osSig:       osSig,
		Logger:      DefaultLogger(),
		names:       make([]string, 0, 4),
		stopActions: make([]func() error, 0, 4),
	}
}

type Shutdown struct {
	waitSeconds int
	done        chan struct{}
	notify      context.CancelCauseFunc
	osSig       chan os.Signal
	Logger      Logger
	mu          sync.Mutex

	stopQty     int
	names       []string
	stopActions []func() error
}

func (s *Shutdown) StopService(name string, action func() error) *Shutdown {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		return s
	default:
	}

	if s.waitSeconds > 0 {
		action = func() error {
			result := make(chan error, 1)
			timeout := time.NewTimer(time.Duration(s.waitSeconds) * time.Second)

			go func() {
				result <- action()
			}()

			select {
			case <-timeout.C:
				return errors.New("timeout")
			case err := <-result:
				return err
			}
		}
	}

	s.stopQty++
	s.names = append(s.names, name)
	s.stopActions = append(s.stopActions, action)
	return s
}

func (s *Shutdown) Notify(cause error) {
	select {
	case <-s.done:
		return
	default:
		s.notify(cause)
	}
}

func (s *Shutdown) WaitChannel() <-chan struct{} {
	return s.done
}

func (s *Shutdown) Serve(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		return
	default:
	}

	defer close(s.done)

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, s.notify = context.WithCancelCause(ctx)

	select {
	case sig := <-s.osSig:
		s.Logger.Info("recv os signal: %v", sig)

	case <-ctx.Done():
		err := context.Cause(ctx)
		if errors.Is(err, context.Canceled) {
			s.Logger.Info("recv go context")
		} else {
			s.Logger.Error("recv go context: %v", err)
		}
	}

	s.Logger.Info("shutdown total service qty=%v", s.stopQty)
	wg := sync.WaitGroup{}

	for i := 0; i < s.stopQty; i++ {
		number := i + 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Logger.Info("number %v service %q shutdown start", number, s.names[number-1])
			err := s.stopActions[number-1]()
			if err != nil {
				s.Logger.Error("number %v service %q shutdown fail: %v", number, s.names[number-1], err)
				return
			}
			s.Logger.Info("number %v service %q shutdown finish", number, s.names[number-1])
		}()
	}
	wg.Wait()
	s.Logger.Info("shutdown finish")
}
