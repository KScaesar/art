package Artifex

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

type PermanentTasker struct {
	Run                     func() error
	ActiveStop              func() bool
	SelfRepair              func() error
	BackoffMaxElapsedMinute int
}

func (tasker PermanentTasker) Start() error {
	backoffCfg := backoff.NewExponentialBackOff()
	backoffCfg.InitialInterval = 500 * time.Millisecond
	backoffCfg.Multiplier = 1.5
	backoffCfg.RandomizationFactor = 0.5

	if tasker.BackoffMaxElapsedMinute <= 0 {
		tasker.BackoffMaxElapsedMinute = 30
	}
	backoffCfg.MaxElapsedTime = time.Duration(tasker.BackoffMaxElapsedMinute) * time.Minute

Loop:
	err := tasker.Run()
	if err == nil {
		return nil
	}

	var Err error
	for !tasker.ActiveStop() {
		Err = backoff.Retry(func() error {
			if tasker.ActiveStop() {
				return nil
			}
			return tasker.SelfRepair()
		}, backoffCfg)

		if Err == nil {
			goto Loop
		}
	}
	return Err
}
