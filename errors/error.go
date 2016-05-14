package errors

import (
	"fmt"
	"runtime"

	"github.com/kataras/iris/logger"
)

// Error holds the error
type Error struct {
	message string
}

// Error returns the message of the actual error
func (e *Error) Error() string {
	return e.message
}

// Format returns a formatted new error based on the arguments
func (e *Error) Format(args ...interface{}) error {
	return fmt.Errorf(e.message, args)
}

// With does the same thing as Format but it receives an error type which if it's nil it returns a nil error
func (e *Error) With(err error) error {
	if err == nil {
		return nil
	}

	return e.Format(err.Error())
}

// Return returns the actual error as it is
func (e *Error) Return() error {
	return fmt.Errorf(e.message)
}

// Panic output the message and after panics
func (e *Error) Panic() {
	if e == nil {
		return
	}
	_, fn, line, _ := runtime.Caller(1)
	errMsg := e.message
	errMsg = "\nCaller was: " + fmt.Sprintf("%s:%d", fn, line)
	panic(errMsg)
}

// Panicf output the formatted message and after panics
func (e *Error) Panicf(args ...interface{}) {
	if e == nil {
		return
	}
	_, fn, line, _ := runtime.Caller(1)
	errMsg := e.Format(args...).Error()
	errMsg = "\nCaller was: " + fmt.Sprintf("%s:%d", fn, line)
	panic(errMsg)
}

//

// New creates and returns an Error with a message
func New(errMsg string) *Error {
	//	return &Error{fmt.Errorf("\n" + logger.Prefix + "Error: " + errMsg)}
	return &Error{message: "\n" + logger.Prefix + " Error: " + errMsg}
}

// Printf prints to the logger a specific error with optionally arguments
func Printf(logger *logger.Logger, err error, args ...interface{}) {
	logger.Printf(err.Error(), args...)
}
