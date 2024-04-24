package Artifex

import (
	"encoding/json"
	"runtime/debug"
)

type MW[Message any] struct {
	Logger Logger
}

func (mw MW[Message]) Recover() Middleware[Message] {
	if mw.Logger == nil {
		mw.Logger = DefaultLogger()
	}

	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(dep any, message *Message, route *RouteParam) error {

			defer func() {
				if r := recover(); r != nil {
					mw.Logger.Error("recovered from panic: %v", r)
					mw.Logger.Error("call stack: %v", string(debug.Stack()))
				}
			}()

			return next(dep, message, route)
		}
	}
}

func (mw MW[Message]) PrintError(getSubject NewSubjectFunc[Message]) Middleware[Message] {
	if mw.Logger == nil {
		mw.Logger = DefaultLogger()
	}

	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(dep any, message *Message, route *RouteParam) error {
			subject := getSubject(message)

			err := next(dep, message, route)
			if err != nil {
				mw.Logger.Error("handle %v fail: %v", subject, err)
			}
			mw.Logger.Info("handle %v ok", subject)

			return nil
		}
	}
}

func (mw MW[Message]) ExcludedSubject(excludeSubjects []string, getSubject NewSubjectFunc[Message]) Middleware[Message] {
	if mw.Logger == nil {
		mw.Logger = DefaultLogger()
	}

	excluded := make(map[string]bool, len(excludeSubjects))
	for i := 0; i < len(excluded); i++ {
		excluded[excludeSubjects[i]] = true
	}

	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(dep any, message *Message, route *RouteParam) error {
			subject := getSubject(message)
			if excluded[subject] {
				return nil
			}
			return next(dep, message, route)
		}
	}
}

func HandlePrintDetail[Message any](
	getSubject NewSubjectFunc[Message],
	newBody func(subject string) (any, bool),
	getByteBody func(*Message) []byte,
	unmarshal func(bBody []byte, body any) error,
	logger Logger,
) HandleFunc[Message] {

	return func(dep any, message *Message, route *RouteParam) error {
		subject := getSubject(message)

		if newBody == nil {
			logger.Debug("print %v", subject)
			return nil
		}

		body, ok := newBody(subject)
		if !ok {
			logger.Debug("print %v", subject)
			return nil
		}

		bBody := getByteBody(message)
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
