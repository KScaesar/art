package Artifex

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

type PermanentTasker struct {
	RunUntilNormalStop      func() error
	SelfRepair              func() error
	BackoffMaxElapsedMinute int
}

func (tasker PermanentTasker) Start() {
	backoffCfg := backoff.NewExponentialBackOff()
	backoffCfg.InitialInterval = 500 * time.Millisecond
	backoffCfg.Multiplier = 1.5
	backoffCfg.RandomizationFactor = 0.5

	if tasker.BackoffMaxElapsedMinute <= 0 {
		tasker.BackoffMaxElapsedMinute = 30
	}
	backoffCfg.MaxElapsedTime = time.Duration(tasker.BackoffMaxElapsedMinute) * time.Minute

Loop:
	var err error
	err = tasker.RunUntilNormalStop()
	if err == nil {
		return
	}

	for {
		err = backoff.Retry(tasker.SelfRepair, backoffCfg)
		if err == nil {
			goto Loop
		}
	}
}
