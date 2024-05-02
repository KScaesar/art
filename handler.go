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

func UseAdHocFunc(AdHoc func(message *Message, dep any) error) HandleFunc {
	return AdHoc
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

func UseAsync() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			message = message.Copy()
			go func() {
				next(message, dep)
				PutMessage(message)
			}()
			return nil
		}
	}
}

type SafeConcurrencyKind int

const (
	SafeConcurrency_Skip SafeConcurrencyKind = iota
	SafeConcurrency_Mutex
	SafeConcurrency_Copy
)

func UseLogger(withMsgId bool, safeConcurrency SafeConcurrencyKind) Middleware {
	return func(next HandleFunc) HandleFunc {
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

			switch {
			case safeConcurrency == SafeConcurrency_Mutex:
				message.Mutex.Lock()
				defer message.Mutex.Unlock()

			case safeConcurrency == SafeConcurrency_Copy:
				message = message.Copy()
				defer PutMessage(message)
			}

			if withMsgId {
				logger = logger.WithKeyValue("msg_id", message.MsgId())
			}

			message.UpdateContext(
				func(ctx context.Context) context.Context {
					return CtxWithLogger(ctx, dep, logger)
				},
			)

			return next(message, dep)
		}
	}
}

func UseHowMuchTime() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			startTime := time.Now()
			defer func() {
				logger := CtxGetLogger(message.Ctx, dep)

				finishTime := time.Now()
				logger.Info("spend %v", finishTime.Sub(startTime))
			}()
			return next(message, dep)
		}
	}
}

func UsePrintResult(isEgress bool, ignoreOkSubjects []string) Middleware {
	ignore := make(map[string]bool, len(ignoreOkSubjects))
	for i := 0; i < len(ignoreOkSubjects); i++ {
		ignore[ignoreOkSubjects[i]] = true
	}

	return func(next HandleFunc) HandleFunc {
		return func(message *Message, dep any) error {
			err := next(message, dep)

			subject := message.Subject
			logger := CtxGetLogger(message.Ctx, dep)

			if err != nil {
				if isEgress {
					logger.Error("send %q fail: %v", subject, err)
				} else {
					logger.Error("handle %q fail: %v", subject, err)
				}
				return err
			}

			if ignore[subject] {
				return nil
			}

			if isEgress {
				logger.Info("send %q ok", subject)
			} else {
				logger.Info("handle %q ok", subject)
			}
			return nil
		}
	}
}

func UsePrintDetail() HandleFunc {
	return func(message *Message, dep any) error {
		logger := CtxGetLogger(message.Ctx, dep)

		if message.Body != nil {
			logger.Debug("print detail: %T %v", message.Body, AnyToString(message.Body))
			return nil
		}
		if len(message.Bytes) != 0 {
			logger.Debug("print detail: %v", AnyToString(message.Bytes))
			return nil
		}
		return nil
	}
}
