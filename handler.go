package Artifex

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
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

func (h HandleFunc) Link(middlewares ...Middleware) HandleFunc {
	return Link(h, middlewares...)
}

type Middleware func(next HandleFunc) HandleFunc

func (mw Middleware) HandleFunc() HandleFunc {
	return Link(UseSkipMessage(), mw)
}

func (mw Middleware) Link(handler HandleFunc) HandleFunc {
	return Link(handler, mw)
}

func Link(handler HandleFunc, middlewares ...Middleware) HandleFunc {
	n := len(middlewares)
	for i := n - 1; 0 <= i; i-- {
		decorator := middlewares[i]
		handler = decorator(handler)
	}
	return handler
}

//

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

func UseExclude(subjects []string) Middleware {
	exclude := make(map[string]bool, len(subjects))
	for i := 0; i < len(subjects); i++ {
		exclude[subjects[i]] = true
	}

	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			if exclude[message.Subject] {
				return nil
			}
			return next(message, dep)
		}
	}
}

func UseInclude(subjects []string) Middleware {
	include := make(map[string]bool, len(subjects))
	for i := 0; i < len(subjects); i++ {
		include[subjects[i]] = true
	}

	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			if include[message.Subject] {
				return next(message, dep)
			}
			return nil
		}
	}
}

func UseRecover() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("recovered from panic: %v\n%v", r, string(debug.Stack()))
				}
			}()
			return next(message, dep)
		}
	}
}

func UseLogger(withMsgId bool) HandleFunc {
	return func(message *Message, dep any) error {
		type Getter interface {
			Log() Logger
		}

		var logger Logger
		getter, ok := dep.(Getter)
		if !ok {
			logger = DefaultLogger()
		} else {
			logger = getter.Log()
		}

		if withMsgId {
			logger = logger.WithKeyValue("msg_id", message.MsgId())
		}

		message.Mutex.Lock()
		defer message.Mutex.Unlock()
		message.UpdateContext(
			func(ctx context.Context) context.Context {
				return CtxWithLogger(ctx, logger)
			},
		)

		return nil
	}
}

func UsePrintResult() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			subject := message.Subject
			logger := CtxGetLogger(message.Ctx)

			err := next(message, dep)
			if err != nil {
				logger.Error("handle %q: %v", subject, err)
				return err
			}
			logger.Info("handle %q ok", subject)
			return nil
		}
	}
}

func UseHowMuchTime() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			startTime := time.Now()
			defer func() {
				subject := message.Subject
				logger := CtxGetLogger(message.Ctx)

				finishTime := time.Now()
				logger.Info("handle %q spend %v", subject, finishTime.Sub(startTime))
			}()
			return next(message, dep)
		}
	}
}

func UsePrintDetail() HandleFunc {
	return func(message *Message, dep any) error {
		subject := message.Subject
		logger := CtxGetLogger(message.Ctx)

		if message.Body != nil {
			logger.Debug("print %q: %T %v", subject, message.Body, AnyToString(message.Body))
			return nil
		}
		if len(message.Bytes) != 0 {
			logger.Debug("print %q: %v", subject, AnyToString(message.Bytes))
			return nil
		}
		return nil
	}
}
