package Artifex

import (
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
)

func ReliableTask(task func() error, allowStop func() bool, fixup func() error, maxElapsedMinute int) {
	if task == nil || allowStop == nil {
		panic("ReliableTask: task or allowStop is nil")
	}

	param := backoff.NewExponentialBackOff()
	param.InitialInterval = time.Second
	param.RandomizationFactor = 0.5
	param.Multiplier = 1.5
	param.MaxInterval = 2 * time.Minute
	param.MaxElapsedTime = time.Duration(maxElapsedMinute) * time.Minute

Task:
	err := task()
	if err == nil {
		return
	}

	if fixup == nil {
		backoff.Retry(func() error {
			if allowStop() {
				return backoff.Permanent(errors.New("recv stop command"))
			}
			return task()
		}, param)
		return
	}

	err = backoff.Retry(func() error {
		if allowStop() {
			return backoff.Permanent(errors.New("recv stop command"))
		}
		return fixup()
	}, param)
	if err == nil {
		goto Task
	}
}
