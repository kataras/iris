package accesslog

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
)

// FieldExtractor extracts a log's entry key with its corresponding value.
// If key or value is empty then this field is not printed.
type FieldExtractor func(*context.Context) (string, interface{})

type (
	// Clock is an interface which contains a single `Now` method.
	// It can be used to set a static timer on end to end testing.
	// See `AccessLog.Clock` field.
	Clock     interface{ Now() time.Time }
	clockFunc func() time.Time
)

// Now completes the `Clock` interface.
func (c clockFunc) Now() time.Time {
	return c()
}

var (
	// UTC returns time with UTC based location.
	UTC = clockFunc(func() time.Time { return time.Now().UTC() })
	// TClock accepts a static time.Time to use as
	// accesslog's Now method on current log fired timestamp.
	// Useful for testing.
	TClock = func(t time.Time) clockFunc { return func() time.Time { return t } }
)

// AccessLog is a middleware which prints information
// incoming HTTP requests.
//
// Sample access log line:
// 2020-08-22 00:44:20|1ms|POST|/read_body||200|{"id":10,"name":"Tim","age":22}|{"message":"OK"}|
//
// Look `New`, `File` package-level functions
// and its `Handler` method to learn more.
type AccessLog struct {
	mu sync.Mutex // ensures atomic writes.
	// If not nil then it overrides the Application's Logger.
	// Useful to write to a file.
	// If multiple output required, then define an `io.MultiWriter`.
	// See `SetOutput` and `AddOutput` methods too.
	Writer io.Writer
	// If enabled, it locks the underline Writer.
	// It should be turned off if the given `Writer` is already protected with a locker.
	// It should be enabled when you don't know if the writer locks itself
	// or when the writer is os.Stdout/os.Stderr and e.t.c.
	//
	// Defaults to false,
	// as the default Iris Application's Logger is protected with mutex.
	LockWriter bool

	// If not empty then each one of them is called on `Close` method.
	Closers []io.Closer

	// If not empty then overrides the time.Now to this custom clocker's `Now` method,
	// useful for testing (see `TClock`) and
	// on platforms that its internal clock is not compatible by default (advanced case) and
	// to change the time location (e.g. `UTC`).
	//
	// This field is used to set the time the log fired.
	// By default the middleware is using the local time, however
	// can be changed to `UTC` too.
	//
	// Do NOT touch this field if you don't know what you're doing.
	Clock Clock

	// If true then the middleware will fire the logs in a separate
	// go routine, making the request to finish first.
	// The log will be printed based on a copy of the Request's Context instead.
	//
	// Defaults to false.
	Async bool
	// If not empty then it overrides the Application's configuration's TimeFormat field.
	TimeFormat string
	// Force minify request and response contents.
	BodyMinify bool
	// Enable request body logging.
	// Note that, if this is true then it modifies the underline request's body type.
	RequestBody bool
	// Enable response body logging.
	// Note that, if this is true then it uses a response recorder.
	ResponseBody bool

	// Map log fields with custom request values.
	// See `Log.Fields`.
	Fields []FieldExtractor
	// Note: We could use a map but that way we lose the
	// order of registration so use a slice and
	// take the field key from the extractor itself.
	formatter Formatter
}

// New returns a new AccessLog value with the default values.
// Writes to the Application's logger. Output be modified through its `SetOutput` method.
// Register by its `Handler` method.
// See `File` package-level function too.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/logging/request-logger/accesslog
func New() *AccessLog {
	return &AccessLog{
		Async:        false,
		LockWriter:   false,
		BodyMinify:   true,
		RequestBody:  true,
		ResponseBody: true,
		TimeFormat:   "2006-01-02 15:04:05",
	}
}

// File returns a new AccessLog value with the given "path"
// as the log's output file destination.
// Register by its `Handler` method.
//
// A call of its `Close` method to unlock the underline
// file is required on program termination.
//
// It panics on error.
func File(path string) *AccessLog {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	ac := New()
	ac.SetOutput(f)
	return ac
}

