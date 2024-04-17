package Artifex

import (
	"errors"
	"fmt"
	"time"
)

func WaitPingSendPong(waitPing <-chan error, sendPong func() error, isStop func() bool, waitPingSecond int) error {
	waitPingTime := time.Duration(waitPingSecond) * time.Second

	waitPingTimer := time.NewTimer(waitPingTime)
	defer waitPingTimer.Stop()

	for !isStop() {
		select {
		case <-waitPingTimer.C:
			if isStop() {
				return nil
			}

			return errors.New("wait ping timeout")

		case err := <-waitPing:
			if err != nil {
				return fmt.Errorf("handle ping: %v", err)
			}

			err = sendPong()
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

func SendPingWaitPong(sendPing func() error, waitPong <-chan error, isStopped func() bool, waitPongSecond int) error {
	waitPongTime := time.Duration(waitPongSecond) * time.Second
	sendPingPeriod := waitPongTime / 2

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

			case err := <-waitPong:
				if err != nil {
					result <- fmt.Errorf("handle pong: %v", err)
					return
				}

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
