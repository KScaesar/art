package Artifex

import (
	"golang.org/x/exp/constraints"
)

type ConnectParam[Subject constraints.Ordered, rMessage, sMessage any] struct {
	RecvMux                 *MessageMux[Subject, rMessage]                           // Must
	NewAdapter              NewAdapterFunc[Subject, rMessage, sMessage]              // Must
	SpawnHandlers           []func(sess *Session[Subject, rMessage, sMessage]) error // Option
	ExitHandlers            []func(sess *Session[Subject, rMessage, sMessage])       // Option
	BackoffMaxElapsedMinute int                                                      // Option
}

func Connect[Subject constraints.Ordered, rMessage, sMessage any](param ConnectParam[Subject, rMessage, sMessage]) (*Session[Subject, rMessage, sMessage], error) {
	adapter, err := param.NewAdapter()
	if err != nil {
		return nil, err
	}
	sess, err := NewSession(param.RecvMux, adapter)
	if err != nil {
		return nil, err
	}

	if param.SpawnHandlers == nil {
		param.SpawnHandlers = make([]func(sess *Session[Subject, rMessage, sMessage]) error, 0)
	}
	if param.ExitHandlers == nil {
		param.ExitHandlers = make([]func(sess *Session[Subject, rMessage, sMessage]), 0)
	}

	for _, spawn := range param.SpawnHandlers {
		err = spawn(sess)
		if err == nil {
			continue
		}
		for _, exist := range param.ExitHandlers {
			exist(sess)
		}
		sess.Stop()
		return nil, err
	}

	go func() {
		defer func() {
			for _, exit := range param.ExitHandlers {
				exit(sess)
			}
			sess.Stop()
		}()

		cnt := 0
		tasker := PermanentTasker{
			Run: func() error {
				return sess.Listen()
			},
			ActiveStop: func() bool {
				return sess.IsStop()
			},
			SelfRepair: func() error {
				cnt++
				sess.Logger().Info("%v reconnect %v times", sess.Identifier, cnt)
				adapter, err = param.NewAdapter()
				if err != nil {
					return err
				}
				cnt = 0
				sess.Logger().Info("%v reconnect success", sess.Identifier)

				sess.recv = adapter.Recv
				sess.send = adapter.Send
				sess.stop = adapter.Stop
				sess.isListen.Store(false)
				return nil
			},
			BackoffMaxElapsedMinute: param.BackoffMaxElapsedMinute,
		}

		tasker.Start()
	}()

	return sess, nil
}
