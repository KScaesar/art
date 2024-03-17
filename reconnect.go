package Artifex

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/exp/constraints"
)

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
		sess.isStop.Store(false)

		sess.Logger().Info("%v reconnect success", adapter.Identifier)
		return nil

	}, cfg)
}
