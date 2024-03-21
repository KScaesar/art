package Artifex

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

func ReliableTask(task func() error, allowStop func() bool, fixup func() error) {
	if task == nil || allowStop == nil {
		panic("ReliableTask: task or allowStop is nil")
	}

	backoffCfg := backoff.NewExponentialBackOff()
	backoffCfg.InitialInterval = 500 * time.Millisecond
	backoffCfg.Multiplier = 1.5
	backoffCfg.RandomizationFactor = 0.5
	backoffCfg.MaxElapsedTime = time.Duration(30) * time.Minute

Task:
	err := task()
	if err == nil {
		return
	}

	if fixup == nil {
		for !allowStop() {
			err = backoff.Retry(func() error {
				if allowStop() {
					return nil
				}
				return task()
			}, backoffCfg)
			if err == nil {
				return
			}
		}
		return
	}

	for !allowStop() {
		err = backoff.Retry(func() error {
			if allowStop() {
				return nil
			}
			return fixup()
		}, backoffCfg)
		if err == nil {
			goto Task
		}
	}
}