// Write writes to the log destination.
// It completes the io.Writer interface.
// Safe for concurrent use.
func (ac *AccessLog) Write(p []byte) (int, error) {
	ac.mu.Lock()
	n, err := ac.Writer.Write(p)
	ac.mu.Unlock()
	return n, err
}

// SetOutput sets the log's output destination. Accepts one or more io.Writer values.
// Also, if a writer is a Closer, then it is automatically appended to the Closers.
// Call it before `SetFormatter` and `Handler` methods.
func (ac *AccessLog) SetOutput(writers ...io.Writer) *AccessLog {
	for _, w := range writers {
		if closer, ok := w.(io.Closer); ok {
			ac.Closers = append(ac.Closers, closer)
		}
	}

	switch len(writers) {
	case 0:
	case 1:
		ac.Writer = writers[0]
	default:
		ac.Writer = io.MultiWriter(writers...)
	}

	return ac
}

// AddOutput appends an io.Writer value to the existing writer.
// Call it before `SetFormatter` and `Handler` methods.
func (ac *AccessLog) AddOutput(writers ...io.Writer) *AccessLog {
	if ac.Writer != nil { // prepend if one exists.
		writers = append([]io.Writer{ac.Writer}, writers...)
	}

	return ac.SetOutput(writers...)
}

// SetFormatter sets a custom formatter to print the logs.
// Any custom output writers should be
// already registered before calling this method.
// Returns this AccessLog instance.
//
// Usage:
// ac.SetFormatter(&accesslog.JSON{Indent: "  "})
func (ac *AccessLog) SetFormatter(f Formatter) *AccessLog {
	f.SetOutput(ac.Writer) // inject the writer here.
	ac.formatter = f
	return ac
}

// AddField maps a log entry with a value extracted by the Request Context.
// Not safe for concurrent use. Call it before `Handler` method.
func (ac *AccessLog) AddField(extractors ...FieldExtractor) *AccessLog {
	ac.Fields = append(ac.Fields, extractors...)
	return ac
}

// Close calls each registered Closer's Close method.
// Exits when all close methods have been executed.
func (ac *AccessLog) Close() (err error) {
	for _, closer := range ac.Closers {
		cErr := closer.Close()
		if cErr != nil {
			if err == nil {
				err = cErr
			} else {
				err = fmt.Errorf("%v, %v", err, cErr)
			}
		}
	}

	return
}

// Handler prints request information to the output destination.
// It is the main method of the AccessLog middleware.
//
// Usage:
// ac := New() or File("access.log")
// defer ac.Close()
// app.UseRouter(ac.Handler)
func (ac *AccessLog) Handler(ctx *context.Context) {
	var (
		startTime = time.Now()
		// Store some values, as future handler chain
		// can modify those (note: we could clone the request or context object too).
		method = ctx.Method()
		path   = ctx.Path()
	)

	// Enable response recording.
	if ac.ResponseBody {
		ctx.Record()
	}
	// Enable reading the request body
	// multiple times (route handler and this middleware).
	if ac.RequestBody {
		ctx.RecordBody()
	}

	// Proceed to the handlers chain.
	ctx.Next()

	latency := time.Since(startTime)
	if ac.Async {
		ctxCopy := ctx.Clone()
		go ac.after(ctxCopy, latency, method, path)
	} else {
		// wait to finish before proceed with response end.
		ac.after(ctx, latency, method, path)
	}
}

