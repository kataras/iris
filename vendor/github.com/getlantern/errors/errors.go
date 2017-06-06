/*
Package errors defines error types used across Lantern project.

	n, err := Foo()
	if err != nil {
	    return n, errors.New("Unable to do Foo: %v", err)
	}

or

  n, err := Foo()
	return n, errors.Wrap(err)

New() method will create a new error with err as its cause. Wrap will wrap err,
returning nil if err is nil.  If err is an error from Go's standard library,
errors will extract details from that error, at least the Go type name and the
return value of err.Error().

One can record the operation on which the error occurred using Op():

  return n, errors.New("Unable to do Foo: %v", err).Op("FooDooer")

One can also record additional data:

  return n, errors.
		New("Unable to do Foo: %v", err).
		Op("FooDooer").
		With("mydata", "myvalue").
		With("moredata", 5)

When used with github.com/getlantern/ops, Error captures its current context
and propagates that data for use in calling layers.

When used with github.com/getlantern/golog, Error provides stacktraces:

	Hello World
		at github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)
		at testing.tRunner (testing.go:999)
		at runtime.goexit (asm_amd999.s:999)
	Caused by: World
		at github.com/getlantern/errors.buildCause (errors_test.go:999)
		at github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)
		at testing.tRunner (testing.go:999)
		at runtime.goexit (asm_amd999.s:999)
	Caused by: orld
	Caused by: ld
		at github.com/getlantern/errors.buildSubSubCause (errors_test.go:999)
		at github.com/getlantern/errors.buildSubCause (errors_test.go:999)
		at github.com/getlantern/errors.buildCause (errors_test.go:999)
		at github.com/getlantern/errors.TestNewWithCause (errors_test.go:999)
		at testing.tRunner (testing.go:999)
		at runtime.goexit (asm_amd999.s:999)
	Caused by: d

It's the caller's responsibility to avoid race conditions accessing the same
error instance from multiple goroutines.
*/
package errors

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/getlantern/context"
	"github.com/getlantern/hidden"
	"github.com/getlantern/ops"
	"github.com/getlantern/stack"
)

// Error wraps system and application defined errors in unified structure for
// reporting and logging. It's not meant to be created directly. User New(),
// Wrap() and Report() instead.
type Error interface {
	error
	context.Contextual

	// ErrorClean returns a non-parameterized version of the error whenever
	// possible. For example, if the error text is:
	//
	//     unable to dial www.google.com caused by: i/o timeout
	//
	// ErrorClean might return:
	//
	//     unable to dial %v caused by: %v
	//
	// This can be useful when performing analytics on the error.
	ErrorClean() string

	// MultiLinePrinter implements the interface golog.MultiLine
	MultiLinePrinter() func(buf *bytes.Buffer) bool

	// Op attaches a hint of the operation triggers this Error. Many error types
	// returned by net and os package have Op pre-filled.
	Op(op string) Error

	// With attaches arbitrary field to the error. keys will be normalized as
	// underscore_divided_words, so all characters except letters and numbers will
	// be replaced with underscores, and all letters will be lowercased.
	With(key string, value interface{}) Error

	// RootCause returns the bottom-most cause of this Error. If the Error
	// resulted from wrapping a plain error, the wrapped error will be returned as
	// the cause.
	RootCause() error
}

type structured struct {
	id        uint64
	hiddenID  string
	data      context.Map
	context   context.Map
	wrapped   error
	cause     Error
	callStack stack.CallStack
}

// New creates an Error with supplied description and format arguments to the
// description. If any of the arguments is an error, we use that as the cause.
func New(desc string, args ...interface{}) Error {
	return NewOffset(1, desc, args...)
}

// NewOffset is like New but offsets the stack by the given offset. This is
// useful for utilities like golog that may create errors on behalf of others.
func NewOffset(offset int, desc string, args ...interface{}) Error {
	var cause error
	for _, arg := range args {
		err, isError := arg.(error)
		if isError {
			cause = err
			break
		}
	}
	e := buildError(desc, fmt.Sprintf(desc, args...), nil, Wrap(cause))
	e.attachStack(2 + offset)
	return e
}

// Wrap creates an Error based on the information in an error instance.  It
// returns nil if the error passed in is nil, so we can simply call
// errors.Wrap(s.l.Close()) regardless there's an error or not. If the error is
// already wrapped, it is returned as is.
func Wrap(err error) Error {
	return wrapSkipFrames(err, 1)
}

