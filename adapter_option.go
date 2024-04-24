package Artifex

func NewPubSubOption[Ingress, Egress any]() (opt *AdapterOption[Ingress, Egress]) {
	pubsub := &Adapter[Ingress, Egress]{
		recvResult: make(chan error, 2),
		lifecycle:  new(Lifecycle),
		waitStop:   make(chan struct{}),
	}
	return &AdapterOption[Ingress, Egress]{
		adapter: pubsub,
	}
}

func NewPublisherOption[Egress any]() (opt *AdapterOption[struct{}, Egress]) {
	pub := &Adapter[struct{}, Egress]{
		recvResult: make(chan error, 2),
		lifecycle:  new(Lifecycle),
		waitStop:   make(chan struct{}),
	}
	return &AdapterOption[struct{}, Egress]{
		adapter: pub,
	}
}

func NewSubscriberOption[Ingress any]() (opt *AdapterOption[Ingress, struct{}]) {
	sub := &Adapter[Ingress, struct{}]{
		recvResult: make(chan error, 2),
		lifecycle:  new(Lifecycle),
		waitStop:   make(chan struct{}),
	}
	return &AdapterOption[Ingress, struct{}]{
		adapter: sub,
	}
}

type AdapterOption[Ingress, Egress any] struct {
	adapter         *Adapter[Ingress, Egress]
	decorateAdapter func(adp IAdapter) (app IAdapter)
}

func (opt *AdapterOption[Ingress, Egress]) Build() (adp IAdapter, err error) {
	pubsub := opt.adapter

	if pubsub.logger == nil {
		pubsub.logger = DefaultLogger()
	}

	if opt.decorateAdapter != nil {
		pubsub.application = opt.decorateAdapter(pubsub)
	} else {
		pubsub.application = pubsub
	}

	if pubsub.hub != nil {
		err = pubsub.hub.Join(pubsub.identifier, pubsub.application)
		if err != nil {
			return nil, err
		}
	}

	err = pubsub.lifecycle.initialize(pubsub.application)
	if err != nil {
		return nil, err
	}

	pubsub.pingpong()

	return pubsub.application, nil
}

func (opt *AdapterOption[Ingress, Egress]) DecorateAdapter(wrap func(adapter IAdapter) (application IAdapter)) *AdapterOption[Ingress, Egress] {
	opt.decorateAdapter = wrap
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) Identifier(identifier string) *AdapterOption[Ingress, Egress] {
	pubsub := opt.adapter
	pubsub.identifier = identifier
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) Logger(logger Logger) *AdapterOption[Ingress, Egress] {
	pubsub := opt.adapter
	pubsub.logger = logger
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterHub(hub AdapterHub) *AdapterOption[Ingress, Egress] {
	pubsub := opt.adapter
	pubsub.hub = hub
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) Lifecycle(setup func(life *Lifecycle)) *AdapterOption[Ingress, Egress] {
	if setup != nil {
		setup(opt.adapter.lifecycle)
	}
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) IngressMux(mux *Mux[Ingress]) *AdapterOption[Ingress, Egress] {
	sub := opt.adapter
	sub.ingressMux = mux
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterRecv(adapterRecv func(logger Logger) (message *Ingress, err error)) *AdapterOption[Ingress, Egress] {
	sub := opt.adapter
	sub.adapterRecv = adapterRecv
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) EgressMux(mux *Mux[Egress]) *AdapterOption[Ingress, Egress] {
	sub := opt.adapter
	sub.egressMux = mux
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterSend(adapterSend func(logger Logger, message *Egress) error) *AdapterOption[Ingress, Egress] {
	pub := opt.adapter
	pub.adapterSend = adapterSend
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterStop(adapterStop func(logger Logger) error) *AdapterOption[Ingress, Egress] {
	pubsub := opt.adapter
	pubsub.adapterStop = adapterStop
	return opt
}

func (opt *AdapterOption[Ingress, Egress]) AdapterFixup(maxRetrySecond int, adapterFixup func(IAdapter) error) *AdapterOption[Ingress, Egress] {
	pubsub := opt.adapter
	pubsub.fixupMaxRetrySecond = maxRetrySecond
	pubsub.adapterFixup = adapterFixup
	return opt
}

// SendPing
//
// When SendPingWaitPong sends a ping message and waits for a corresponding pong message.
// SendPeriod = WaitSecond / 2
func (opt *AdapterOption[Ingress, Egress]) SendPing(sendPing func() error, waitPong chan error, waitPongSecond int) *AdapterOption[Ingress, Egress] {
	second := waitPongSecond
	if second <= 0 {
		second = 30
	}

	pubsub := opt.adapter
	pubsub.pp = func() error { return SendPingWaitPong(sendPing, waitPong, pubsub.IsStopped, second) }
	return opt
}

// WaitPing
//
// When WaitPingSendPong waits for a ping message and response a corresponding pong message.
// SendPeriod = WaitSecond
func (opt *AdapterOption[Ingress, Egress]) WaitPing(waitPing chan error, waitPingSecond int, sendPong func() error) *AdapterOption[Ingress, Egress] {
	second := waitPingSecond
	if second <= 0 {
		second = 30
	}

	pubsub := opt.adapter
	pubsub.pp = func() error { return WaitPingSendPong(waitPing, sendPong, pubsub.IsStopped, second) }
	return opt
}
