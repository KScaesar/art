package art

import (
	"errors"
	"fmt"
)

var (
	ErrClosed          = NewCustomError(2001, "service has been closed")
	ErrNotFound        = NewCustomError(2100, "not found")
	ErrNotFoundSubject = NewCustomError(2101, "not found subject mux")
)

//

func ErrorExtractCode(err error) int {
	var Err *CustomError
	if errors.As(err, &Err) {
		return Err.myCode
	}

	unknown := 0
	return unknown
}

func ErrorJoin3rdPartyWithMsg(myErr error, Err3rd error, msg string, args ...any) error {
	return fmt.Errorf("%v: %w: %w", fmt.Sprintf(msg, args...), Err3rd, myErr)
}

func ErrorJoin3rdParty(myErr error, Err3rd error) error {
	return fmt.Errorf("%w: %w", Err3rd, myErr)
}

func ErrorWrapWithMessage(myErr error, msg string, args ...any) error {
	return fmt.Errorf("%v: %w", fmt.Sprintf(msg, args...), myErr)
}

func NewCustomError(myCode int, title string) *CustomError {
	return &CustomError{title: title, myCode: myCode}
}

type CustomError struct {
	title  string
	myCode int
}

func (c *CustomError) Error() string {
	return c.title
}

func (c *CustomError) MyCode() int {
	return c.myCode
}

func (c *CustomError) CustomError() {}
