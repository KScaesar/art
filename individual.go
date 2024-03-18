package Artifex

import (
	"golang.org/x/exp/constraints"
)

// Roamer can manage the lifecycle of a single Session.
// On the contrary, Artist can manage the lifecycle of multiple Sessions.
//
//   - RecvMux is a multiplexer used for handle messages.
//   - NewAdapter is used to create a new adapter.
//   - SpawnHandlers are used for additional operations during session creation.
//   - ExitHandlers are used for cleanup operations when the session ends.
type Roamer[Subject constraints.Ordered, rMessage, sMessage any] struct {
	RecvMux       *Mux[Subject, rMessage]                                  // Must
	NewAdapter    NewAdapterFunc[Subject, rMessage, sMessage]              // Must
	SpawnHandlers []func(sess *Session[Subject, rMessage, sMessage]) error // Option
	ExitHandlers  []func(sess *Session[Subject, rMessage, sMessage])       // Option
}

func (individual Roamer[Subject, rMessage, sMessage]) Connect() (*Session[Subject, rMessage, sMessage], error) {
	adapter, err := individual.NewAdapter()
	if err != nil {
		return nil, err
	}

	sess, err := NewSession(individual.RecvMux, adapter)
	if err != nil {
		return nil, err
	}

	if individual.SpawnHandlers == nil {
		individual.SpawnHandlers = make([]func(sess *Session[Subject, rMessage, sMessage]) error, 0)
	}
	if individual.ExitHandlers == nil {
		individual.ExitHandlers = make([]func(sess *Session[Subject, rMessage, sMessage]), 0)
	}

	for _, spawn := range individual.SpawnHandlers {
		err = spawn(sess)
		if err == nil {
			continue
		}
		defer sess.Stop()
		for _, exist := range individual.ExitHandlers {
			exist(sess)
		}
		return nil, err
	}

	go func() {
		defer sess.Stop()
		defer func() {
			for _, exit := range individual.ExitHandlers {
				exit(sess)
			}
		}()
		sess.Listen()
	}()

	return sess, nil
}
