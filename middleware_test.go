//go:build longtime
// +build longtime

package Artifex

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestMW_Retry(t *testing.T) {
	mw := MW[string]{}

	start := time.Now()

	msg := "task fail"
	retryMaxSecond := 20
	task := func(dependency any, message *string, route *RouteParam) error {
		diff := time.Now().Sub(start)
		fmt.Println(*message, diff)
		if diff > 5*time.Second {
			return nil
		}
		return errors.New(msg)
	}
	err := LinkMiddlewares(task, mw.Retry(retryMaxSecond))(nil, &msg, nil)
	if err != nil {
		t.Errorf("unexpected output: %v", err)
	}
}