// Fill implements the method from the context.Contextual interface.
func (e *structured) Fill(m context.Map) {
	if e != nil {
		if e.cause != nil {
			// Include data from cause, which supercedes context
			e.cause.Fill(m)
		}
		// Include the context, which supercedes the cause
		for key, value := range e.context {
			m[key] = value
		}
		// Now include the error's data, which supercedes everything
		for key, value := range e.data {
			m[key] = value
		}
	}
}

func (e *structured) Op(op string) Error {
	e.data["error_op"] = op
	return e
}

func (e *structured) With(key string, value interface{}) Error {
	parts := strings.FieldsFunc(key, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	})
	k := strings.ToLower(strings.Join(parts, "_"))
	if k == "error" || k == "error_op" {
		// Never overwrite these
		return e
	}
	switch actual := value.(type) {
	case string, int, bool, time.Time:
		e.data[k] = actual
	default:
		e.data[k] = fmt.Sprint(actual)
	}
	return e
}

func (e *structured) RootCause() error {
	if e.cause == nil {
		if e.wrapped != nil {
			return e.wrapped
		}
		return e
	}
	return e.cause.RootCause()
}

func (e *structured) ErrorClean() string {
	return e.data["error"].(string)
}

// Error satisfies the error interface
func (e *structured) Error() string {
	return e.data["error_text"].(string) + e.hiddenID
}

func (e *structured) MultiLinePrinter() func(buf *bytes.Buffer) bool {
	first := true
	indent := false
	err := e
	stackPosition := 0
	switchedCause := false
	return func(buf *bytes.Buffer) bool {
		if indent {
			buf.WriteString("  ")
		}
		if first {
			buf.WriteString(e.Error())
			first = false
			indent = true
			return true
		}
		if switchedCause {
			fmt.Fprintf(buf, "Caused by: %v", err)
			if err.callStack != nil && len(err.callStack) > 0 {
				switchedCause = false
				indent = true
				return true
			}
			if err.cause == nil {
				return false
			}
			err = err.cause.(*structured)
			return true
		}
		if stackPosition < len(err.callStack) {
			buf.WriteString("at ")
			call := err.callStack[stackPosition]
			fmt.Fprintf(buf, "%+n (%s:%d)", call, call, call)
			stackPosition++
		}
		if stackPosition >= len(err.callStack) {
			switch cause := err.cause.(type) {
			case *structured:
				err = cause
				indent = false
				stackPosition = 0
				switchedCause = true
			default:
				return false
			}
		}
		return err != nil
	}
}

func wrapSkipFrames(err error, skip int) Error {
	if err == nil {
		return nil
	}

	// Look for *structureds
	if e, ok := err.(*structured); ok {
		return e
	}

	var cause Error
	// Look for hidden *structureds
	hiddenIDs, err2 := hidden.Extract(err.Error())
	if err2 == nil && len(hiddenIDs) > 0 {
		// Take the first hidden ID as our cause
		cause = get(hiddenIDs[0])
	}

	// Create a new *structured
	return buildError("", "", err, cause)
}

func (e *structured) attachStack(skip int) {
	call := stack.Caller(skip)
	e.callStack = stack.Trace().TrimBelow(call)
	e.data["error_location"] = fmt.Sprintf("%+n (%s:%d)", call, call, call)
}

func buildError(desc string, fullText string, wrapped error, cause Error) *structured {
	e := &structured{
		data: make(context.Map),
		// We capture the current context to allow it to propagate to higher layers.
		context: ops.AsMap(nil, false),
		wrapped: wrapped,
		cause:   cause,
	}
	e.save()

	errorType := "errors.Error"
	if wrapped != nil {
		op, goType, wrappedDesc, extra := parseError(wrapped)
		if desc == "" {
			desc = wrappedDesc
		}
		e.Op(op)
		errorType = goType
		if extra != nil {
			for key, value := range extra {
				e.data[key] = value
			}
		}
	}

	cleanedDesc := hidden.Clean(desc)
	e.data["error"] = cleanedDesc
	if fullText != "" {
		e.data["error_text"] = hidden.Clean(fullText)
	} else {
		e.data["error_text"] = cleanedDesc
	}
	e.data["error_type"] = errorType

	return e
}

