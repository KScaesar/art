package Artifex

import (
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
)

func ReliableTask(task func() error, allowStop func() bool, retryMaxSecond int, fixup func() error) error {
	if task == nil || allowStop == nil {
		panic("ReliableTask: task or allowStop is nil")
	}

	param := backoff.NewExponentialBackOff()
	param.InitialInterval = time.Second
	param.RandomizationFactor = 0.5
	param.Multiplier = 1.5
	param.MaxInterval = 1 * time.Minute
	param.MaxElapsedTime = time.Duration(retryMaxSecond) * time.Second

Task:
	err := task()
	if err == nil {
		return nil
	}

	if fixup == nil {
		return backoff.Retry(func() error {
			if allowStop() {
				return backoff.Permanent(errors.New("task has actively been stopped"))
			}
			return task()
		}, param)
	}

	err = backoff.Retry(func() error {
		if allowStop() {
			return backoff.Permanent(errors.New("task has actively been stopped"))
		}
		return fixup()
	}, param)
	if err == nil {
		goto Task
	}
	return err
}
