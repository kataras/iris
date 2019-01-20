package errors

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/iris-contrib/go.uuid"
)

var (
	// Prefix the error prefix, applies to each error's message.
	Prefix = ""
)

// Error holds the error message, this message never really changes
type Error struct {
	// ID returns the unique id of the error, it's needed
	// when we want to check if a specific error returned
	// but the `Error() string` value is not the same because the error may be dynamic
	// by a `Format` call.
	ID string `json:"id"`
	// The message of the error.
	Message string `json:"message"`
	// Apennded is true whenever it's a child error.
	Appended bool `json:"appended"`
	// Stack returns the list of the errors that are shown at `Error() string`.
	Stack []Error `json:"stack"` // filled on AppendX.
}

// New creates and returns an Error with a pre-defined user output message
// all methods below that doesn't accept a pointer receiver because actually they are not changing the original message
func New(errMsg string) Error {
	uidv4, _ := uuid.NewV4() // skip error.
	return Error{
		ID:      uidv4.String(),
		Message: Prefix + errMsg,
	}
}

// NewFromErr same as `New` but pointer for nil checks without the need of the `Return()` function.
func NewFromErr(err error) *Error {
	if err == nil {
		return nil
	}

	errp := New(err.Error())
	return &errp
}

// Equal returns true if "e" and "to" are matched, by their IDs if it's a core/errors type otherwise it tries to match their error messages.
// It will always returns true if the "to" is a children of "e"
// or the error messages are exactly the same, otherwise false.
func (e Error) Equal(to error) bool {
	if e2, ok := to.(Error); ok {
		return e.ID == e2.ID
	} else if e2, ok := to.(*Error); ok {
		return e.ID == e2.ID
	}

	return e.Error() == to.Error()
}

// Empty returns true if the "e" Error has no message on its stack.
func (e Error) Empty() bool {
	return e.Message == ""
}

// NotEmpty returns true if the "e" Error has got a non-empty message on its stack.
func (e Error) NotEmpty() bool {
	return !e.Empty()
}

// String returns the error message
func (e Error) String() string {
	return e.Message
}

// Error returns the message of the actual error
// implements the error
func (e Error) Error() string {
	return e.String()
}

// Format returns a formatted new error based on the arguments
// it does NOT change the original error's message
func (e Error) Format(a ...interface{}) Error {
	e.Message = fmt.Sprintf(e.Message, a...)
	return e
}

func omitNewLine(message string) string {
	if strings.HasSuffix(message, "\n") {
		return message[0 : len(message)-2]
	} else if strings.HasSuffix(message, "\\n") {
		return message[0 : len(message)-3]
	}
	return message
}

// AppendInline appends an error to the stack.
// It doesn't try to append a new line if needed.
func (e Error) AppendInline(format string, a ...interface{}) Error {
	msg := fmt.Sprintf(format, a...)
	e.Message += msg
	e.Appended = true
	e.Stack = append(e.Stack, New(omitNewLine(msg)))
	return e
}

// Append adds a message to the predefined error message and returns a new error
// it does NOT change the original error's message
func (e Error) Append(format string, a ...interface{}) Error {
	// if new line is false then append this error but first
	// we need to add a new line to the first, if it was true then it has the newline already.
	if e.Message != "" {
		e.Message += "\n"
	}

	return e.AppendInline(format, a...)
}

// AppendErr adds an error's message to the predefined error message and returns a new error.
// it does NOT change the original error's message
func (e Error) AppendErr(err error) Error {
	return e.Append(err.Error())
}

// HasStack returns true if the Error instance is created using Append/AppendInline/AppendErr funcs.
func (e Error) HasStack() bool {
	return len(e.Stack) > 0
}

// With does the same thing as Format but it receives an error type which if it's nil it returns a nil error.
func (e Error) With(err error) error {
	if err == nil {
		return nil
	}

	return e.Format(err.Error())
}

// Ignore will ignore the "err" and return nil.
func (e Error) Ignore(err error) error {
	if err == nil {
		return e
	}
	if e.Error() == err.Error() {
		return nil
	}
	return e
}

// Panic output the message and after panics.
func (e Error) Panic() {
	_, fn, line, _ := runtime.Caller(1)
	errMsg := e.Message
	errMsg += "\nCaller was: " + fmt.Sprintf("%s:%d", fn, line)
	panic(errMsg)
}

// Panicf output the formatted message and after panics.
func (e Error) Panicf(args ...interface{}) {
	_, fn, line, _ := runtime.Caller(1)
	errMsg := e.Format(args...).Error()
	errMsg += "\nCaller was: " + fmt.Sprintf("%s:%d", fn, line)
	panic(errMsg)
}
