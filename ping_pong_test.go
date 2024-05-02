//go:build longtime
// +build longtime

package art

import (
	"fmt"
	"testing"
)

func Test_NoTimeout_When_ClientSendPingSeconds_LessThan_ServerWaitPingSeconds(t *testing.T) {
	clientSendPingSeconds := 1
	serverWaitPingSeconds := 2
	err := pingpong(t, clientSendPingSeconds, serverWaitPingSeconds)
	if err != nil {
		t.Errorf("pingping should success: %v", err)
		return
	}
}

func Test_HasTimeout_When_ClientSendPingSeconds_GreaterThan_ServerWaitPingSeconds(t *testing.T) {
	clientSendPingSeconds := 2
	serverWaitPingSeconds := 1
	err := pingpong(t, clientSendPingSeconds, serverWaitPingSeconds)
	if err == nil {
		t.Errorf("pingping should timeout")
	}
	// t.Logf("pingping fail: %v", err)
}

func pingpong(t *testing.T, clientSendPingSeconds, serverWaitPingSeconds int) error {
	sendPingCount := 0
	targetCount := 2
	isStop := func() bool { return sendPingCount == targetCount }

	clientWaitPong := NewWaitPingPong()
	serverWaitPing := NewWaitPingPong()
	notify := make(chan error, 2)

	client := func() {
		clientSendPing := func() error {
			sendPingCount++
			t.Logf("client send ping")
			serverWaitPing.Ack()
			t.Logf("server ack ping")
			return nil
		}
		err := SendPingWaitPong(clientSendPingSeconds, clientSendPing, clientWaitPong, isStop)
		if err != nil {
			notify <- fmt.Errorf("client: %w", err)
		}
		notify <- nil
	}

	server := func() {
		serverSendPong := func() error {
			t.Logf("server send pong")
			clientWaitPong.Ack()
			t.Logf("client ack pong")
			t.Logf("")
			return nil
		}
		err := WaitPingSendPong(serverWaitPingSeconds, serverWaitPing, serverSendPong, isStop)
		if err != nil {
			notify <- fmt.Errorf("server: %w", err)
			return
		}
		notify <- nil
	}

	go client()
	go server()
	return <-notify
}
