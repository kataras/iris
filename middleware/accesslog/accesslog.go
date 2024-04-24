package accesslog

import (
	"bufio"
	"bytes"
	stdContext "context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/host"
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
//
// To use with MVC: Register(accesslog.GetFields).
// DI Handlers: ConfigureContainer().RegisterDependency(accesslog.GetFields).
func GetFields(ctx *context.Context) (fields *Fields) {
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
func Skip(ctx *context.Context) {
	ctx.Values().Set(skipLogContextKey, struct{}{})
}

// SkipHandler same as `Skip` but it can be used
// as a middleware, it executes ctx.Next().
func SkipHandler(ctx *context.Context) {
	Skip(ctx)
	ctx.Next()
}

func shouldSkip(ctx *context.Context) bool {
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
// Default log format:
// Time|Latency|Code|Method|Path|IP|Path Params Query Fields|Bytes Received|Bytes Sent|Request|Response|
//
// Look `New`, `File` package-level functions
// and its `Handler` method to learn more.
// If the given writer is a buffered one,
// its contents are flushed automatically on Close call.
//
// A new AccessLog middleware MUST
// be created after a `New` function call.
type AccessLog struct {
	mu sync.RWMutex // ensures atomic writes.
	// The destination writer.
	// If multiple output required, then define an `io.MultiWriter`.
	// See `SetOutput` and `AddOutput` methods too.
	Writer io.Writer

	// If not empty then each one of them is called on `Close` method.
	// File type destinations are automatically added.
	Flushers []Flusher
	Closers  []io.Closer
	// Outputs that support the Truncate method.
	BufferTruncaters []BufferTruncater
	FileTruncaters   []FileTruncater

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
	// The delimiter between fields when logging with the default format.
	// See `SetFormatter` to customize the log even further.
	//
	// Defaults to '|'.
	Delim byte
	// The time format for current time on log print.
	// Set it to empty to inherit the Iris Application's TimeFormat.
	//
	// Defaults to "2006-01-02 15:04:05"
	TimeFormat string
	// A text that will appear in a blank value.
	// Applies to the default formatter on
	// IP, RequestBody and ResponseBody fields, if enabled, so far.
	//
	// Defaults to nil.
	Blank []byte
	// Round the latency based on the given duration, e.g. time.Second.
	//
	// Defaults to 0.
	LatencyRound time.Duration

	// IP displays the remote address.
	//
	// Defaults to true.
	IP bool
	// The number of bytes for the request body only.
	// Applied when BytesReceived is false.
	//
	// Defaults to true.
	BytesReceivedBody bool
	// The number of bytes for the response body only.
	// Applied when BytesSent is false.
	//
	// Defaults to true.
	BytesSentBody bool
	// The actual number of bytes received and sent on the network (headers + body).
	// It is kind of "slow" operation as it uses the httputil to dumb request
	// and response to get the total amount of bytes (headers + body).
	//
	// They override the BytesReceivedBody and BytesSentBody fields.
	// These two fields provide a more a acquirate measurement
	// than BytesReceivedBody and BytesSentBody however,
	// they are expensive operations, expect a slower execution.
	//
	// They both default to false.
	BytesReceived bool
	BytesSent     bool
	// Enable request body logging.
	// Note that, if this is true then it modifies the underline request's body type.
	//
	// Defaults to true.
	RequestBody bool
	// Enable response body logging.
	// Note that, if this is true then it uses a response recorder.
	//
	// Defaults to false.
	ResponseBody bool
	// Force minify request and response contents.
	//
	// Defaults to true.
	BodyMinify bool

	// KeepMultiLineError displays the Context's error as it's.
	// If set to false then it replaces all line characters with spaces.
	//
	// See `PanicLog` to customize recovered-from-panic errors even further.
	//
	// Defaults to true.
	KeepMultiLineError bool
	// What the logger should write to the output destination
	// when recovered from a panic.
	// Available options:
	// * LogHandler (default, logs the handler's file:line only)
	// * LogCallers (logs callers separated by line breaker)
	// * LogStack   (logs the debug stack)
	PanicLog PanicLog

	// Map log fields with custom request values.
	// See `AddFields` method.
	FieldSetters []FieldSetter
	// Note: We could use a map but that way we lose the
	// order of registration so use a slice and
	// take the field key from the extractor itself.
	formatter Formatter
	broker    *Broker

	// the log instance for custom formatters.
	logsPool *sync.Pool
	// the builder for the default format.
	bufPool *sync.Pool
	// remaining logs when Close is called, we wait for timeout (see CloseContext).
	remaining uint32
	// reports whether the logger is already closed, see `Close` & `CloseContext` methods.
	isClosed uint32
}

// PanicLog holds the type for the available panic log levels.
type PanicLog uint8

const (
	// LogHandler logs the handler's file:line that recovered from.
	LogHandler PanicLog = iota
	// LogCallers logs all callers separated by new lines.
	LogCallers
	// LogStack logs the whole debug stack.
	LogStack
)

const (
	defaultDelim      = '|'
	defaultTimeFormat = "2006-01-02 15:04:05"
	newLine           = '\n'
)

// New returns a new AccessLog value with the default values.
// Writes to the "w". Output can be further modified through its `Set/AddOutput` methods.
//
// Register by its `Handler` method.
// See `File` package-level function too.
//
// Examples:
// https://github.com/kataras/iris/tree/main/_examples/logging/request-logger/accesslog
// https://github.com/kataras/iris/tree/main/_examples/logging/request-logger/accesslog-template
// https://github.com/kataras/iris/tree/main/_examples/logging/request-logger/accesslog-broker
func New(w io.Writer) *AccessLog {
	ac := &AccessLog{
		Clock:              clockFunc(time.Now),
		Delim:              defaultDelim,
		TimeFormat:         defaultTimeFormat,
		Blank:              nil,
		LatencyRound:       0,
		Async:              false,
		IP:                 true,
		BytesReceived:      false,
		BytesSent:          false,
		BytesReceivedBody:  true,
		BytesSentBody:      true,
		RequestBody:        true,
		ResponseBody:       false,
		BodyMinify:         true,
		KeepMultiLineError: true,
		PanicLog:           LogHandler,
		logsPool: &sync.Pool{New: func() interface{} {
			return new(Log)
		}},
		bufPool: &sync.Pool{New: func() interface{} {
			return new(bytes.Buffer)
		}},
	}

	if w == nil {
		w = os.Stdout
	}
	ac.SetOutput(w)

	// workers := 20
	// listener := ac.Broker().NewListener()
	// for i := 0; i < workers; i++ {
	// 	go func() {
	// 		for log := range listener {
	// 			atomic.AddUint32(&ac.remaining, 1)
	// 			ac.handleLog(log)
	// 			atomic.AddUint32(&ac.remaining, ^uint32(0))
	// 		}
	// 	}()
	// }

	host.RegisterOnInterrupt(func() {
		ac.Close()
	})

	return ac
}

// File returns a new AccessLog value with the given "path"
// as the log's output file destination.
// The Writer is now a buffered file writer & reader.
// Register by its `Handler` method.
//
// A call of its `Close` method to unlock the underline
// file is required on program termination.
//
// It panics on error.
func File(path string) *AccessLog {
	f := mustOpenFile(path)
	return New(bufio.NewReadWriter(bufio.NewReader(f), bufio.NewWriter(f)))
}

// FileUnbuffered same as File but it does not buffer the data,
// it flushes the loggers contents as soon as possible.
func FileUnbuffered(path string) *AccessLog {
	f := mustOpenFile(path)
	return New(f)
}

func mustOpenFile(path string) *os.File {
	// Note: we add os.RDWR in order to be able to read from it,
	// some formatters (e.g. CSV) needs that.
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}

	return f
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
	}
	ac.mu.Unlock()

	return ac.broker
}

// SetOutput sets the log's output destination. Accepts one or more io.Writer values.
// Also, if a writer is a Closer, then it is automatically appended to the Closers.
// It's safe to used concurrently (experimental).
func (ac *AccessLog) SetOutput(writers ...io.Writer) *AccessLog {
	ac.setOutput(true, writers...)
	return ac
}

// AddOutput appends an io.Writer value to the existing writer.
// Call it before `SetFormatter` and `Handler` methods.
func (ac *AccessLog) AddOutput(writers ...io.Writer) *AccessLog {
	ac.setOutput(false, writers...)
	return ac
}

func (ac *AccessLog) setOutput(reset bool, writers ...io.Writer) {

	/*
	   Initial idea was to wait for remaining logs to be written
	   in the existing writer before resetting to the new one.
	   But, a faster approach would be to just write the logs
	   to the new writers instead. This can be done by:
	   1. copy all existing closers and flushers,
	   2. change the writer immediately
	   3. fire a goroutine which flushes and closes the old writers,
	       no locks required there because they are not used for concurrent writing
	       anymore. Errors there are ignored (we could collect them with sync errgroup
	       and wait for them before exit this Reset method, but we don't).
	*/

	if !reset {
		// prepend if one exists.
		ac.mu.Lock()
		if ac.Writer != nil {
			writers = append([]io.Writer{ac.Writer}, writers...)
		}
		ac.mu.Unlock()
	}

	switch len(writers) {
	case 0:
		return
	case 1:
		ac.mu.Lock()
		ac.Writer = writers[0]
		ac.mu.Unlock()
	default:
		multi := io.MultiWriter(writers...)
		ac.mu.Lock()
		ac.Writer = multi
		ac.mu.Unlock()
	}

	// NO need to check for a "hadWriter",
	// because it will always have a previous writer
	// on serve-time (the spot we care about performance),
	// so if it set by New, on build-time, we don't rly care about some read locks slowdown.
	ac.mu.RLock()
	n := len(ac.Flushers)
	ac.mu.RUnlock()

	flushers := make([]Flusher, n)
	if n > 0 {
		ac.mu.Lock()
		copy(flushers, ac.Flushers)
		ac.mu.Unlock()
	}

	ac.mu.RLock()
	n = len(ac.Closers)
	ac.mu.RUnlock()

	closers := make([]io.Closer, n)
	if n > 0 {
		ac.mu.Lock()
		copy(closers, ac.Closers)
		ac.mu.Unlock()
	}

	if reset {
		// Reset previous flushers and closers,
		// so any middle request can't flush to the old ones.
		// Note that, because we don't lock the whole operation,
		// there is a chance of Flush while we are doing this,
		// not by the middleware (unless panic, but again, the data are written
		// to the new writer, they are not lost, just not flushed immediately),
		// an outsider may call it, and if it does
		// then it is its responsibility to lock between manual Flush calls and
		// SetOutput ones. This is done to be able
		// to serve requests fast even on Async == false
		// while SetOutput is called at serve-time, if we didn't care about it
		// we could lock the whole operation which would make the
		// log writers to wait and be done with this.
		ac.mu.Lock()
		ac.Flushers = ac.Flushers[0:0]
		ac.Closers = ac.Closers[0:0]
		ac.BufferTruncaters = ac.BufferTruncaters[0:0]
		ac.FileTruncaters = ac.FileTruncaters[0:0]
		ac.mu.Unlock()
	}

	// Store the new flushers, closers and truncaters...
	for _, w := range writers {
		if flusher, ok := w.(Flusher); ok {
			ac.mu.Lock()
			ac.Flushers = append(ac.Flushers, flusher)
			ac.mu.Unlock()
		}

		if closer, ok := w.(io.Closer); ok {
			ac.mu.Lock()
			ac.Closers = append(ac.Closers, closer)
			ac.mu.Unlock()
		}

		if truncater, ok := w.(BufferTruncater); ok {
			ac.mu.Lock()
			ac.BufferTruncaters = append(ac.BufferTruncaters, truncater)
			ac.mu.Unlock()
		}

		if truncater, ok := w.(FileTruncater); ok {
			ac.mu.Lock()
			ac.FileTruncaters = append(ac.FileTruncaters, truncater)
			ac.mu.Unlock()
		}
	}

	if reset {
		// And finally, wait before exit this method
		// until previous writer's closers and flush finish.
		for _, flusher := range flushers {
			if flusher != nil {
				flusher.Flush()
			}
		}
		for _, closer := range closers {
			if closer != nil {
				// cannot close os.Stdout/os.Stderr
				if closer == os.Stdout || closer == os.Stderr {
					continue
				}
				closer.Close()
			}
		}
	}
}

// Close terminates any broker listeners,
// waits for any remaining logs up to 10 seconds
// (see `CloseContext` to set custom deadline),
// flushes any formatter and any buffered data to the underline writer
// and finally closes any registered closers (files are automatically added as Closer).
//
// After Close is called the AccessLog is not accessible.
func (ac *AccessLog) Close() (err error) {
	ctx, cancelFunc := stdContext.WithTimeout(stdContext.Background(), 10*time.Second)
	defer cancelFunc()

	return ac.CloseContext(ctx)
}

// CloseContext same as `Close` but waits until given "ctx" is done.
func (ac *AccessLog) CloseContext(ctx stdContext.Context) (err error) {
	if !atomic.CompareAndSwapUint32(&ac.isClosed, 0, 1) {
		return
	}

	if ac.broker != nil {
		ac.broker.close <- struct{}{}
	}

	if ac.Async {
		ac.waitRemaining(ctx)
	}

	if fErr := ac.Flush(); fErr != nil {
		if err == nil {
			err = fErr
		} else {
			err = fmt.Errorf("%v, %v", err, fErr)
		}
	}

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

func (ac *AccessLog) waitRemaining(ctx stdContext.Context) {
	if n := atomic.LoadUint32(&ac.remaining); n == 0 {
		return
	}

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if atomic.LoadUint32(&ac.remaining) == 0 {
				return
			}
		}
	}
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
func (ac *AccessLog) Write(p []byte) (n int, err error) {
	if ac.Async {
		if atomic.LoadUint32(&ac.isClosed) > 0 {
			return 0, io.ErrClosedPipe
		}
	}

	ac.mu.Lock()
	n, err = ac.Writer.Write(p)
	ac.mu.Unlock()
	return
}

// Flush writes any buffered data to the underlying Fluser Writer.
// Flush is called automatically on Close.
func (ac *AccessLog) Flush() (err error) {
	ac.mu.Lock()
	for _, f := range ac.Flushers {
		fErr := f.Flush()
		if fErr != nil {
			if err == nil {
				err = fErr
			} else {
				err = fmt.Errorf("%v, %v", err, fErr)
			}
		}
	}
	ac.mu.Unlock()
	return
}

// Truncate if the output is a buffer, then
// it discards all but the first n unread bytes.
// See `TruncateFile` for a file size.
//
// It panics if n is negative or greater than the length of the buffer.
func (ac *AccessLog) Truncate(n int) {
	ac.mu.Lock() // Lock as we do with all write operations.
	for _, truncater := range ac.BufferTruncaters {
		truncater.Truncate(n)
	}
	ac.mu.Unlock()
}

// TruncateFile flushes any buffered contents
// and changes the size of the internal file destination, directly.
// It does not change the I/O offset.
//
// Note that `TruncateFile` calls the `Truncate(int(size))` automatically
// in order to clear any buffered contents (if the file was wrapped by a buffer)
// before truncating the file itself.
//
// Usage, clear a file:
// err := TruncateFile(0)
func (ac *AccessLog) TruncateFile(size int64) (err error) {
	ac.Truncate(int(size))

	ac.mu.Lock()
	for _, truncater := range ac.FileTruncaters {
		tErr := truncater.Truncate(size)
		if tErr != nil {
			if err == nil {
				err = tErr
			} else {
				err = fmt.Errorf("%v, %v", err, tErr)
			}
		}
	}
	ac.mu.Unlock()

	return err
}

// SetFormatter sets a custom formatter to print the logs.
// Any custom output writers should be
// already registered before calling this method.
// Returns this AccessLog instance.
//
// Usage:
// ac.SetFormatter(&accesslog.JSON{Indent: "    "})
func (ac *AccessLog) SetFormatter(f Formatter) *AccessLog {
	if ac.Writer == nil {
		panic("accesslog: SetFormatter called with nil Writer")
	}

	if f == nil {
		return ac
	}

	if flusher, ok := ac.formatter.(Flusher); ok {
		// PREPEND formatter flushes, they should run before destination's ones.
		ac.Flushers = append([]Flusher{flusher}, ac.Flushers...)
	}

	// Inject the writer (AccessLog) here, the writer
	// is protected with mutex.
	f.SetOutput(ac)

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

func (ac *AccessLog) shouldReadRequestBody() bool {
	return ac.RequestBody || ac.BytesReceived || ac.BytesReceivedBody

}

func (ac *AccessLog) shouldReadResponseBody() bool {
	return ac.ResponseBody || ac.BytesSent /* || ac.BytesSentBody this can be measured by the default writer's Written() */
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
		ctx.RecordRequestBody(true)
	}

	// Set the fields context value so they can be modified
	// on the following handlers chain. Same as `AddFields` but per-request.
	// ctx.Values().Set(fieldsContextKey, new(Fields))
	// No need ^ The GetFields will set it if it's missing.
	// So we initialize them whenever, and if, asked.

	// Proceed to the handlers chain.
	currentIndex := ctx.HandlerIndex(-1)
	ctx.Next()
	if context.StatusCodeNotSuccessful(ctx.GetStatusCode()) {
		_, wasRecovered := ctx.IsRecovered()
		// The ctx.HandlerName is still accesslog because
		// on end of router filters the router resets
		// the handler index, same for errors.
		// So, as a special case, if it's a failure status code
		// call FireErorrCode manually instead of wait
		// to be called on EndRequest (which is, correctly, called on end of everything
		// so we don't have chance to record its body by default).
		//
		// Note: this however will call the error handler twice
		// if the end-developer registered that using `UseError` instead of `UseRouter`,
		// there is a way to fix that too: by checking the handler index diff:
		if currentIndex == ctx.HandlerIndex(-1) || wasRecovered {
			// if handler index before and after ctx.Next
			// is the same, then it means we are in `UseRouter`
			// and on error handler.
			ctx.Application().FireErrorCode(ctx)
		}
	}

	if shouldSkip(ctx) { // normal flow, we can get the context by executing the handler first.
		return
	}

	latency := time.Since(startTime).Round(ac.LatencyRound)

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
		bytesReceived int // total or body, depends on the configuration.
		bytesSent     int
	)

	if ac.shouldReadRequestBody() {
		//	any error handler stored ( ctx.SetErr or StopWith(Plain)Error )
		if ctxErr := ctx.GetErr(); ctxErr != nil {
			// If there is an error here
			// we may need to NOT read the body for security reasons, e.g.
			// unauthorized user tries to send a malicious body.
			requestBody = ac.getErrorText(ctxErr)
		} else {
			requestData, err := ctx.GetBody()
			requestBodyLength := len(requestData)
			if ac.BytesReceivedBody {
				bytesReceived = requestBodyLength // store it, if the total is enabled then this will be overridden.
			}
			if err != nil && ac.RequestBody {
				if err != http.ErrBodyReadAfterClose { // if body was already closed, don't send it as error.
					requestBody = ac.getErrorText(err)
				}
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
		actualResponseData := ctx.Recorder().Body()
		responseBodyLength := len(actualResponseData)

		if ac.BytesSentBody {
			bytesSent = responseBodyLength
		}
		if ac.ResponseBody && responseBodyLength > 0 {
			if ac.BodyMinify {
				// Copy response data as minifier now can change the back slice,
				// fixes: https://github.com/kataras/iris-premium/issues/17.
				responseData := make([]byte, len(actualResponseData))
				copy(responseData, actualResponseData)

				if minified, err := ctx.Application().Minifier().Bytes(ctx.GetContentType(), responseData); err == nil {
					responseBody = string(minified)
					responseBodyLength = len(responseBody)
				}
			}

			if responseBody == "" {
				responseBody = string(actualResponseData)
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
	} else if ac.BytesSentBody {
		bytesSent = ctx.ResponseWriter().Written()
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

	ip := ""
	if ac.IP {
		ip = ctx.RemoteAddr()
	}

	if err := ac.Print(ctx,
		// latency between begin and finish of the handlers chain.
		lat,
		timeFormat,
		// response code.
		ctx.GetStatusCode(),
		// original request's method and path.
		method, path,
		// remote ip.
		ip,
		requestBody, responseBody,
		bytesReceived, bytesSent,
		ctx.Params().Store, ctx.URLParamsSorted(), *fields,
	); err != nil {
		ctx.Application().Logger().Errorf("accesslog: %v", err)
	}
}

// Print writes a log manually.
// The `Handler` method calls it.
func (ac *AccessLog) Print(ctx *context.Context,
	latency time.Duration,
	timeFormat string,
	code int,
	method, path, ip, reqBody, respBody string,
	bytesReceived, bytesSent int,
	params memstore.Store, query []memstore.StringEntry, fields []memstore.Entry) (err error) {

	if ac.Async {
		// atomic.AddUint32(&ac.remaining, 1)
		// This could work ^
		// but to make sure we have the correct number of increments.
		// CAS loop:
		for {
			cur := atomic.LoadUint32(&ac.remaining)
			if atomic.CompareAndSwapUint32(&ac.remaining, cur, cur+1) {
				break
			}
		}

		defer atomic.AddUint32(&ac.remaining, ^uint32(0))
	}

	now := ac.Clock.Now()

	if hasFormatter, hasBroker := ac.formatter != nil, ac.broker != nil; hasFormatter || hasBroker {
		log := ac.logsPool.Get().(*Log)
		log.Logger = ac
		log.Now = now
		log.TimeFormat = timeFormat
		log.Timestamp = now.UnixNano() / 1000000
		log.Latency = latency
		log.Code = code
		log.Method = method
		log.Path = path
		log.IP = ip
		log.Query = query
		log.PathParams = params
		log.Fields = fields
		log.Request = reqBody
		log.Response = respBody
		log.BytesReceived = bytesReceived
		log.BytesSent = bytesSent
		log.Ctx = ctx

		var handled bool
		if hasFormatter {
			handled, err = ac.formatter.Format(log) // formatter can alter this, we wait until it's finished.
			if err != nil {
				ac.logsPool.Put(log)
				return
			}
		}

		if hasBroker { // after Format, it may want to customize the log's fields.
			ac.broker.notify(log.Clone()) // a listener cannot edit the log as we use object pooling.
		}

		ac.logsPool.Put(log) // we don't need it anymore.
		if handled {
			return // OK, it's handled, exit now.
		}
	}

	// the number of separators is the same, in order to be easier
	// for 3rd-party programs to read the result log file.
	builder := ac.bufPool.Get().(*bytes.Buffer)

	builder.WriteString(now.Format(timeFormat))
	builder.WriteByte(ac.Delim)

	builder.WriteString(latency.String())
	builder.WriteByte(ac.Delim)

	builder.WriteString(strconv.Itoa(code))
	builder.WriteByte(ac.Delim)

	builder.WriteString(method)
	builder.WriteByte(ac.Delim)

	builder.WriteString(path)
	builder.WriteByte(ac.Delim)

	if ac.IP {
		ac.writeText(builder, ip)
		builder.WriteByte(ac.Delim)
	}

	// url parameters, path parameters and custom fields separated by space,
	// key=value key2=value2.
	if n, all := parseRequestValues(builder, code, params, query, fields); n > 0 {
		builder.Truncate(all - 1) // remove the last space.
		builder.WriteByte(ac.Delim)
	}

	if ac.BytesReceived || ac.BytesReceivedBody {
		builder.WriteString(formatBytes(bytesReceived))
		builder.WriteByte(ac.Delim)
	}

	if ac.BytesSent || ac.BytesSentBody {
		builder.WriteString(formatBytes(bytesSent))
		builder.WriteByte(ac.Delim)
	}

	if ac.RequestBody {
		ac.writeText(builder, reqBody)
		builder.WriteByte(ac.Delim)
	}

	if ac.ResponseBody {
		ac.writeText(builder, respBody)
		builder.WriteByte(ac.Delim)
	}

	builder.WriteByte(newLine)

	_, err = ac.Write(builder.Bytes())
	builder.Reset()
	ac.bufPool.Put(builder)

	return
}

// We could have a map of blanks per field,
// but let's don't coplicate things so much
// as the end-developer can use a custom template.
func (ac *AccessLog) writeText(buf *bytes.Buffer, s string) {
	if s == "" {
		if len(ac.Blank) == 0 {
			return
		}
		buf.Write(ac.Blank)
	} else {
		buf.WriteString(s)
	}
}

var lineBreaksReplacer = strings.NewReplacer("\n\r", " ", "\n", " ")

func (ac *AccessLog) getErrorText(err error) (text string) { // caller checks for nil.
	if errPanic, ok := context.IsErrPanicRecovery(err); ok {
		ac.Flush() // flush any buffered contents to be written to the output.

		switch ac.PanicLog {
		case LogHandler:
			text = errPanic.CurrentHandlerFileLine
		case LogCallers:
			text = strings.Join(errPanic.Callers, "\n")
		case LogStack:
			text = string(errPanic.Stack)
		}

		text = fmt.Sprintf("error(%v %s)", errPanic.Cause, text)
	} else {
		text = fmt.Sprintf("error(%s)", err.Error())
	}

	if !ac.KeepMultiLineError {
		return lineBreaksReplacer.Replace(text)
	}

	return text
}
