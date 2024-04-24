package api

import (
	"fmt"
	"time"

	"github.com/kataras/iris/v12"
)

// Error holds the error sent by server to clients (JSON).
type Error struct {
	StatusCode int    `json:"code"`
	Method     string `json:"-"`
	Path       string `json:"-"`
	Message    string `json:"message"`
	Timestamp  int64  `json:"timestamp"`
}

func newError(statusCode int, method, path, format string, args ...interface{}) Error {
	msg := format
	if len(args) > 0 {
		// why we check for that? If the original error message came from our database
		// and it contains fmt-reserved words
		// like %s or %d we will get MISSING(=...) in our error message and we don't want that.
		msg = fmt.Sprintf(msg, args...)
	}

	if msg == "" {
		msg = iris.StatusText(statusCode)
	}

	return Error{
		StatusCode: statusCode,
		Method:     method,
		Path:       path,
		Message:    msg,
		Timestamp:  time.Now().Unix(),
	}
}

// Error implements the internal Go error interface.
func (e Error) Error() string {
	return fmt.Sprintf("[%d] %s: %s: %s", e.StatusCode, e.Method, e.Path, e.Message)
}

// Is implements the standard `errors.Is` internal interface.
// Usage: errors.Is(e, target)
func (e Error) Is(target error) bool {
	if target == nil {
		return false
	}

	err, ok := target.(Error)
	if !ok {
		return false
	}

	return (err.StatusCode == e.StatusCode || e.StatusCode == 0) &&
		(err.Message == e.Message || e.Message == "")
}
