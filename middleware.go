package Artifex

import (
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
		return func(message *Message, route *RouteParam) error {

			defer func() {
				if r := recover(); r != nil {
					mw.Logger.Error("recovered from panic: %v", r)
					mw.Logger.Error("call stack: %v", string(debug.Stack()))
				}
			}()

			return next(message, route)
		}
	}
}

func (mw MW[Message]) PrintError(getSubject NewSubjectFunc[Message]) Middleware[Message] {
	if mw.Logger == nil {
		mw.Logger = DefaultLogger()
	}

	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(message *Message, route *RouteParam) error {
			subject := getSubject(message)

			err := next(message, route)
			if err != nil {
				mw.Logger.Error("handle %v fail: %v", subject, err)
			}
			mw.Logger.Info("handle %v success", subject)

			return nil
		}
	}
}
