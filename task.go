package Artifex

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

func ReliableTask(task func() error, allowStop func() bool, fixup func() error, retryMaxMinute int) {
	if task == nil || allowStop == nil {
		panic("ReliableTask: task or allowStop is nil")
	}

	if retryMaxMinute <= 0 {
		retryMaxMinute = 5
	}
	backoffCfg := backoff.NewExponentialBackOff()
	backoffCfg.InitialInterval = time.Second
	backoffCfg.Multiplier = 1.5
	backoffCfg.RandomizationFactor = 0.5
	backoffCfg.MaxElapsedTime = time.Duration(retryMaxMinute) * time.Minute

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