func (ac *AccessLog) after(ctx *context.Context, lat time.Duration, method, path string) {
	var (
		// request and response data or error reading them.
		requestBody  string
		responseBody string
	)

	// any error handler stored ( ctx.SetErr or StopWith(Plain)Error )
	if ctxErr := ctx.GetErr(); ctxErr != nil {
		requestBody = fmt.Sprintf("error(%s)", ctxErr.Error())
	} else if ac.RequestBody {
		requestData, err := ctx.GetBody()
		if err != nil {
			requestBody = fmt.Sprintf("error(%s)", ctxErr.Error())
		} else {
			if ac.BodyMinify {
				if minified, err := ctx.Application().Minifier().Bytes(ctx.GetContentTypeRequested(), requestData); err == nil {
					requestBody = string(minified)
				}
			}
			/* Some content types, like the text/plain,
			   no need minifier. Should be printed with spaces and \n. */
			if requestBody == "" {
				requestBody = string(requestData)
			}
		}
	}

	if ac.RequestBody {
		if responseData := ctx.Recorder().Body(); len(responseData) > 0 {
			if ac.BodyMinify {
				if minified, err := ctx.Application().Minifier().Bytes(ctx.GetContentType(), ctx.Recorder().Body()); err == nil {
					responseBody = string(minified)
				}

			}

			if responseBody == "" {
				responseBody = string(responseData)
			}
		}
	}

	// Grab any custom fields.
	var fields []memstore.Entry

	if n := len(ac.Fields); n > 0 {
		fields = make([]memstore.Entry, 0, n)

		for _, extract := range ac.Fields {
			key, value := extract(ctx)
			if key == "" || value == nil {
				continue
			}

			fields = append(fields, memstore.Entry{Key: key, ValueRaw: value})
		}
	}

	timeFormat := ac.TimeFormat
	if timeFormat == "" {
		timeFormat = ctx.Application().ConfigurationReadOnly().GetTimeFormat()
	}

	if err := ac.Print(ctx,
		// latency between begin and finish of the handlers chain.
		lat,
		timeFormat,
		// response code.
		ctx.GetStatusCode(),
		// original request's method and path.
		method, path,
		requestBody, responseBody,
		ctx.Params(), ctx.URLParamsSorted(), fields,
	); err != nil {
		ctx.Application().Logger().Errorf("accesslog: %v", err)
	}
}

// Print writes a log manually.
// The `Handler` method calls it.
func (ac *AccessLog) Print(ctx *context.Context, latency time.Duration, timeFormat string, code int, method, path, reqBody, respBody string, params *context.RequestParams, query []memstore.StringEntry, fields []memstore.Entry) error {
	var now time.Time

	if ac.Clock != nil {
		now = ac.Clock.Now()
	} else {
		now = time.Now()
	}

	if f := ac.formatter; f != nil {
		log := &Log{
			Logger:     ac,
			Now:        now,
			TimeFormat: timeFormat,
			Timestamp:  now.Unix(),
			Latency:    latency,
			Method:     method,
			Path:       path,
			Code:       code,
			Query:      query,
			PathParams: params.Store,
			Fields:     fields,
			Request:    reqBody,
			Response:   respBody,
			Ctx:        ctx, // ctx should only be used here, it may be nil on testing.
		}

		if err := f.Format(log); err != nil {
			return err
		}

		// OK, it's handled, exit now.
		return nil
	}

	// url parameters, path parameters and custom fields separated by space,
	// key=value key2=value2.
	requestValues := parseRequestValues(code, params, query, fields)

	useLocker := ac.LockWriter
	w := ac.Writer
	if w == nil {
		if ctx != nil {
			w = ctx.Application().Logger().Printer
		} else {
			w = os.Stdout
			useLocker = true // force lock.
		}
	}

	if useLocker {
		ac.mu.Lock()
	}
	// the number of separators are the same, in order to be easier
	// for 3rd-party programs to read the result log file.
	_, err := fmt.Fprintf(w, "%s|%s|%s|%s|%s|%d|%s|%s|\n",
		now.Format(timeFormat),
		latency,
		method,
		path,
		requestValues,
		code,
		reqBody,
		respBody,
	)
	if useLocker {
		ac.mu.Unlock()
	}

	return err
}
