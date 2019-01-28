package httputil

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/kataras/iris"
)

var validStackFuncs = []func(string) bool{
	func(file string) bool {
		return strings.Contains(file, "/mongodb/api/")
	},
}

// RuntimeCallerStack returns the app's `file:line` stacktrace
// to give more information about an error cause.
func RuntimeCallerStack() (s string) {
	var pcs [10]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		for _, fn := range validStackFuncs {
			if fn(frame.File) {
				s += fmt.Sprintf("\n\t\t\t%s:%d", frame.File, frame.Line)
			}
		}

		if !more {
			break
		}
	}

	return s
}

// HTTPError describes an HTTP error.
type HTTPError struct {
	error
	Stack       string    `json:"-"` // the whole stacktrace.
	CallerStack string    `json:"-"` // the caller, file:lineNumber
	When        time.Time `json:"-"` // the time that the error occurred.
	// ErrorCode int: maybe a collection of known error codes.
	StatusCode int `json:"statusCode"`
	// could be named as "reason" as well
	//  it's the message of the error.
	Description string `json:"description"`
}

func newError(statusCode int, err error, format string, args ...interface{}) HTTPError {
	if format == "" {
		format = http.StatusText(statusCode)
	}

	desc := fmt.Sprintf(format, args...)
	if err == nil {
		err = errors.New(desc)
	}

	return HTTPError{
		err,
		string(debug.Stack()),
		RuntimeCallerStack(),
		time.Now(),
		statusCode,
		desc,
	}
}

func (err HTTPError) writeHeaders(ctx iris.Context) {
	ctx.StatusCode(err.StatusCode)
	ctx.Header("X-Content-Type-Options", "nosniff")
}

// LogFailure will print out the failure to the "logger".
func LogFailure(logger io.Writer, ctx iris.Context, err HTTPError) {
	timeFmt := err.When.Format("2006/01/02 15:04:05")
	firstLine := fmt.Sprintf("%s %s: %s", timeFmt, http.StatusText(err.StatusCode), err.Error())
	whitespace := strings.Repeat(" ", len(timeFmt)+1)
	fmt.Fprintf(logger, "%s\n%sIP: %s\n%sURL: %s\n%sSource: %s\n",
		firstLine, whitespace, ctx.RemoteAddr(), whitespace, ctx.FullRequestURI(), whitespace, err.CallerStack)
}

// Fail will send the status code, write the error's reason
// and return the HTTPError for further use, i.e logging, see `InternalServerError`.
func Fail(ctx iris.Context, statusCode int, err error, format string, args ...interface{}) HTTPError {
	httpErr := newError(statusCode, err, format, args...)
	httpErr.writeHeaders(ctx)

	ctx.WriteString(httpErr.Description)
	return httpErr
}

// FailJSON will send to the client the error data as JSON.
// Useful for APIs.
func FailJSON(ctx iris.Context, statusCode int, err error, format string, args ...interface{}) HTTPError {
	httpErr := newError(statusCode, err, format, args...)
	httpErr.writeHeaders(ctx)

	ctx.JSON(httpErr)

	return httpErr
}

// InternalServerError logs to the server's terminal
// and dispatches to the client the 500 Internal Server Error.
// Internal Server errors are critical, so we log them to the `os.Stderr`.
func InternalServerError(ctx iris.Context, err error, format string, args ...interface{}) {
	LogFailure(os.Stderr, ctx, Fail(ctx, iris.StatusInternalServerError, err, format, args...))
}

// InternalServerErrorJSON acts exactly like `InternalServerError` but instead it sends the data as JSON.
// Useful for APIs.
func InternalServerErrorJSON(ctx iris.Context, err error, format string, args ...interface{}) {
	LogFailure(os.Stderr, ctx, FailJSON(ctx, iris.StatusInternalServerError, err, format, args...))
}

// UnauthorizedJSON sends JSON format of StatusUnauthorized(401) HTTPError value.
func UnauthorizedJSON(ctx iris.Context, err error, format string, args ...interface{}) HTTPError {
	return FailJSON(ctx, iris.StatusUnauthorized, err, format, args...)
}
