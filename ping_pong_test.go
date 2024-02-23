package Artifex

import (
	"fmt"
	"testing"
)

func Test_NoTimeout_When_PingWait_gt_PongWait(t *testing.T) {
	err := pingpong(t, 2, 1)
	if err != nil {
		t.Errorf("PingPing fail: %v", err)
		return
	}
	t.Logf("PingPing success")
}

func Test_HasTimeout_When_PingWait_lt_PongWait(t *testing.T) {
	err := pingpong(t, 1, 2)
	if err != nil {
		t.Logf("PingPing fail: %v", err)
		return
	}
	t.Errorf("PingPing success")
}

func pingpong(t *testing.T, pingWaitSecond, pongWaitSecond int) error {
	sendPingCount := 0
	targetCount := 2

	actor1WaitPong := make(chan error, 1)
	actor2WaitPing := make(chan error, 1)

	actr1SendPing := func() error {
		sendPingCount++
		t.Logf("actor1 ping")
		actor2WaitPing <- nil
		return nil
	}
	actor2SendPong := func() error {
		t.Logf("actor2 pong")
		actor1WaitPong <- nil
		return nil
	}

	isStop := func() bool {
		return sendPingCount == targetCount
	}

	notify := make(chan error, 2)

	actor1 := func() {
		err := SendPingWaitPong(actr1SendPing, actor1WaitPong, isStop, pongWaitSecond)
		if err != nil {
			notify <- fmt.Errorf("actor1: %w", err)
		}
		notify <- nil
	}

	actor2 := func() {
		err := WaitPingSendPong(actor2WaitPing, actor2SendPong, isStop, pingWaitSecond)
		if err != nil {
			notify <- fmt.Errorf("actor2: %w", err)
			return
		}
		notify <- nil
	}

	go actor1()
	go actor2()
	return <-notify
}
