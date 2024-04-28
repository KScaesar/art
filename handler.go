package Artifex

import (
	"runtime/debug"
)

type HandleFunc func(message *Message, dep any) error

func (h HandleFunc) PreMiddleware() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			err := h(message, dep)
			if err != nil {
				return err
			}
			return next(message, dep)
		}
	}
}

func (h HandleFunc) PostMiddleware() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			err := next(message, dep)
			if err != nil {
				return err
			}
			return h(message, dep)
		}
	}
}

func (h HandleFunc) LinkMiddlewares(middlewares ...Middleware) HandleFunc {
	return LinkMiddlewares(h, middlewares...)
}

type Middleware func(next HandleFunc) HandleFunc

func (mw Middleware) HandleFunc() HandleFunc {
	return LinkMiddlewares(UseSkipMessage(), mw)
}

func LinkMiddlewares(handler HandleFunc, middlewares ...Middleware) HandleFunc {
	n := len(middlewares)
	for i := n - 1; 0 <= i; i-- {
		decorator := middlewares[i]
		handler = decorator(handler)
	}
	return handler
}

//

func UseLogger(withMsgId, attachToMessage bool) func(message *Message, dep any) Logger {
	return func(message *Message, dep any) Logger {
		if message.Logger != nil {
			return message.Logger
		}

		var logger Logger

		type Getter interface {
			Log() Logger
		}

		getter, ok := dep.(Getter)
		if !ok {
			logger = DefaultLogger()
		} else {
			logger = getter.Log()
		}

		if withMsgId {
			logger = logger.WithKeyValue("msg_id", message.MsgId())
		}

		if attachToMessage {
			message.Logger = logger
		}

		return logger
	}
}

func UseSkipMessage() func(message *Message, dep any) error {
	return func(message *Message, dep any) error { return nil }
}

func UseRetry(retryMaxSecond int) Middleware {
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

func UseExcluded(subjects []string) Middleware {
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

func UseIncluded(subjects []string) Middleware {
	included := make(map[string]bool, len(subjects))
	for i := 0; i < len(subjects); i++ {
		included[subjects[i]] = true
	}

	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			if included[message.Subject] {
				return next(message, dep)
			}
			return nil
		}
	}
}

type Use struct {
	Logger func(message *Message, dep any) Logger
}

func (use Use) logger(message *Message, dep any) Logger {
	if use.Logger == nil {
		return UseLogger(false, false)(message, dep)
	}
	return use.Logger(message, dep)
}

func (use Use) Recover() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			logger := use.logger(message, dep)

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

func (use Use) PrintResult(excludedSubjects []string) func(message *Message, dep any, err error) error {
	excluded := make(map[string]bool, len(excludedSubjects))
	for i := 0; i < len(excludedSubjects); i++ {
		excluded[excludedSubjects[i]] = true
	}

	return func(message *Message, dep any, err error) error {
		subject := message.Subject
		logger := use.logger(message, dep)

		if err != nil {
			logger.Error("handle %q: %v", subject, err)
			return err
		}

		if excluded[subject] {
			return nil
		}
		logger.Info("handle %q ok", subject)
		return nil
	}
}

func (use Use) PrintDetail() HandleFunc {
	return func(message *Message, dep any) error {
		subject := message.Subject
		logger := use.logger(message, dep)

		if message.Body != nil {
			logger.WithKeyValue("payload", message.Body).Debug("print %q: %T=%v", subject, message.Body)
			return nil
		}
		if len(message.Bytes) != 0 {
			logger.WithKeyValue("payload", message.Bytes).Debug("print %q", subject)
			return nil
		}
		return nil
	}
}
