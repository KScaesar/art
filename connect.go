package Artifex

import (
	"golang.org/x/exp/constraints"
)

type ConnectParam[Subject constraints.Ordered, rMessage, sMessage any] struct {
	RecvMux       *MessageMux[Subject, rMessage]                           // Must
	NewAdapter    NewAdapterFunc[Subject, rMessage, sMessage]              // Must
	SpawnHandlers []func(sess *Session[Subject, rMessage, sMessage]) error // Option
	ExitHandlers  []func(sess *Session[Subject, rMessage, sMessage])       // Option
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
		defer sess.Stop()
		for _, exist := range param.ExitHandlers {
			exist(sess)
		}
		return nil, err
	}

	go func() {
		defer sess.Stop()
		defer func() {
			for _, exit := range param.ExitHandlers {
				exit(sess)
			}
		}()
		sess.Listen()
	}()

	return sess, nil
}
