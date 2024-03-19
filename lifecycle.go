package Artifex

import (
	"golang.org/x/exp/constraints"
)

// Lifecycle define a management mechanism when session creation and session end.
//
// SpawnHandlers are used for additional operations during session creation.
// ExitHandlers are used for cleanup operations when the session ends.
type Lifecycle[Subject constraints.Ordered, rMessage, sMessage any] struct {
	SpawnHandlers []func(sess *Session[Subject, rMessage, sMessage]) error
	ExitHandlers  []func(sess *Session[Subject, rMessage, sMessage])
}

func (life *Lifecycle[Subject, rMessage, sMessage]) Spawn(sess *Session[Subject, rMessage, sMessage]) error {
	if life.SpawnHandlers == nil {
		return nil
	}
	for _, enter := range life.SpawnHandlers {
		err := enter(sess)
		if err != nil {
			life.Exit(sess)
			return err
		}
	}
	return nil
}

func (life *Lifecycle[Subject, rMessage, sMessage]) Exit(sess *Session[Subject, rMessage, sMessage]) {
	if life.ExitHandlers == nil {
		return
	}
	for _, action := range life.ExitHandlers {
		action(sess)
	}
	return
}
