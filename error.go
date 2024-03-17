package Artifex

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrUniversal        = NewCustomError(2000, "universal error")
	ErrClosed           = NewCustomError(2000, "service has been closed")
	ErrNotFound         = NewCustomError(2100, "not found")
	ErrInvalidParameter = NewCustomError(2200, "invalid parameter")
	ErrTimeout          = NewCustomError(2300, "timeout")
)

func ConvertErrNetwork(err error) error {
	if os.IsTimeout(err) {
		return ErrorJoin3rdParty(ErrTimeout, err)
	}
	return ErrorJoin3rdParty(ErrUniversal, err)
}

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
