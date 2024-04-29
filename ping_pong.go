package Artifex

import (
	"errors"
	"fmt"
	"time"
)

func NewWaitPingPong() WaitPingPong {
	wait := make(WaitPingPong, 1)
	return wait
}

type WaitPingPong chan struct{}

func (wait WaitPingPong) Ack() {
	select {
	case wait <- struct{}{}:
	default:
	}
}

func WaitPingSendPong(waitPingSeconds int, waitPing WaitPingPong, sendPong func() error, isStop func() bool) error {
	waitPingTime := time.Duration(waitPingSeconds) * time.Second

	waitPingTimer := time.NewTimer(waitPingTime)
	defer waitPingTimer.Stop()

	for !isStop() {
		select {
		case <-waitPingTimer.C:
			if isStop() {
				return nil
			}

			return errors.New("wait ping timeout")

		case <-waitPing:
			err := sendPong()
			if err != nil {
				return fmt.Errorf("send pong: %v", err)
			}

			ok := waitPingTimer.Reset(waitPingTime)
			if !ok {
				waitPingTimer = time.NewTimer(waitPingTime)
			}
		}
	}

	return nil
}

func SendPingWaitPong(sendPingSeconds int, sendPing func() error, waitPong WaitPingPong, isStopped func() bool) error {
	sendPingPeriod := time.Duration(sendPingSeconds) * time.Second
	waitPongTime := sendPingPeriod * 2

	done := make(chan struct{})
	defer close(done)

	result := make(chan error, 2)

	go func() {
		sendPingTicker := time.NewTicker(sendPingPeriod)
		defer sendPingTicker.Stop()

		for !isStopped() {
			select {
			case <-sendPingTicker.C:
				if isStopped() {
					result <- nil
					return
				}

				err := sendPing()
				if err != nil {
					result <- fmt.Errorf("send ping: %v", err)
					return
				}

			case <-done:
				return
			}
		}
		result <- nil
	}()

	go func() {
		waitPongTimer := time.NewTimer(waitPongTime)
		defer waitPongTimer.Stop()

		for !isStopped() {
			select {
			case <-waitPongTimer.C:
				if isStopped() {
					result <- nil
					return
				}

				result <- errors.New("wait pong timeout")
				return

			case <-waitPong:
				ok := waitPongTimer.Reset(waitPongTime)
				if !ok {
					waitPongTimer = time.NewTimer(waitPongTime)
				}

			case <-done:
				return
			}
		}
		result <- nil
	}()

	return <-result
}