func parseError(err error) (op string, goType string, desc string, extra map[string]string) {
	extra = make(map[string]string)

	// interfaces
	if _, ok := err.(net.Error); ok {
		if opError, ok := err.(*net.OpError); ok {
			op = opError.Op
			if opError.Source != nil {
				extra["remote_addr"] = opError.Source.String()
			}
			if opError.Addr != nil {
				extra["local_addr"] = opError.Addr.String()
			}
			extra["network"] = opError.Net
			err = opError.Err
		}
		switch actual := err.(type) {
		case *net.AddrError:
			goType = "net.AddrError"
			desc = actual.Err
			extra["addr"] = actual.Addr
		case *net.DNSError:
			goType = "net.DNSError"
			desc = actual.Err
			extra["domain"] = actual.Name
			if actual.Server != "" {
				extra["dns_server"] = actual.Server
			}
		case *net.InvalidAddrError:
			goType = "net.InvalidAddrError"
			desc = actual.Error()
		case *net.ParseError:
			goType = "net.ParseError"
			desc = "invalid " + actual.Type
			extra["text_to_parse"] = actual.Text
		case net.UnknownNetworkError:
			goType = "net.UnknownNetworkError"
			desc = "unknown network"
		case syscall.Errno:
			goType = "syscall.Errno"
			desc = actual.Error()
		case *url.Error:
			goType = "url.Error"
			desc = actual.Err.Error()
			op = actual.Op
		default:
			goType = reflect.TypeOf(err).String()
			desc = err.Error()
		}
		return
	}
	if _, ok := err.(runtime.Error); ok {
		desc = err.Error()
		switch err.(type) {
		case *runtime.TypeAssertionError:
			goType = "runtime.TypeAssertionError"
		default:
			goType = reflect.TypeOf(err).String()
		}
		return
	}

	// structs
	switch actual := err.(type) {
	case *http.ProtocolError:
		desc = actual.ErrorString
		if name, ok := httpProtocolErrors[err]; ok {
			goType = name
		} else {
			goType = "http.ProtocolError"
		}
	case url.EscapeError, *url.EscapeError:
		goType = "url.EscapeError"
		desc = "invalid URL escape"
	case url.InvalidHostError, *url.InvalidHostError:
		goType = "url.InvalidHostError"
		desc = "invalid character in host name"
	case *textproto.Error:
		goType = "textproto.Error"
		desc = actual.Error()
	case textproto.ProtocolError, *textproto.ProtocolError:
		goType = "textproto.ProtocolError"
		desc = actual.Error()

	case tls.RecordHeaderError:
		goType = "tls.RecordHeaderError"
		desc = actual.Msg
		extra["header"] = hex.EncodeToString(actual.RecordHeader[:])
	case x509.CertificateInvalidError:
		goType = "x509.CertificateInvalidError"
		desc = actual.Error()
	case x509.ConstraintViolationError:
		goType = "x509.ConstraintViolationError"
		desc = actual.Error()
	case x509.HostnameError:
		goType = "x509.HostnameError"
		desc = actual.Error()
		extra["host"] = actual.Host
	case x509.InsecureAlgorithmError:
		goType = "x509.InsecureAlgorithmError"
		desc = actual.Error()
	case x509.SystemRootsError:
		goType = "x509.SystemRootsError"
		desc = actual.Error()
	case x509.UnhandledCriticalExtension:
		goType = "x509.UnhandledCriticalExtension"
		desc = actual.Error()
	case x509.UnknownAuthorityError:
		goType = "x509.UnknownAuthorityError"
		desc = actual.Error()
	case hex.InvalidByteError:
		goType = "hex.InvalidByteError"
		desc = "invalid byte"
	case *json.InvalidUTF8Error:
		goType = "json.InvalidUTF8Error"
		desc = "invalid UTF-8 in string"
	case *json.InvalidUnmarshalError:
		goType = "json.InvalidUnmarshalError"
		desc = actual.Error()
	case *json.MarshalerError:
		goType = "json.MarshalerError"
		desc = actual.Error()
	case *json.SyntaxError:
		goType = "json.SyntaxError"
		desc = actual.Error()
	case *json.UnmarshalFieldError:
		goType = "json.UnmarshalFieldError"
		desc = actual.Error()
	case *json.UnmarshalTypeError:
		goType = "json.UnmarshalTypeError"
		desc = actual.Error()
	case *json.UnsupportedTypeError:
		goType = "json.UnsupportedTypeError"
		desc = actual.Error()
	case *json.UnsupportedValueError:
		goType = "json.UnsupportedValueError"
		desc = actual.Error()

	case *os.LinkError:
		goType = "os.LinkError"
		desc = actual.Error()
	case *os.PathError:
		goType = "os.PathError"
		op = actual.Op
		desc = actual.Err.Error()
	case *os.SyscallError:
		goType = "os.SyscallError"
		op = actual.Syscall
		desc = actual.Err.Error()
	case *exec.Error:
		goType = "exec.Error"
		desc = actual.Err.Error()
	case *exec.ExitError:
		goType = "exec.ExitError"
		desc = actual.Error()
		// TODO: limit the length
		extra["stderr"] = string(actual.Stderr)
	case *strconv.NumError:
		goType = "strconv.NumError"
		desc = actual.Err.Error()
		extra["function"] = actual.Func
	case *time.ParseError:
		goType = "time.ParseError"
		desc = actual.Message
	default:
		desc = err.Error()
		if t, ok := miscErrors[err]; ok {
			goType = t
			return
		}
		goType = reflect.TypeOf(err).String()
	}
	return
}

