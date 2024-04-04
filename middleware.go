package Artifex

import (
	"runtime/debug"
)

func Recover[Message any]() Middleware[Message] {
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
