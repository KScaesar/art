//go:build longtime
// +build longtime

package art

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestUse_Retry(t *testing.T) {
	start := time.Now()

	msg := Message{
		Subject: "Retry",
	}
	retryMaxSecond := 20
	task := func(message *Message, dependency any) error {
		diff := time.Now().Sub(start)
		fmt.Println(message.Subject, diff)
		if diff > 5*time.Second {
			return nil
		}
		return errors.New("timeout")
	}
	err := Link(task, UseRetry(retryMaxSecond))(&msg, nil)
	if err != nil {
		t.Errorf("unexpected output: %v", err)
	}
}
