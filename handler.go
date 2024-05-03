package art

import (
	"context"
	"errors"
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

func UseGenericFunc[Dep any, H func(*Message, *Dep) error](handler H) HandleFunc {
	return func(message *Message, dependency any) error {
		dep := dependency.(*Dep)
		return handler(message, dep)
	}
}

func UseAdHocFunc(AdHoc HandleFunc) HandleFunc {
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

			if safeConcurrency == SafeConcurrency_Mutex {
				message.Mutex.Unlock()
			}

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

type UsePrintResult struct {
	ignoreErrors      []error
	ignoreErrSubjects map[string]bool
	ignoreOkSubjects  map[string]bool
	printIngress      bool
	printEgress       bool
}

func (use UsePrintResult) IgnoreErrors(errs ...error) UsePrintResult {
	use.ignoreErrors = append(use.ignoreErrors, errs...)
	return use
}

func (use UsePrintResult) IgnoreErrSubjects(subjects ...string) UsePrintResult {
	if use.ignoreErrSubjects == nil {
		use.ignoreErrSubjects = make(map[string]bool)
	}

	for _, subject := range subjects {
		use.ignoreErrSubjects[subject] = true
	}
	return use
}

func (use UsePrintResult) IgnoreOkSubjects(subjects ...string) UsePrintResult {
	if use.ignoreOkSubjects == nil {
		use.ignoreOkSubjects = make(map[string]bool)
	}

	for _, subject := range subjects {
		use.ignoreOkSubjects[subject] = true
	}
	return use
}

func (use UsePrintResult) PrintIngress() UsePrintResult {
	use.printIngress = true
	return use
}

func (use UsePrintResult) PrintEgress() UsePrintResult {
	use.printEgress = true
	return use
}

func (use UsePrintResult) PostMiddleware(next HandleFunc) HandleFunc {
	if use.ignoreErrSubjects == nil {
		use.ignoreErrSubjects = make(map[string]bool)
	}
	if use.ignoreOkSubjects == nil {
		use.ignoreOkSubjects = make(map[string]bool)
	}

	return func(message *Message, dep any) error {
		err := next(message, dep)

		subject := message.Subject
		logger := CtxGetLogger(message.Ctx, dep)

		if err != nil {
			for i := range use.ignoreErrors {
				if errors.Is(err, use.ignoreErrors[i]) {
					return err
				}
			}

			if use.ignoreErrSubjects[subject] {
				return err
			}

			if use.printIngress {
				logger.Error("handle %q fail: %v", subject, err)
				return err
			}

			if use.printEgress {
				logger.Error("send %q fail: %v", subject, err)
				return err
			}

			return err
		}

		if use.ignoreOkSubjects[subject] {
			return nil
		}

		if use.printIngress {
			logger.Info("handle %q ok", subject)
			return nil
		}

		if use.printEgress {
			logger.Info("send %q ok", subject)
			return nil
		}

		return nil
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
