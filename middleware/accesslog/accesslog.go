package accesslog

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"sync"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
)

func init() {
	context.SetHandlerName("iris/middleware/accesslog.*", "iris.accesslog")
}

const (
	fieldsContextKey  = "iris.accesslog.request.fields"
	skipLogContextKey = "iris.accesslog.request.skip"
)

// GetFields returns the accesslog fields for this request.
// Returns a store which the caller can use to
// set/get/remove custom log fields. Use its `Set` method.
func GetFields(ctx iris.Context) (fields *Fields) {
	if v := ctx.Values().Get(fieldsContextKey); v != nil {
		fields = v.(*Fields)
	} else {
		fields = new(Fields)
		ctx.Values().Set(fieldsContextKey, fields)
	}

	return
}

// Skip called when a specific route should be skipped from the logging process.
// It's an easy to use alternative for iris.NewConditionalHandler.
func Skip(ctx iris.Context) {
	ctx.Values().Set(skipLogContextKey, struct{}{})
}

func shouldSkip(ctx iris.Context) bool {
	return ctx.Values().Get(skipLogContextKey) != nil
}

type (

	// Fields is a type alias for memstore.Store, used to set
	// more than one field at serve-time. Same as FieldExtractor.
	Fields = memstore.Store
	// FieldSetter sets one or more fields at once.
	FieldSetter func(*context.Context, *Fields)
)

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
	// The destination writer.
	// If multiple output required, then define an `io.MultiWriter`.
	// See `SetOutput` and `AddOutput` methods too.
	Writer io.Writer
	// If enabled, it locks the underline Writer.
	// It should be turned off if the given `Writer` is already protected with a locker.
	// It is enabled when writer is os.Stdout/os.Stderr.
	// You should manually set this field to true if you are not sure
	// whether the underline Writer is protected.
	//
	// Defaults to true on *os.File and *bytes.Buffer, otherwise false.
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
	// The time format for current time on log print.
	// Defaults to the Iris Application's TimeFormat.
	TimeFormat string

	// The actual number of bytes received and sent on the network (headers + body).
	// It is kind of "slow" operation as it uses the httputil to dumb request
	// and response to get the total amount of bytes (headers + body).
	BytesReceived bool
	BytesSent     bool
	// Note: We could calculate only the bodies, which is a fast operation if we already
	// have RequestBody and ResponseBody set to true but this is not an accurate measurement.

	// Force minify request and response contents.
	BodyMinify bool
	// Enable request body logging.
	// Note that, if this is true then it modifies the underline request's body type.
	RequestBody bool
	// Enable response body logging.
	// Note that, if this is true then it uses a response recorder.
	ResponseBody bool

	// Map log fields with custom request values.
	// See `AddFields` method.
	FieldSetters []FieldSetter
	// Note: We could use a map but that way we lose the
	// order of registration so use a slice and
	// take the field key from the extractor itself.
	formatter Formatter
	broker    *Broker
}

// New returns a new AccessLog value with the default values.
// Writes to the "w". Output be further modified through its `Set/AddOutput` methods.
// Register by its `Handler` method.
// See `File` package-level function too.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/logging/request-logger/accesslog
func New(w io.Writer) *AccessLog {
	ac := &AccessLog{
		BytesReceived: true,
		BytesSent:     true,
		BodyMinify:    true,
		RequestBody:   true,
		ResponseBody:  true,
	}

	if w == nil {
		w = os.Stdout
	}
	ac.SetOutput(w)

	return ac
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

	return New(f)
}

// Broker creates or returns the broker.
// Use its `NewListener` and `CloseListener`
// to listen and unlisten for incoming logs.
//
// Should be called before serve-time.
func (ac *AccessLog) Broker() *Broker {
	ac.mu.Lock()
	if ac.broker == nil {
		ac.broker = newBroker()
		// atomic.StoreUint32(&ac.brokerActive, 1)
	}
	ac.mu.Unlock()
	return ac.broker
}

