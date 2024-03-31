package Artifex

func NewPubSubOption[rMessage, sMessage any]() (opt *AdapterOption[rMessage, sMessage]) {
	return &AdapterOption[rMessage, sMessage]{}
}

func NewPublisherOption[sMessage any]() (opt *AdapterOption[struct{}, sMessage]) {
	return &AdapterOption[struct{}, sMessage]{}
}

func NewSubscriberOption[rMessage any]() (opt *AdapterOption[rMessage, struct{}]) {
	return &AdapterOption[rMessage, struct{}]{}
}

type AdapterOption[rMessage, sMessage any] struct {
	handleRecv  HandleFunc[rMessage]
	adapterRecv func(IAdapter) (*rMessage, error)
	adapterSend func(IAdapter, *sMessage) error
	adapterStop func(IAdapter, *sMessage) error

	fixupMaxRetrySecond int
	adapterFixup        func() error

	pingpong func(isStop func() bool) error

	identifier string
}

func (opt *AdapterOption[rMessage, sMessage]) HandleRecv(handleRecv HandleFunc[rMessage]) *AdapterOption[rMessage, sMessage] {
	opt.handleRecv = handleRecv
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterRecv(adapterRecv func(adp IAdapter) (*rMessage, error)) *AdapterOption[rMessage, sMessage] {
	opt.adapterRecv = adapterRecv
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterSend(adapterSend func(adp IAdapter, egress *sMessage) error) *AdapterOption[rMessage, sMessage] {
	opt.adapterSend = adapterSend
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterStop(adapterStop func(adp IAdapter, egress *sMessage) error) *AdapterOption[rMessage, sMessage] {
	opt.adapterStop = adapterStop
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) AdapterFixup(maxRetrySecond int, adapterFixup func() error) *AdapterOption[rMessage, sMessage] {
	opt.fixupMaxRetrySecond = maxRetrySecond
	opt.adapterFixup = adapterFixup
	return opt
}

// SendPing
//
// When SendPingWaitPong sends a ping message and waits for a corresponding pong message.
// SendPeriod = WaitSecond / 2
func (opt *AdapterOption[rMessage, sMessage]) SendPing(sendPing func() error, waitPong chan error, waitPongSecond int) *AdapterOption[rMessage, sMessage] {
	second := waitPongSecond
	if second <= 0 {
		second = 30
	}

	opt.pingpong = func(isStop func() bool) error {
		return SendPingWaitPong(sendPing, waitPong, isStop, second)
	}
	return opt
}

// WaitPing
//
// When WaitPingSendPong waits for a ping message and response a corresponding pong message.
// SendPeriod = WaitSecond
func (opt *AdapterOption[rMessage, sMessage]) WaitPing(waitPing chan error, waitPingSecond int, sendPong func() error) *AdapterOption[rMessage, sMessage] {
	second := waitPingSecond
	if second <= 0 {
		second = 30
	}

	opt.pingpong = func(isStop func() bool) error {
		return WaitPingSendPong(waitPing, sendPong, isStop, second)
	}
	return opt
}

func (opt *AdapterOption[rMessage, sMessage]) Identifier(identifier string) *AdapterOption[rMessage, sMessage] {
	opt.identifier = identifier
	return opt
}
