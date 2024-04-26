package Artifex

import (
	"encoding/json"
	"runtime/debug"
)

type Use struct {
	Logger func(dependency any) Logger
}

func (use Use) Recover() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			logger := use.Logger(dep)

			defer func() {
				if r := recover(); r != nil {
					logger.Error("recovered from panic: %v", r)
					logger.Error("call stack: %v", string(debug.Stack()))
				}
			}()

			return next(message, dep)
		}
	}
}

func (use Use) PrintError() func(message *Message, dep any, err error) error {
	return func(message *Message, dep any, err error) error {
		subject := message.Subject
		logger := use.Logger(dep)

		if err != nil {
			logger.Error("handle %v: %v", subject, err)
		}
		logger.Info("handle %v ok", subject)

		return nil
	}
}

func (use Use) Retry(retryMaxSecond int) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			task := func() error {
				return next(message, dep)
			}
			taskCanStop := func() bool {
				return false
			}
			return ReliableTask(task, taskCanStop, retryMaxSecond, nil)
		}
	}
}

func (use Use) ExcludedSubject(subjects []string) Middleware {
	excluded := make(map[string]bool, len(subjects))
	for i := 0; i < len(subjects); i++ {
		excluded[subjects[i]] = true
	}

	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			if excluded[message.Subject] {
				return nil
			}
			return next(message, dep)
		}
	}
}

func (use Use) PrintDetail(
	newBody func(subject string) (any, bool),
	unmarshal func(bBody []byte, body any) error,
) HandleFunc {

	return func(message *Message, dep any) error {
		subject := message.Subject
		logger := use.Logger(dep)

		if newBody == nil {
			logger.Debug("print %v", subject)
			return nil
		}

		body, ok := newBody(subject)
		if !ok {
			logger.Debug("print %v", subject)
			return nil
		}

		bBody := message.Bytes
		err := unmarshal(bBody, body)
		if err != nil {
			return err
		}

		bBody, err = json.Marshal(body)
		if err != nil {
			return err
		}

		logger.Info("print %q: %T=%v\n\n", subject, body, string(bBody))
		return nil
	}
}
