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

func (life *Lifecycle[Subject, rMessage, sMessage]) execute(sess *Session[Subject, rMessage, sMessage]) error {
	err := life.spawn(sess)
	if err != nil {
		return err
	}

	if len(life.ExitHandlers) == 0 {
		return nil
	}

	go func() {
		notify := sess.Notify()
		select {
		case <-notify:
			life.exit(sess)
		}
	}()
	return nil
}

func (life *Lifecycle[Subject, rMessage, sMessage]) spawn(sess *Session[Subject, rMessage, sMessage]) error {
	if len(life.SpawnHandlers) == 0 {
		return nil
	}
	for _, enter := range life.SpawnHandlers {
		err := enter(sess)
		if err != nil {
			life.exit(sess)
			return err
		}
	}
	return nil
}

func (life *Lifecycle[Subject, rMessage, sMessage]) exit(sess *Session[Subject, rMessage, sMessage]) {
	if len(life.ExitHandlers) == 0 {
		return
	}
	for _, action := range life.ExitHandlers {
		action(sess)
	}
	return
}