var httpProtocolErrors = map[error]string{
	http.ErrHeaderTooLong:        "http.ErrHeaderTooLong",
	http.ErrShortBody:            "http.ErrShortBody",
	http.ErrNotSupported:         "http.ErrNotSupported",
	http.ErrUnexpectedTrailer:    "http.ErrUnexpectedTrailer",
	http.ErrMissingContentLength: "http.ErrMissingContentLength",
	http.ErrNotMultipart:         "http.ErrNotMultipart",
	http.ErrMissingBoundary:      "http.ErrMissingBoundary",
}

var miscErrors = map[error]string{
	bufio.ErrInvalidUnreadByte: "bufio.ErrInvalidUnreadByte",
	bufio.ErrInvalidUnreadRune: "bufio.ErrInvalidUnreadRune",
	bufio.ErrBufferFull:        "bufio.ErrBufferFull",
	bufio.ErrNegativeCount:     "bufio.ErrNegativeCount",
	bufio.ErrTooLong:           "bufio.ErrTooLong",
	bufio.ErrNegativeAdvance:   "bufio.ErrNegativeAdvance",
	bufio.ErrAdvanceTooFar:     "bufio.ErrAdvanceTooFar",
	bufio.ErrFinalToken:        "bufio.ErrFinalToken",

	http.ErrWriteAfterFlush:    "http.ErrWriteAfterFlush",
	http.ErrBodyNotAllowed:     "http.ErrBodyNotAllowed",
	http.ErrHijacked:           "http.ErrHijacked",
	http.ErrContentLength:      "http.ErrContentLength",
	http.ErrBodyReadAfterClose: "http.ErrBodyReadAfterClose",
	http.ErrHandlerTimeout:     "http.ErrHandlerTimeout",
	http.ErrLineTooLong:        "http.ErrLineTooLong",
	http.ErrMissingFile:        "http.ErrMissingFile",
	http.ErrNoCookie:           "http.ErrNoCookie",
	http.ErrNoLocation:         "http.ErrNoLocation",
	http.ErrSkipAltProtocol:    "http.ErrSkipAltProtocol",

	io.EOF:              "io.EOF",
	io.ErrClosedPipe:    "io.ErrClosedPipe",
	io.ErrNoProgress:    "io.ErrNoProgress",
	io.ErrShortBuffer:   "io.ErrShortBuffer",
	io.ErrShortWrite:    "io.ErrShortWrite",
	io.ErrUnexpectedEOF: "io.ErrUnexpectedEOF",

	os.ErrInvalid:    "os.ErrInvalid",
	os.ErrPermission: "os.ErrPermission",
	os.ErrExist:      "os.ErrExist",
	os.ErrNotExist:   "os.ErrNotExist",

	exec.ErrNotFound: "exec.ErrNotFound",

	x509.ErrUnsupportedAlgorithm: "x509.ErrUnsupportedAlgorithm",
	x509.IncorrectPasswordError:  "x509.IncorrectPasswordError",

	hex.ErrLength: "hex.ErrLength",
}
