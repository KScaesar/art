package Artifex

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/exp/constraints"
)

type PingPong[S constraints.Ordered, rM, sM any] struct {
	Enable             bool
	IsSendPingWaitPong bool
	SendFunc           func() error
	WaitNotify         chan error

	// When SendPingWaitPong sends a ping message and waits for a corresponding pong message.
	// SendPeriod = WaitSecond / 2
	//
	// When WaitPingSendPong waits for a ping message and response a corresponding pong message.
	// SendPeriod = WaitSecond
	WaitSecond int // Must,  when enable Pingpong
}

func (pp PingPong[S, rM, sM]) RunBySession(sess *Session[S, rM, sM]) error {
	return pp.Run(sess.IsStop)
}

func (pp PingPong[S, rM, sM]) Run(allowStop func() bool) error {
	if !pp.Enable {
		return nil
	}
	if pp.IsSendPingWaitPong {
		return SendPingWaitPong(pp.SendFunc, pp.WaitNotify, allowStop, pp.WaitSecond)
	}
	return WaitPingSendPong(pp.WaitNotify, pp.SendFunc, allowStop, pp.WaitSecond)
}

func WaitPingSendPong(waitPing <-chan error, sendPong func() error, isStop func() bool, pingWaitSecond int) error {
	pingWaitTime := time.Duration(pingWaitSecond) * time.Second

	timer := time.NewTimer(pingWaitTime)
	defer timer.Stop()

	for !isStop() {
		select {
		case <-timer.C:
			return errors.New("wait ping timeout")

		case err := <-waitPing:
			if err != nil {
				return fmt.Errorf("wait ping: %v", err)
			}

			err = sendPong()
			if err != nil {
				return fmt.Errorf("AdapterSend pong: %v", err)
			}

			ok := timer.Reset(pingWaitTime)
			if !ok {
				timer = time.NewTimer(pingWaitTime)
			}
		}
	}

	return nil
}

func SendPingWaitPong(ping func() error, pong <-chan error, isStop func() bool, pongWaitSecond int) error {
	pongWaitTime := time.Duration(pongWaitSecond) * time.Second
	pingPeriod := pongWaitTime / 2

	done := make(chan struct{})
	defer close(done)

	result := make(chan error, 2)

	sendPing := func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for !isStop() {
			select {
			case <-ticker.C:
				err := ping()
				if err != nil {
					result <- fmt.Errorf("Send ping: %v", err)
					return
				}

			case <-done:
				return
			}
		}
		result <- nil
	}

	waitPong := func() {
		timer := time.NewTimer(pongWaitTime)
		defer timer.Stop()

		for !isStop() {
			select {
			case <-timer.C:
				result <- errors.New("wait pong timeout")
				return

			case err := <-pong:
				if err != nil {
					result <- fmt.Errorf("handle pong: %v", err)
					return
				}

				ok := timer.Reset(pongWaitTime)
				if !ok {
					timer = time.NewTimer(pongWaitTime)
				}

			case <-done:
				return
			}
		}
		result <- nil
	}

	go sendPing()
	go waitPong()
	return <-result
}
