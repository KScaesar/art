package Artifex

import (
	"encoding/json"
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

		type getLogger interface {
			Log() Logger
		}

		getter, ok := dep.(getLogger)
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

func UseExcludedSubject(subjects []string) Middleware {
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

func (use Use) PrintError() func(message *Message, dep any, err error) error {
	return func(message *Message, dep any, err error) error {
		subject := message.Subject
		logger := use.logger(message, dep)

		if err != nil {
			logger.Error("handle %q: %v", subject, err)
			return err
		}
		logger.Info("handle %q ok", subject)
		return nil
	}
}

func (use Use) PrintDetail(
	newBody func(subject string) (body any, found bool),
	unmarshal func(bBody []byte, body any) error,
) HandleFunc {

	return func(message *Message, dep any) error {
		subject := message.Subject
		logger := use.logger(message, dep)
		bBody := message.Bytes

		switch {
		case newBody != nil && unmarshal != nil:
			body, ok := newBody(subject)
			if !ok {
				logger.Debug("print %q", subject)
				return nil
			}

			err := unmarshal(bBody, body)
			if err != nil {
				return err
			}

			bBody, err = json.Marshal(body)
			if err != nil {
				return err
			}

			logger.Debug("print %q: %T=%v", subject, body, string(bBody))
			return nil

		case newBody != nil && unmarshal == nil:
			body, ok := newBody(subject)
			if !ok {
				logger.Debug("print %q", subject)
				return nil
			}

			logger.Debug("print %q: %T=%v", subject, body, string(bBody))
			return nil

		case newBody == nil && unmarshal != nil:
			logger.Error("invalid middleware param: print %q", subject)
			return nil

		case newBody == nil && unmarshal == nil:
			logger.Debug("print %q: %v", subject, string(bBody))
			return nil
		}

		return nil
	}
}
