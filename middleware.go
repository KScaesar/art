package Artifex

import (
	"runtime/debug"
)

func MW_Recover[Message any]() Middleware[Message] {
	logger := DefaultLogger()
	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(message *Message, route *RouteParam) error {

			defer func() {
				if r := recover(); r != nil {
					logger.Error("recovered from panic: %v", r)
					logger.Error("call stack: %v", string(debug.Stack()))
				}
			}()

			return next(message, route)
		}
	}
}

func MW_Error[Message any](getSubject NewSubjectFunc[Message]) Middleware[Message] {
	logger := DefaultLogger()
	return func(next HandleFunc[Message]) HandleFunc[Message] {
		return func(message *Message, route *RouteParam) error {
			subject := getSubject(message)
			err := next(message, route)
			if err != nil {
				logger.Error("handle %v fail: %v", subject, err)
			}
			logger.Info("handle %v success", subject)
			return nil
		}
	}
}
