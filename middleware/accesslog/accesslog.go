package accesslog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
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
	// If not empty then each one of them is called on `Close` method.
	Closers []io.Closer

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
}

// New returns a new AccessLog value with the default values.
// Writes to the Application's logger.
// Register by its `Handler` method.
// See `File` package-level function too.
func New() *AccessLog {
	return &AccessLog{
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
func (ac *AccessLog) SetOutput(writers ...io.Writer) *AccessLog {
	for _, w := range writers {
		if closer, ok := w.(io.Closer); ok {
			ac.Closers = append(ac.Closers, closer)
		}
	}

	ac.Writer = io.MultiWriter(writers...)
	return ac
}

// AddOutput appends an io.Writer value to the existing writer.
func (ac *AccessLog) AddOutput(writers ...io.Writer) *AccessLog {
	if ac.Writer != nil { // prepend if one exists.
		writers = append([]io.Writer{ac.Writer}, writers...)
	}

	return ac.SetOutput(writers...)
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
	ac.after(ctx, latency, method, path)
}

func (ac *AccessLog) after(ctx *context.Context, lat time.Duration, method, path string) {
	var (
		code = ctx.GetStatusCode() // response status code
		// request and response data or error reading them.
		requestBody  string
		responseBody string

		// url parameters and path parameters separated by space,
		// key=value key2=value2.
		requestValues string
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

	var buf strings.Builder

	if !context.StatusCodeNotSuccessful(code) {
		// collect path parameters on a successful request-response only.
		ctx.Params().Visit(func(key, value string) {
			buf.WriteString(key)
			buf.WriteByte('=')
			buf.WriteString(value)
			buf.WriteByte(' ')
		})
	}

	for _, entry := range ctx.URLParamsSorted() {
		buf.WriteString(entry.Key)
		buf.WriteByte('=')
		buf.WriteString(entry.Value)
		buf.WriteByte(' ')
	}

	if n := buf.Len(); n > 1 {
		requestValues = buf.String()[0 : n-1] // remove last space.
	}

	timeFormat := ac.TimeFormat
	if timeFormat == "" {
		timeFormat = ctx.Application().ConfigurationReadOnly().GetTimeFormat()
	}

	w := ac.Writer
	if w == nil {
		w = ctx.Application().Logger().Printer
	}

	// the number of separators are the same, in order to be easier
	// for 3rd-party programs to read the result log file.
	fmt.Fprintf(w, "%s|%s|%s|%s|%s|%d|%s|%s|\n",
		time.Now().Format(timeFormat),
		lat,
		method,
		path,
		requestValues,
		code,
		requestBody,
		responseBody,
	)
}