// func (ac *AccessLog) isBrokerActive() bool { // see `Print` method.
// 	return atomic.LoadUint32(&ac.brokerActive) > 0
// }
// ^ No need, we declare that the Broker should be called
// before serve-time. Let's respect our comment
// and don't try to make it safe for write and read concurrent access.

// Write writes to the log destination.
// It completes the io.Writer interface.
// Safe for concurrent use.
func (ac *AccessLog) Write(p []byte) (int, error) {
	if ac.LockWriter {
		ac.mu.Lock()
	}
	n, err := ac.Writer.Write(p)
	if ac.LockWriter {
		ac.mu.Unlock()
	}
	return n, err
}

// SetOutput sets the log's output destination. Accepts one or more io.Writer values.
// Also, if a writer is a Closer, then it is automatically appended to the Closers.
// Call it before `SetFormatter` and `Handler` methods.
func (ac *AccessLog) SetOutput(writers ...io.Writer) *AccessLog {
	if len(writers) == 0 {
		return ac
	}

	lockWriter := false
	for _, w := range writers {
		if closer, ok := w.(io.Closer); ok {
			ac.Closers = append(ac.Closers, closer)
		}

		if !lockWriter {
			switch w.(type) {
			case *os.File, *bytes.Buffer: // force lock writer.
				lockWriter = true
			}
		}
	}
	ac.LockWriter = lockWriter

	if len(writers) == 1 {
		ac.Writer = writers[0]
	} else {
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
	if ac.Writer == nil {
		panic("accesslog: SetFormatter called with nil Writer")
	}

	f.SetOutput(ac.Writer) // inject the writer here.
	ac.formatter = f
	return ac
}

// AddFields maps one or more log entries with values extracted by the Request Context.
// You can also add fields per request handler, look the `GetFields` package-level function.
// Note that this method can override a key stored by a handler's fields.
func (ac *AccessLog) AddFields(setters ...FieldSetter) *AccessLog {
	ac.FieldSetters = append(ac.FieldSetters, setters...)
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

func (ac *AccessLog) shouldReadRequestBody() bool {
	return ac.RequestBody || ac.BytesReceived

}

func (ac *AccessLog) shouldReadResponseBody() bool {
	return ac.ResponseBody || ac.BytesSent
}

// Handler prints request information to the output destination.
// It is the main method of the AccessLog middleware.
//
// Usage:
// ac := New(io.Writer) or File("access.log")
// defer ac.Close()
// app.UseRouter(ac.Handler)
func (ac *AccessLog) Handler(ctx *context.Context) {
	if shouldSkip(ctx) { // usage: another middleware before that one disables logging.
		ctx.Next()
		return
	}

	var (
		startTime = time.Now()
		// Store some values, as future handler chain
		// can modify those (note: we could clone the request or context object too).
		method = ctx.Method()
		path   = ctx.Path()
	)

	// Enable response recording.
	if ac.shouldReadResponseBody() {
		ctx.Record()
	}
	// Enable reading the request body
	// multiple times (route handler and this middleware).
	if ac.shouldReadRequestBody() {
		ctx.RecordBody()
	}

	// Set the fields context value so they can be modified
	// on the following handlers chain. Same as `AddFields` but per-request.
	// ctx.Values().Set(fieldsContextKey, new(Fields))
	// No need ^ The GetFields will set it if it's missing.
	// So we initialize them whenever, and if, asked.

	// Proceed to the handlers chain.
	ctx.Next()

	if shouldSkip(ctx) { // normal flow, we can get the context by executing the handler first.
		return
	}

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
		requestBody   string
		responseBody  string
		bytesReceived int
		bytesSent     int
	)

	if ac.shouldReadRequestBody() {
		//	any error handler stored ( ctx.SetErr or StopWith(Plain)Error )
		if ctxErr := ctx.GetErr(); ctxErr != nil {
			// If there is an error here
			// we may need to NOT read the body for security reasons, e.g.
			// unauthorized user tries to send a malicious body.
			requestBody = fmt.Sprintf("error(%s)", ctxErr.Error())
		} else {
			requestData, err := ctx.GetBody()
			requestBodyLength := len(requestData)
			if err != nil && ac.RequestBody {
				requestBody = fmt.Sprintf("error(%s)", err)
			} else if requestBodyLength > 0 {
				if ac.RequestBody {
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

			if ac.BytesReceived {
				// Unfortunally the DumpRequest cannot read the body
				// length as expected (see postman's i/o values)
				// so we had to read the data length manually even if RequestBody/ResponseBody
				// are false, extra operation if they are enabled is to minify their log entry representation.

				b, _ := httputil.DumpRequest(ctx.Request(), false)
				bytesReceived = len(b) + requestBodyLength
			}
		}
	}

	if ac.shouldReadResponseBody() {
		responseData := ctx.Recorder().Body()
		responseBodyLength := len(responseData)
		if ac.ResponseBody && responseBodyLength > 0 {
			if ac.BodyMinify {
				if minified, err := ctx.Application().Minifier().Bytes(ctx.GetContentType(), responseData); err == nil {
					responseBody = string(minified)
				}
			}

			if responseBody == "" {
				responseBody = string(responseData)
			}
		}

		if ac.BytesSent {
			resp := ctx.Recorder().Result()
			b, _ := httputil.DumpResponse(resp, false)
			dateLengthProx := 38 /* it's actually ~37 */
			if resp.Header.Get("Date") != "" {
				dateLengthProx = 0 // dump response calculated it.
			}
			bytesSent = len(b) + responseBodyLength + dateLengthProx
		}
	}

	// Grab any custom fields.
	fields := GetFields(ctx)

	for _, setter := range ac.FieldSetters {
		setter(ctx, fields)
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
		bytesReceived, bytesSent,
		ctx.Params(), ctx.URLParamsSorted(), *fields,
	); err != nil {
		ctx.Application().Logger().Errorf("accesslog: %v", err)
	}
}

// Print writes a log manually.
// The `Handler` method calls it.
func (ac *AccessLog) Print(ctx *context.Context, latency time.Duration, timeFormat string, code int, method, path, reqBody, respBody string, bytesReceived, bytesSent int, params *context.RequestParams, query []memstore.StringEntry, fields []memstore.Entry) (err error) {
	var now time.Time

	if ac.Clock != nil {
		now = ac.Clock.Now()
	} else {
		now = time.Now()
	}

	if hasFormatter, hasBroker := ac.formatter != nil, ac.broker != nil; hasFormatter || hasBroker {
		log := &Log{
			Logger:        ac,
			Now:           now,
			TimeFormat:    timeFormat,
			Timestamp:     now.Unix(),
			Latency:       latency,
			Method:        method,
			Path:          path,
			Code:          code,
			Query:         query,
			PathParams:    params.Store,
			Fields:        fields,
			BytesReceived: bytesReceived,
			BytesSent:     bytesSent,
			Request:       reqBody,
			Response:      respBody,
			Ctx:           ctx, // ctx should only be used here, it may be nil on testing.
		}

		var handled bool
		if hasFormatter {
			handled, err = ac.formatter.Format(log)
			if err != nil {
				return
			}
		}

		if hasBroker { // after Format, it may want to customize the log's fields.
			ac.broker.notify(log)
		}

		if handled {
			return // OK, it's handled, exit now.
		}
	}

	// url parameters, path parameters and custom fields separated by space,
	// key=value key2=value2.
	requestValues := parseRequestValues(code, params, query, fields)

	// the number of separators are the same, in order to be easier
	// for 3rd-party programs to read the result log file.
	_, err = fmt.Fprintf(ac, "%s|%s|%s|%s|%s|%d|%s|%s|%s|%s|\n",
		now.Format(timeFormat),
		latency,
		method,
		path,
		requestValues,
		code,
		formatBytes(bytesReceived),
		formatBytes(bytesSent),
		reqBody,
		respBody,
	)

	return
}
