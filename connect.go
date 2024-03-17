package Artifex

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/exp/constraints"
)

type ConnectParam[Subject constraints.Ordered, rMessage, sMessage any] struct {
	RecvMux                 *MessageMux[Subject, rMessage]                           // Must
	NewAdapter              NewAdapterFunc[Subject, rMessage, sMessage]              // Must
	Permanent               bool                                                     // Option
	EnterHandlers           []func(sess *Session[Subject, rMessage, sMessage]) error // Option
	LeaveHandlers           []func(sess *Session[Subject, rMessage, sMessage])       // Option
	BackoffMaxElapsedMinute int                                                      // Option
}

func Connect[Subject constraints.Ordered, rMessage, sMessage any](param ConnectParam[Subject, rMessage, sMessage]) (*Session[Subject, rMessage, sMessage], error) {
	adapter, err := param.NewAdapter()
	if err != nil {
		return nil, err
	}

	sess, err := NewSession(param.RecvMux, adapter)
	if err != nil {
		return nil, err
	}

	if param.EnterHandlers != nil {
		for _, enter := range param.EnterHandlers {
			err = enter(sess)
			if err == nil {
				continue
			}

			if param.LeaveHandlers != nil {
				for _, leave := range param.LeaveHandlers {
					leave(sess)
				}
			}
			sess.Stop()
			return nil, err
		}
	}

	backoffMaxElapsedMinute := 30
	if param.BackoffMaxElapsedMinute > 0 {
		backoffMaxElapsedMinute = param.BackoffMaxElapsedMinute
	}

	go func() {
		defer func() {
			if param.LeaveHandlers != nil {
				for _, leave := range param.LeaveHandlers {
					leave(sess)
				}
			}
			sess.Stop()
		}()

	Listen:
		err = sess.Listen()
		if err == nil {
			return
		}

		if !param.Permanent {
			return
		}

		for !sess.IsStop() {
			err = Reconnect(sess, param.NewAdapter, backoffMaxElapsedMinute)
			if err == nil {
				goto Listen
			}
		}
	}()

	return sess, nil
}

func Reconnect[Subject constraints.Ordered, rMessage, sMessage any](
	sess *Session[Subject, rMessage, sMessage],
	newAdapter NewAdapterFunc[Subject, rMessage, sMessage],
	backoffMaxElapsedMinute int,
) error {
	cfg := backoff.NewExponentialBackOff()
	cfg.InitialInterval = 500 * time.Millisecond
	cfg.Multiplier = 1.5
	cfg.RandomizationFactor = 0.5
	cfg.MaxElapsedTime = time.Duration(backoffMaxElapsedMinute) * time.Minute

	cnt := 0
	return backoff.Retry(func() error {
		if sess.IsStop() {
			return nil
		}

		adapter, err := newAdapter()

		cnt++
		sess.Logger().Info("%v reconnect %v times", adapter.Identifier, cnt)

		if err != nil {
			return err
		}

		sess.recv = adapter.Recv
		sess.send = adapter.Send
		sess.stop = adapter.Stop
		sess.isListen.Store(false)

		sess.Logger().Info("%v reconnect success", adapter.Identifier)
		return nil

	}, cfg)
}
