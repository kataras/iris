package context

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/kataras/iris/v12/core/memstore"
	"github.com/kataras/iris/v12/core/netutil"

	"github.com/Shopify/goreferrer"
	"github.com/fatih/structs"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/iris-contrib/schema"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	"github.com/microcosm-cc/bluemonday"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

var (
	// BuildRevision holds the vcs commit id information of the program's build.
	// Available at go version 1.18+
	BuildRevision string
	// BuildTime holds the vcs commit time information of the program's build.
	// Available at go version 1.18+
	BuildTime string
)

type (
	// BodyDecoder is an interface which any struct can implement in order to customize the decode action
	// from ReadJSON and ReadXML
	//
	// Trivial example of this could be:
	// type User struct { Username string }
	//
	// func (u *User) Decode(data []byte) error {
	//	  return json.Unmarshal(data, u)
	// }
	//
	// the 'Context.ReadJSON/ReadXML(&User{})' will call the User's
	// Decode option to decode the request body
	//
	// Note: This is totally optionally, the default decoders
	// for ReadJSON is the encoding/json and for ReadXML is the encoding/xml.
	//
	// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-custom-per-type/main.go
	BodyDecoder interface {
		Decode(data []byte) error
	}

	// BodyDecoderWithContext same as BodyDecoder but it can accept a standard context,
	// which is binded to the HTTP request's context.
	BodyDecoderWithContext interface {
		DecodeContext(ctx context.Context, data []byte) error
	}

	// Unmarshaler is the interface implemented by types that can unmarshal any raw data.
	// TIP INFO: Any pointer to a value which implements the BodyDecoder can be override the unmarshaler.
	Unmarshaler interface {
		Unmarshal(data []byte, outPtr interface{}) error
	}

	// UnmarshalerFunc a shortcut for the Unmarshaler interface
	//
	// See 'Unmarshaler' and 'BodyDecoder' for more.
	//
	// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-custom-via-unmarshaler/main.go
	UnmarshalerFunc func(data []byte, outPtr interface{}) error

	// DecodeFunc is a generic type of decoder function.
	// When the returned error is not nil the decode operation
	// is terminated and the error is received by the ReadJSONStream method,
	// otherwise it continues to read the next available object.
	// Look the `Context.ReadJSONStream` method.
	DecodeFunc func(outPtr interface{}) error
)

// Unmarshal parses the X-encoded data and stores the result in the value pointed to by v.
// Unmarshal uses the inverse of the encodings that Marshal uses, allocating maps,
// slices, and pointers as necessary.
func (u UnmarshalerFunc) Unmarshal(data []byte, v interface{}) error {
	return u(data, v)
}

// LimitRequestBodySize is a middleware which sets a request body size limit
// for all next handlers in the chain.
var LimitRequestBodySize = func(maxRequestBodySizeBytes int64) Handler {
	return func(ctx *Context) {
		ctx.SetMaxRequestBodySize(maxRequestBodySizeBytes)
		ctx.Next()
	}
}

// Map is just a type alias of the map[string]interface{} type.
type Map = map[string]interface{}

// Context is the midle-man server's "object" dealing with incoming requests.
//
// A New context is being acquired from a sync.Pool on each connection.
// The Context is the most important thing on the iris's http flow.
//
// Developers send responses to the client's request through a Context.
// Developers get request information from the client's request a Context.
type Context struct {
	// the http.ResponseWriter wrapped by custom writer.
	writer ResponseWriter
	// the original http.Request
	request *http.Request
	// the current route registered to this request path.
	currentRoute RouteReadOnly

	// the local key-value storage
	params RequestParams  // url named parameters.
	values memstore.Store // generic storage, middleware communication.
	query  url.Values     // GET url query temp cache, useful on many URLParamXXX calls.
	// the underline application app.
	app Application
	// the route's handlers
	handlers Handlers
	// the current position of the handler's chain
	currentHandlerIndex int
	// proceeded reports whether `Proceed` method
	// called before a `Next`. It is a flash field and it is set
	// to true on `Next` call when its called on the last handler in the chain.
	// Reports whether a `Next` is called,
	// even if the handler index remains the same (last handler).
	//
	// Also it's responsible to keep the old value of the last known handler index
	// before StopExecution. See ResumeExecution.
	proceeded int

	// if true, caller is responsible to release the context (put the context to the pool).
	manualRelease bool
}

// NewContext returns a new Context instance.
func NewContext(app Application) *Context {
	return &Context{app: app}
}

/* Not required, unless requested.
// SetApplication sets an Iris Application on-fly.
// Do NOT use it after ServeHTTPC is fired.
func (ctx *Context) SetApplication(app Application) {
	ctx.app = app
}
*/

// Clone returns a copy of the context that
// can be safely used outside the request's scope.
// Note that if the request-response lifecycle terminated
// or request canceled by the client (can be checked by `ctx.IsCanceled()`)
// then the response writer is totally useless.
// The http.Request pointer value is shared.
func (ctx *Context) Clone() *Context {
	valuesCopy := make(memstore.Store, len(ctx.values))
	copy(valuesCopy, ctx.values)

	paramsCopy := make(memstore.Store, len(ctx.params.Store))
	copy(paramsCopy, ctx.params.Store)

	queryCopy := make(url.Values, len(ctx.query))
	for k, v := range ctx.query {
		queryCopy[k] = v
	}

	req := ctx.request.Clone(ctx.request.Context())
	return &Context{
		app:                 ctx.app,
		values:              valuesCopy,
		params:              RequestParams{Store: paramsCopy},
		query:               queryCopy,
		writer:              ctx.writer.Clone(),
		request:             req,
		currentHandlerIndex: stopExecutionIndex,
		proceeded:           ctx.proceeded,
		manualRelease:       ctx.manualRelease,
		currentRoute:        ctx.currentRoute,
	}
}

// BeginRequest is executing once for each request
// it should prepare the (new or acquired from pool) context's fields for the new request.
// Do NOT call it manually. Framework calls it automatically.
//
// Resets
// 1. handlers to nil.
// 2. values to empty.
// 3. the defer function.
// 4. response writer to the http.ResponseWriter.
// 5. request to the *http.Request.
func (ctx *Context) BeginRequest(w http.ResponseWriter, r *http.Request) {
	ctx.currentRoute = nil
	ctx.handlers = nil           // will be filled by router.Serve/HTTP
	ctx.values = ctx.values[0:0] // >>      >>     by context.Values().Set
	ctx.params.Store = ctx.params.Store[0:0]
	ctx.query = nil
	ctx.request = r
	ctx.currentHandlerIndex = 0
	ctx.proceeded = 0
	ctx.manualRelease = false
	ctx.writer = AcquireResponseWriter()
	ctx.writer.BeginResponse(w)
}

// EndRequest is executing once after a response to the request was sent and this context is useless or released.
// Do NOT call it manually. Framework calls it automatically.
//
// 1. executes the OnClose function (if any).
// 2. flushes the response writer's result or fire any error handler.
// 3. releases the response writer.
func (ctx *Context) EndRequest() {
	if !ctx.app.ConfigurationReadOnly().GetDisableAutoFireStatusCode() &&
		StatusCodeNotSuccessful(ctx.GetStatusCode()) {
		ctx.app.FireErrorCode(ctx)
	}

	ctx.writer.FlushResponse()
	ctx.writer.EndResponse()
}

// DisablePoolRelease disables the auto context pool Put call.
// Do NOT use it, unless you know what you are doing.
func (ctx *Context) DisablePoolRelease() {
	ctx.manualRelease = true
}

// IsCanceled reports whether the client canceled the request
// or the underlying connection has gone.
// Note that it will always return true
// when called from a goroutine after the request-response lifecycle.
func (ctx *Context) IsCanceled() bool {
	var err error
	if reqCtx := ctx.request.Context(); reqCtx != nil {
		err = reqCtx.Err()
	} else {
		err = ctx.GetErr()
	}

	return IsErrCanceled(err)
}

// IsErrCanceled reports whether the "err" is caused by a cancellation or timeout.
func IsErrCanceled(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	return (errors.As(err, &netErr) && netErr.Timeout()) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, http.ErrHandlerTimeout) ||
		err.Error() == "closed pool"
}

// OnConnectionClose registers the "cb" Handler
// which will be fired on its on goroutine on a cloned Context
// when the underlying connection has gone away.
//
// The code inside the given callback is running on its own routine,
// as explained above, therefore the callback should NOT
// try to access to handler's Context response writer.
//
// This mechanism can be used to cancel long operations on the server
// if the client has disconnected before the response is ready.
//
// It depends on the Request's Context.Done() channel.
//
// Finally, it reports whether the protocol supports pipelines (HTTP/1.1 with pipelines disabled is not supported).
// The "cb" will not fire for sure if the output value is false.
//
// Note that you can register only one callback per route.
//
// See `OnClose` too.
func (ctx *Context) OnConnectionClose(cb Handler) bool {
	if cb == nil {
		return false
	}

	reqCtx := ctx.Request().Context()
	if reqCtx == nil {
		return false
	}

	notifyClose := reqCtx.Done()
	if notifyClose == nil {
		return false
	}

	go func() {
		<-notifyClose
		// Note(@kataras): No need to clone if not canceled,
		// EndRequest will be called on the end of the handler chain,
		// no matter the cancelation.
		// therefore the context will still be there.
		cb(ctx.Clone())
	}()

	return true
}

// OnConnectionCloseErr same as `OnConnectionClose` but instead it
// receives a function which returns an error.
// If error is not nil, it will be logged as a debug message.
func (ctx *Context) OnConnectionCloseErr(cb func() error) bool {
	if cb == nil {
		return false
	}

	reqCtx := ctx.Request().Context()
	if reqCtx == nil {
		return false
	}

	notifyClose := reqCtx.Done()
	if notifyClose == nil {
		return false
	}

	go func() {
		<-notifyClose
		if err := cb(); err != nil {
			// Can be ignored.
			ctx.app.Logger().Debugf("OnConnectionCloseErr: received error: %v", err)
		}
	}()

	return true
}

// OnClose registers a callback which
// will be fired when the underlying connection has gone away(request canceled)
// on its own goroutine or in the end of the request-response lifecylce
// on the handler's routine itself (Context access).
//
// See `OnConnectionClose` too.
func (ctx *Context) OnClose(cb Handler) {
	if cb == nil {
		return
	}

	// Note(@kataras):
	// - on normal request-response lifecycle
	// the `SetBeforeFlush` will be called first
	// and then `OnConnectionClose`,
	// - when request was canceled before handler finish its job
	// then the `OnConnectionClose` will be called first instead,
	// and when the handler function completed then `SetBeforeFlush` is fired.
	// These are synchronized, they cannot be executed the same exact time,
	// below we just make sure the "cb" is executed once
	// by simple boolean check or an atomic one.
	var executed uint32

	callback := func(ctx *Context) {
		if atomic.CompareAndSwapUint32(&executed, 0, 1) {
			cb(ctx)
		}
	}

	ctx.OnConnectionClose(callback)

	onFlush := func() {
		callback(ctx)
	}

	ctx.writer.SetBeforeFlush(onFlush)
}

// OnCloseErr same as `OnClose` but instead it
// receives a function which returns an error.
// If error is not nil, it will be logged as a debug message.
func (ctx *Context) OnCloseErr(cb func() error) {
	if cb == nil {
		return
	}

	var executed uint32

	callback := func() error {
		if atomic.CompareAndSwapUint32(&executed, 0, 1) {
			return cb()
		}

		return nil
	}

	ctx.OnConnectionCloseErr(callback)

	onFlush := func() {
		if err := callback(); err != nil {
			// Can be ignored.
			ctx.app.Logger().Debugf("OnClose: SetBeforeFlush: received error: %v", err)
		}
	}

	ctx.writer.SetBeforeFlush(onFlush)
}

/* Note(@kataras): just leave end-developer decide.
const goroutinesContextKey = "iris.goroutines"

type goroutines struct {
	wg     *sync.WaitGroup
	length int
	mu     sync.RWMutex
}

var acquireGoroutines = func() interface{} {
	return &goroutines{wg: new(sync.WaitGroup)}
}

func (ctx *Context) Go(fn func(cancelCtx context.Context)) (running int) {
	g := ctx.values.GetOrSet(goroutinesContextKey, acquireGoroutines).(*goroutines)
	if fn != nil {
		g.wg.Add(1)

		g.mu.Lock()
		g.length++
		g.mu.Unlock()

		ctx.waitFunc = g.wg.Wait

		go func(reqCtx context.Context) {
			fn(reqCtx)
			g.wg.Done()

			g.mu.Lock()
			g.length--
			g.mu.Unlock()
		}(ctx.request.Context())
	}

	g.mu.RLock()
	running = g.length
	g.mu.RUnlock()
	return
}
*/

// ResponseWriter returns an http.ResponseWriter compatible response writer, as expected.
func (ctx *Context) ResponseWriter() ResponseWriter {
	return ctx.writer
}

// ResetResponseWriter sets a new ResponseWriter implementation
// to this Context to use as its writer.
// Note, to change the underline http.ResponseWriter use
// ctx.ResponseWriter().SetWriter(http.ResponseWriter) instead.
func (ctx *Context) ResetResponseWriter(newResponseWriter ResponseWriter) {
	if rec, ok := ctx.IsRecording(); ok {
		releaseResponseRecorder(rec)
	}

	ctx.writer = newResponseWriter
}

// Request returns the original *http.Request, as expected.
func (ctx *Context) Request() *http.Request {
	return ctx.request
}

// ResetRequest sets the Context's Request,
// It is useful to store the new request created by a std *http.Request#WithContext() into Iris' Context.
// Use `ResetRequest` when for some reason you want to make a full
// override of the *http.Request.
// Note that: when you just want to change one of each fields you can use the Request() which returns a pointer to Request,
// so the changes will have affect without a full override.
// Usage: you use a native http handler which uses the standard "context" package
// to get values instead of the Iris' Context#Values():
// r := ctx.Request()
// stdCtx := context.WithValue(r.Context(), key, val)
// ctx.ResetRequest(r.WithContext(stdCtx)).
func (ctx *Context) ResetRequest(r *http.Request) {
	ctx.request = r
}

// SetCurrentRoute sets the route internally,
// See `GetCurrentRoute()` method too.
// It's being initialized by the Router.
// See `Exec` or `SetHandlers/AddHandler` methods to simulate a request.
func (ctx *Context) SetCurrentRoute(route RouteReadOnly) {
	ctx.currentRoute = route
}

// GetCurrentRoute returns the current "read-only" route that
// was registered to this request's path.
func (ctx *Context) GetCurrentRoute() RouteReadOnly {
	return ctx.currentRoute
}

// Do sets the "handlers" as the chain
// and executes the first handler,
// handlers should not be empty.
//
// It's used by the router, developers may use that
// to replace and execute handlers immediately.
func (ctx *Context) Do(handlers Handlers) {
	if len(handlers) == 0 {
		return
	}

	ctx.handlers = handlers
	handlers[0](ctx)
}

// AddHandler can add handler(s)
// to the current request in serve-time,
// these handlers are not persistenced to the router.
//
// Router is calling this function to add the route's handler.
// If AddHandler called then the handlers will be inserted
// to the end of the already-defined route's handler.
func (ctx *Context) AddHandler(handlers ...Handler) {
	ctx.handlers = append(ctx.handlers, handlers...)
}

// SetHandlers replaces all handlers with the new.
func (ctx *Context) SetHandlers(handlers Handlers) {
	ctx.handlers = handlers
}

// Handlers keeps tracking of the current handlers.
func (ctx *Context) Handlers() Handlers {
	return ctx.handlers
}

// HandlerIndex sets the current index of the
// current context's handlers chain.
// If n < 0 or the current handlers length is 0 then it just returns the
// current handler index without change the current index.
//
// Look Handlers(), Next() and StopExecution() too.
func (ctx *Context) HandlerIndex(n int) (currentIndex int) {
	if n < 0 || n > len(ctx.handlers)-1 {
		return ctx.currentHandlerIndex
	}

	ctx.currentHandlerIndex = n
	return n
}

// Proceed is an alternative way to check if a particular handler
// has been executed.
// The given "h" Handler can report a failure with `StopXXX` methods
// or ignore calling a `Next` (see `iris.ExecutionRules` too).
//
// This is useful only when you run a handler inside
// another handler. It justs checks for before index and the after index.
//
// A usecase example is when you want to execute a middleware
// inside controller's `BeginRequest` that calls the `ctx.Next` inside it.
// The Controller looks the whole flow (BeginRequest, method handler, EndRequest)
// as one handler, so `ctx.Next` will not be reflected to the method handler
// if called from the `BeginRequest`.
//
// Although `BeginRequest` should NOT be used to call other handlers,
// the `BeginRequest` has been introduced to be able to set
// common data to all method handlers before their execution.
// Controllers can accept middleware(s) from the MVC's Application's Router as normally.
//
// That said let's see an example of `ctx.Proceed`:
//
//	var authMiddleware = basicauth.New(basicauth.Config{
//		Users: map[string]string{
//			"admin": "password",
//		},
//	})
//
//	func (c *UsersController) BeginRequest(ctx iris.Context) {
//		if !ctx.Proceed(authMiddleware) {
//			ctx.StopExecution()
//		}
//	}
//
// This Get() will be executed in the same handler as `BeginRequest`,
// internally controller checks for `ctx.StopExecution`.
// So it will not be fired if BeginRequest called the `StopExecution`.
//
//	func(c *UsersController) Get() []models.User {
//		  return c.Service.GetAll()
//	}
//
// Alternative way is `!ctx.IsStopped()` if middleware make use of the `ctx.StopExecution()` on failure.
func (ctx *Context) Proceed(h Handler) bool {
	_, ok := ctx.ProceedAndReportIfStopped(h)
	return ok
}

// ProceedAndReportIfStopped same as "Proceed" method
// but the first output parameter reports whether the "h"
// called "StopExecution" manually.
func (ctx *Context) ProceedAndReportIfStopped(h Handler) (bool, bool) {
	ctx.proceeded = internalPauseExecutionIndex

	// Store the current index.
	beforeIdx := ctx.currentHandlerIndex
	h(ctx)
	// Retrieve the next one, if Next is called this is beforeIdx + 1 and so on.
	afterIdx := ctx.currentHandlerIndex
	// Restore prev index, no matter what.
	ctx.currentHandlerIndex = beforeIdx

	proceededByNext := ctx.proceeded == internalProceededHandlerIndex
	ctx.proceeded = beforeIdx

	// Stop called, return false but keep the handlers index.
	if afterIdx == stopExecutionIndex {
		return true, false
	}

	if proceededByNext {
		return false, true
	}

	// Next called or not.
	return false, afterIdx > beforeIdx
}

// HandlerName returns the current handler's name, helpful for debugging.
func (ctx *Context) HandlerName() string {
	return HandlerName(ctx.handlers[ctx.currentHandlerIndex])
}

// HandlerFileLine returns the current running handler's function source file and line information.
// Useful mostly when debugging.
func (ctx *Context) HandlerFileLine() (file string, line int) {
	return HandlerFileLine(ctx.handlers[ctx.currentHandlerIndex])
}

// RouteName returns the route name that this handler is running on.
// Note that it may return empty on not found handlers.
func (ctx *Context) RouteName() string {
	if ctx.currentRoute == nil {
		return ""
	}

	return ctx.currentRoute.Name()
}

// Next calls the next handler from the handlers chain,
// it should be used inside a middleware.
func (ctx *Context) Next() {
	if ctx.IsStopped() {
		return
	}

	if ctx.proceeded <= internalPauseExecutionIndex /* pause and proceeded */ {
		ctx.proceeded = internalProceededHandlerIndex
		return
	}

	nextIndex, n := ctx.currentHandlerIndex+1, len(ctx.handlers)
	if nextIndex < n {
		ctx.currentHandlerIndex = nextIndex
		ctx.handlers[nextIndex](ctx)
	}
}

// NextOr checks if chain has a next handler, if so then it executes it
// otherwise it sets a new chain assigned to this Context based on the given handler(s)
// and executes its first handler.
//
// Returns true if next handler exists and executed, otherwise false.
//
// Note that if no next handler found and handlers are missing then
// it sends a Status Not Found (404) to the client and it stops the execution.
func (ctx *Context) NextOr(handlers ...Handler) bool {
	if next := ctx.NextHandler(); next != nil {
		ctx.Skip() // skip this handler from the chain.
		next(ctx)
		return true
	}

	if len(handlers) == 0 {
		ctx.NotFound()
		ctx.StopExecution()
		return false
	}

	ctx.Do(handlers)

	return false
}

// NextOrNotFound checks if chain has a next handler, if so then it executes it
// otherwise it sends a Status Not Found (404) to the client and stops the execution.
//
// Returns true if next handler exists and executed, otherwise false.
func (ctx *Context) NextOrNotFound() bool { return ctx.NextOr() }

// NextHandler returns (it doesn't execute) the next handler from the handlers chain.
//
// Use .Skip() to skip this handler if needed to execute the next of this returning handler.
func (ctx *Context) NextHandler() Handler {
	if ctx.IsStopped() {
		return nil
	}
	nextIndex := ctx.currentHandlerIndex + 1
	// check if it has a next middleware
	if nextIndex < len(ctx.handlers) {
		return ctx.handlers[nextIndex]
	}
	return nil
}

// Skip skips/ignores the next handler from the handlers chain,
// it should be used inside a middleware.
func (ctx *Context) Skip() {
	ctx.HandlerIndex(ctx.currentHandlerIndex + 1)
}

const (
	stopExecutionIndex            = -1
	internalPauseExecutionIndex   = -2
	internalProceededHandlerIndex = -3
)

// StopExecution stops the handlers chain of this request.
// Meaning that any following `Next` calls are ignored,
// as a result the next handlers in the chain will not be fire.
//
// See ResumeExecution too.
func (ctx *Context) StopExecution() {
	if curIdx := ctx.currentHandlerIndex; curIdx != stopExecutionIndex {
		// Protect against multiple calls of StopExecution.
		// Resume should set the last proceeded handler index.
		// Store the current index.
		ctx.proceeded = curIdx
		// And stop.
		ctx.currentHandlerIndex = stopExecutionIndex
	}
}

// IsStopped reports whether the current position of the context's handlers is -1,
// means that the StopExecution() was called at least once.
func (ctx *Context) IsStopped() bool {
	return ctx.currentHandlerIndex == stopExecutionIndex
}

// ResumeExecution sets the current handler index to the last
// index of the executed handler before StopExecution method was fired.
//
// Reports whether it's restored after a StopExecution call.
func (ctx *Context) ResumeExecution() bool {
	if ctx.IsStopped() {
		ctx.currentHandlerIndex = ctx.proceeded
		return true
	}

	return false
}

// StopWithStatus stops the handlers chain and writes the "statusCode".
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
func (ctx *Context) StopWithStatus(statusCode int) {
	ctx.StopExecution()
	ctx.StatusCode(statusCode)
}

// StopWithText stops the handlers chain and writes the "statusCode"
// among with a fmt-style text of "format" and optional arguments.
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
func (ctx *Context) StopWithText(statusCode int, format string, args ...interface{}) {
	ctx.StopWithStatus(statusCode)
	ctx.WriteString(fmt.Sprintf(format, args...))
}

// StopWithError stops the handlers chain and writes the "statusCode"
// among with the error "err".
// It Calls the `SetErr` method so error handlers can access the given error.
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
//
// If the given "err" is private then the
// status code's text is rendered instead (unless a registered error handler overrides it).
func (ctx *Context) StopWithError(statusCode int, err error) {
	if err == nil {
		return
	}

	ctx.SetErr(err)
	if _, ok := err.(ErrPrivate); ok {
		// error is private, we SHOULD not render it,
		// leave the error handler alone to
		// render the code's text instead.
		ctx.StopWithStatus(statusCode)
		return
	}

	ctx.StopWithText(statusCode, err.Error())
}

// StopWithPlainError like `StopWithError` but it does NOT
// write anything to the response writer, it stores the error
// so any error handler matching the given "statusCode" can handle it by its own.
func (ctx *Context) StopWithPlainError(statusCode int, err error) {
	if err == nil {
		return
	}

	ctx.SetErr(err)
	ctx.StopWithStatus(statusCode)
}

// StopWithJSON stops the handlers chain, writes the status code
// and sends a JSON response.
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
func (ctx *Context) StopWithJSON(statusCode int, jsonObject interface{}) error {
	ctx.StopWithStatus(statusCode)
	return ctx.writeJSON(jsonObject, &DefaultJSONOptions) // do not modify - see errors.DefaultContextErrorHandler.
}

// StopWithProblem stops the handlers chain, writes the status code
// and sends an application/problem+json response.
// See `iris.NewProblem` to build a "problem" value correctly.
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
func (ctx *Context) StopWithProblem(statusCode int, problem Problem) error {
	ctx.StopWithStatus(statusCode)
	problem.Status(statusCode)
	return ctx.Problem(problem)
}

//  +------------------------------------------------------------+
//  | Current "user/request" storage                             |
//  | and share information between the handlers - Values().     |
//  | Save and get named path parameters - Params()              |
//  +------------------------------------------------------------+

// Params returns the current url's named parameters key-value storage.
// Named path parameters are being saved here.
// This storage, as the whole context, is per-request lifetime.
func (ctx *Context) Params() *RequestParams {
	return &ctx.params
}

// Values returns the current "user" storage.
// Named path parameters and any optional data can be saved here.
// This storage, as the whole context, is per-request lifetime.
//
// You can use this function to Set and Get local values
// that can be used to share information between handlers and middleware.
func (ctx *Context) Values() *memstore.Store {
	return &ctx.values
}

//  +------------------------------------------------------------+
//  | Path, Host, Subdomain, IP, Headers etc...                  |
//  +------------------------------------------------------------+

// Method returns the request.Method, the client's http method to the server.
func (ctx *Context) Method() string {
	return ctx.request.Method
}

// Path returns the full request path,
// escaped if EnablePathEscape config field is true.
func (ctx *Context) Path() string {
	return ctx.RequestPath(ctx.app.ConfigurationReadOnly().GetEnablePathEscape())
}

// DecodeQuery returns the uri parameter as url (string)
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.QueryUnescape
func DecodeQuery(path string) string {
	if path == "" {
		return ""
	}
	encodedPath, err := url.QueryUnescape(path)
	if err != nil {
		return path
	}
	return encodedPath
}

// DecodeURL returns the decoded uri
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.Parse
func DecodeURL(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return u.String()
}

// RequestPath returns the full request path,
// based on the 'escape'.
func (ctx *Context) RequestPath(escape bool) string {
	if escape {
		return ctx.request.URL.EscapedPath() // DecodeQuery(ctx.request.URL.EscapedPath())
	}

	return ctx.request.URL.Path // RawPath returns empty, requesturi can be used instead also.
}

const sufscheme = "://"

// GetScheme returns the full scheme of the request URL (https://, http:// or ws:// and e.t.c.).
func GetScheme(r *http.Request) string {
	scheme := r.URL.Scheme

	if scheme == "" {
		if r.TLS != nil {
			scheme = netutil.SchemeHTTPS
		} else {
			scheme = netutil.SchemeHTTP
		}
	}

	return scheme + sufscheme
}

// Scheme returns the full scheme of the request (including :// suffix).
func (ctx *Context) Scheme() string {
	return GetScheme(ctx.Request())
}

// PathPrefixMap accepts a map of string and a handler.
// The key of "m" is the key, which is the prefix, regular expressions are not valid.
// The value of "m" is the handler that will be executed if HasPrefix(context.Path).
// func (ctx *Context) PathPrefixMap(m map[string]context.Handler) bool {
// 	path := ctx.Path()
// 	for k, v := range m {
// 		if strings.HasPrefix(path, k) {
// 			v(ctx)
// 			return true
// 		}
// 	}
// 	return false
// } no, it will not work because map is a random peek data structure.

// GetHost returns the host part of the current URI.
func GetHost(r *http.Request) string {
	// contains subdomain.
	if host := r.URL.Host; host != "" {
		return host
	}
	return r.Host
}

// Host returns the host:port part of the request URI, calls the `Request().Host`.
// To get the subdomain part as well use the `Request().URL.Host` method instead.
// To get the subdomain only use the `Subdomain` method instead.
// This method makes use of the `Configuration.HostProxyHeaders` field too.
func (ctx *Context) Host() string {
	for header, ok := range ctx.app.ConfigurationReadOnly().GetHostProxyHeaders() {
		if !ok {
			continue
		}

		if host := ctx.GetHeader(header); host != "" {
			return host
		}
	}

	return GetHost(ctx.request)
}

// GetDomain resolves and returns the server's domain.
// To customize its behavior, developers can modify this package-level function at initialization.
var GetDomain = func(hostport string) string {
	host := hostport
	if tmp, _, err := net.SplitHostPort(hostport); err == nil {
		host = tmp
	}

	switch host {
	// We could use the netutil.LoopbackRegex but leave it as it's for now, it's faster.
	case "localhost", "127.0.0.1", "0.0.0.0", "::1", "[::1]", "0:0:0:0:0:0:0:0", "0:0:0:0:0:0:0:1":
		// loopback.
		return "localhost"
	default:
		if net.ParseIP(host) != nil { // if it's an IP, see #1945.
			return host
		}

		if domain, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
			host = domain
		}

		return host
	}
}

// Domain returns the root level domain.
func (ctx *Context) Domain() string {
	return GetDomain(ctx.Host())
}

// GetSubdomainFull returns the full subdomain level, e.g.
// [test.user.]mydomain.com.
func GetSubdomainFull(r *http.Request) string {
	host := GetHost(r)            // host:port
	rootDomain := GetDomain(host) // mydomain.com
	rootDomainIdx := strings.Index(host, rootDomain)
	if rootDomainIdx == -1 {
		return ""
	}

	return host[0:rootDomainIdx]
}

// SubdomainFull returns the full subdomain level, e.g.
// [test.user.]mydomain.com.
// Note that HostProxyHeaders are being respected here.
func (ctx *Context) SubdomainFull() string {
	host := ctx.Host()            // host:port
	rootDomain := GetDomain(host) // mydomain.com
	rootDomainIdx := strings.Index(host, rootDomain)
	if rootDomainIdx == -1 {
		return ""
	}

	return host[0:rootDomainIdx]
}

// Subdomain returns the first subdomain of this request,
// e.g. [user.]mydomain.com.
// See `SubdomainFull` too.
func (ctx *Context) Subdomain() (subdomain string) {
	host := ctx.Host()
	if index := strings.IndexByte(host, '.'); index > 0 {
		subdomain = host[0:index]
	}

	// listening on mydomain.com:80
	// subdomain = mydomain, but it's wrong, it should return ""
	vhost := ctx.app.ConfigurationReadOnly().GetVHost()
	if strings.Contains(vhost, subdomain) { // then it's not subdomain
		return ""
	}

	return
}

// FindClosest returns a list of "n" paths close to
// this request based on subdomain and request path.
//
// Order may change.
// Example: https://github.com/kataras/iris/tree/main/_examples/routing/intelligence/manual
func (ctx *Context) FindClosest(n int) []string {
	return ctx.app.FindClosestPaths(ctx.Subdomain(), ctx.Path(), n)
}

// IsWWW returns true if the current subdomain (if any) is www.
func (ctx *Context) IsWWW() bool {
	host := ctx.Host()
	if index := strings.IndexByte(host, '.'); index > 0 {
		// if it has a subdomain and it's www then return true.
		if subdomain := host[0:index]; !strings.Contains(ctx.app.ConfigurationReadOnly().GetVHost(), subdomain) {
			return subdomain == "www"
		}
	}
	return false
}

// FullRequestURI returns the full URI,
// including the scheme, the host and the relative requested path/resource.
func (ctx *Context) FullRequestURI() string {
	return ctx.AbsoluteURI(ctx.Path())
}

// RemoteAddr tries to parse and return the real client's request IP.
//
// Based on allowed headers names that can be modified from Configuration.RemoteAddrHeaders.
//
// If parse based on these headers fail then it will return the Request's `RemoteAddr` field
// which is filled by the server before the HTTP handler,
// unless the Configuration.RemoteAddrHeadersForce was set to true
// which will force this method to return the first IP from RemoteAddrHeaders
// even if it's part of a private network.
//
// Look `Configuration.RemoteAddrHeaders`,
//
//	Configuration.RemoteAddrHeadersForce,
//	Configuration.WithRemoteAddrHeader(...),
//	Configuration.WithoutRemoteAddrHeader(...) and
//	Configuration.RemoteAddrPrivateSubnetsW for more.
func (ctx *Context) RemoteAddr() string {
	if remoteHeaders := ctx.app.ConfigurationReadOnly().GetRemoteAddrHeaders(); len(remoteHeaders) > 0 {
		privateSubnets := ctx.app.ConfigurationReadOnly().GetRemoteAddrPrivateSubnets()

		for _, headerName := range remoteHeaders {
			ipAddresses := strings.Split(ctx.GetHeader(headerName), ",")
			if ip, ok := netutil.GetIPAddress(ipAddresses, privateSubnets); ok {
				return ip
			}
		}

		if ctx.app.ConfigurationReadOnly().GetRemoteAddrHeadersForce() {
			for _, headerName := range remoteHeaders {
				// return the first valid IP,
				//  even if it's a part of a private network.
				ipAddresses := strings.Split(ctx.GetHeader(headerName), ",")
				for _, addr := range ipAddresses {
					if ip, _, err := net.SplitHostPort(addr); err == nil {
						return ip
					}
				}
			}
		}
	}

	addr := strings.TrimSpace(ctx.request.RemoteAddr)
	if addr != "" {
		// if addr has port use the net.SplitHostPort otherwise(error occurs) take as it is
		if ip, _, err := net.SplitHostPort(addr); err == nil {
			return ip
		}
	}

	return addr
}

// TrimHeaderValue returns the "v[0:first space or semicolon]".
func TrimHeaderValue(v string) string {
	for i, char := range v {
		if char == ' ' || char == ';' {
			return v[:i]
		}
	}
	return v
}

// GetHeader returns the request header's value based on its name.
func (ctx *Context) GetHeader(name string) string {
	return ctx.request.Header.Get(name)
}

// IsAjax returns true if this request is an 'ajax request'( XMLHttpRequest)
//
// There is no a 100% way of knowing that a request was made via Ajax.
// You should never trust data coming from the client, they can be easily overcome by spoofing.
//
// Note that "X-Requested-With" Header can be modified by any client(because of "X-"),
// so don't rely on IsAjax for really serious stuff,
// try to find another way of detecting the type(i.e, content type),
// there are many blogs that describe these problems and provide different kind of solutions,
// it's always depending on the application you're building,
// this is the reason why this `IsAjax` is simple enough for general purpose use.
//
// Read more at: https://developer.mozilla.org/en-US/docs/AJAX
// and https://xhr.spec.whatwg.org/
func (ctx *Context) IsAjax() bool {
	return ctx.GetHeader("X-Requested-With") == "XMLHttpRequest"
}

var isMobileRegex = regexp.MustCompile("(?:hpw|i|web)os|alamofire|alcatel|amoi|android|avantgo|blackberry|blazer|cell|cfnetwork|darwin|dolfin|dolphin|fennec|htc|ip(?:hone|od|ad)|ipaq|j2me|kindle|midp|minimo|mobi|motorola|nec-|netfront|nokia|opera m(ob|in)i|palm|phone|pocket|portable|psp|silk-accelerated|skyfire|sony|ucbrowser|up.browser|up.link|windows ce|xda|zte|zune")

// IsMobile checks if client is using a mobile device(phone or tablet) to communicate with this server.
// If the return value is true that means that the http client using a mobile
// device to communicate with the server, otherwise false.
//
// Keep note that this checks the "User-Agent" request header.
func (ctx *Context) IsMobile() bool {
	s := strings.ToLower(ctx.GetHeader("User-Agent"))
	return isMobileRegex.MatchString(s)
}

var isScriptRegex = regexp.MustCompile("curl|wget|collectd|python|urllib|java|jakarta|httpclient|phpcrawl|libwww|perl|go-http|okhttp|lua-resty|winhttp|awesomium")

// IsScript reports whether a client is a script.
func (ctx *Context) IsScript() bool {
	s := strings.ToLower(ctx.GetHeader("User-Agent"))
	return isScriptRegex.MatchString(s)
}

// IsSSL reports whether the client is running under HTTPS SSL.
//
// See `IsHTTP2` too.
func (ctx *Context) IsSSL() bool {
	ssl := strings.EqualFold(ctx.request.URL.Scheme, "https") || ctx.request.TLS != nil
	if !ssl {
		for k, v := range ctx.app.ConfigurationReadOnly().GetSSLProxyHeaders() {
			if ctx.GetHeader(k) == v {
				ssl = true
				break
			}
		}
	}
	return ssl
}

// IsHTTP2 reports whether the protocol version for incoming request was HTTP/2.
// The client code always uses either HTTP/1.1 or HTTP/2.
//
// See `IsSSL` too.
func (ctx *Context) IsHTTP2() bool {
	return ctx.request.ProtoMajor == 2
}

// IsGRPC reports whether the request came from a gRPC client.
func (ctx *Context) IsGRPC() bool {
	return ctx.IsHTTP2() && strings.Contains(ctx.GetContentTypeRequested(), ContentGRPCHeaderValue)
}

type (
	// Referrer contains the extracted information from the `GetReferrer`
	//
	// The structure contains struct tags for JSON, form, XML, YAML and TOML.
	// Look the `GetReferrer() Referrer` and `goreferrer` external package.
	Referrer struct {
		// The raw refer(r)er URL.
		Raw        string                   `json:"raw" form:"raw" xml:"Raw" yaml:"Raw" toml:"Raw"`
		Type       ReferrerType             `json:"type" form:"referrer_type" xml:"Type" yaml:"Type" toml:"Type"`
		Label      string                   `json:"label" form:"referrer_form" xml:"Label" yaml:"Label" toml:"Label"`
		URL        string                   `json:"url" form:"referrer_url" xml:"URL" yaml:"URL" toml:"URL"`
		Subdomain  string                   `json:"subdomain" form:"referrer_subdomain" xml:"Subdomain" yaml:"Subdomain" toml:"Subdomain"`
		Domain     string                   `json:"domain" form:"referrer_domain" xml:"Domain" yaml:"Domain" toml:"Domain"`
		Tld        string                   `json:"tld" form:"referrer_tld" xml:"Tld" yaml:"Tld" toml:"Tld"`
		Path       string                   `json:"path" form:"referrer_path" xml:"Path" yaml:"Path" toml:"Path"`
		Query      string                   `json:"query" form:"referrer_query" xml:"Query" yaml:"Query" toml:"GoogleType"`
		GoogleType ReferrerGoogleSearchType `json:"googleType" form:"referrer_google_type" xml:"GoogleType" yaml:"GoogleType" toml:"GoogleType"`
	}

	// ReferrerType is the goreferrer enum for a referrer type (indirect, direct, email, search, social).
	ReferrerType = goreferrer.ReferrerType

	// ReferrerGoogleSearchType is the goreferrer enum for a google search type (organic, adwords).
	ReferrerGoogleSearchType = goreferrer.GoogleSearchType
)

// String returns the raw ref url.
func (ref Referrer) String() string {
	return ref.Raw
}

// Contains the available values of the goreferrer enums.
const (
	ReferrerInvalid ReferrerType = iota
	ReferrerIndirect
	ReferrerDirect
	ReferrerEmail
	ReferrerSearch
	ReferrerSocial

	ReferrerNotGoogleSearch ReferrerGoogleSearchType = iota
	ReferrerGoogleOrganicSearch
	ReferrerGoogleAdwords
)

// unnecessary but good to know the default values upfront.
var emptyReferrer = Referrer{Type: ReferrerInvalid, GoogleType: ReferrerNotGoogleSearch}

// GetReferrer extracts and returns the information from the "Referer" (or "Referrer") header
// and url query parameter as specified in https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy.
func (ctx *Context) GetReferrer() Referrer {
	// the underline net/http follows the https://tools.ietf.org/html/rfc7231#section-5.5.2,
	// so there is nothing special left to do.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
	refURL := ctx.GetHeader("Referer")
	if refURL == "" {
		refURL = ctx.GetHeader("Referrer")
		if refURL == "" {
			refURL = ctx.URLParam("referer")
			if refURL == "" {
				refURL = ctx.URLParam("referrer")
			}
		}
	}

	if refURL == "" {
		return emptyReferrer
	}

	if ref := goreferrer.DefaultRules.Parse(refURL); ref.Type > goreferrer.Invalid {
		return Referrer{
			Raw:        refURL,
			Type:       ReferrerType(ref.Type),
			Label:      ref.Label,
			URL:        ref.URL,
			Subdomain:  ref.Subdomain,
			Domain:     ref.Domain,
			Tld:        ref.Tld,
			Path:       ref.Path,
			Query:      ref.Query,
			GoogleType: ReferrerGoogleSearchType(ref.GoogleType),
		}
	}

	return emptyReferrer
}

// SetLanguage force-sets the language for i18n, can be used inside a middleare.
// It has the highest priority over the rest and if it is empty then it is ignored,
// if it set to a static string of "default" or to the default language's code
// then the rest of the language extractors will not be called at all and
// the default language will be set instead.
//
// See `i18n.ExtractFunc` for a more organised way of the same feature.
func (ctx *Context) SetLanguage(langCode string) {
	ctx.values.Set(ctx.app.ConfigurationReadOnly().GetLanguageContextKey(), langCode)
}

// GetLocale returns the current request's `Locale` found by i18n middleware.
// It always fallbacks to the default one.
// See `Tr` too.
func (ctx *Context) GetLocale() Locale {
	// Cache the Locale itself for multiple calls of `Tr` method.
	contextKey := ctx.app.ConfigurationReadOnly().GetLocaleContextKey()
	if v := ctx.values.Get(contextKey); v != nil {
		if locale, ok := v.(Locale); ok {
			return locale
		}
	}

	if locale := ctx.app.I18nReadOnly().GetLocale(ctx); locale != nil {
		ctx.values.Set(contextKey, locale)
		return locale
	}

	return nil
}

// Tr returns a i18n localized message based on format with optional arguments.
// See `GetLocale` too.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/i18n
func (ctx *Context) Tr(key string, args ...interface{}) string {
	return ctx.app.I18nReadOnly().TrContext(ctx, key, args...)
}

//  +------------------------------------------------------------+
//  | Response Headers helpers                                   |
//  +------------------------------------------------------------+

// Header adds a header to the response, if value is empty
// it removes the header by its name.
func (ctx *Context) Header(name string, value string) {
	if value == "" {
		ctx.writer.Header().Del(name)
		return
	}
	ctx.writer.Header().Add(name, value)
}

const contentTypeContextKey = "iris.content_type"

func shouldAppendCharset(cType string) bool {
	if idx := strings.IndexRune(cType, '/'); idx > 1 && len(cType) > idx+1 {
		typ := cType[0:idx]
		if typ == "application" {
			switch cType[idx+1:] {
			case "json", "xml", "yaml", "problem+json", "problem+xml":
				return true
			default:
				return false
			}
		}

	}

	return true
}

func (ctx *Context) contentTypeOnce(cType string, charset string) {
	if charset == "" {
		charset = ctx.app.ConfigurationReadOnly().GetCharset()
	}

	if shouldAppendCharset(cType) {
		cType += "; charset=" + charset
	}

	ctx.values.Set(contentTypeContextKey, cType)
	ctx.writer.Header().Set(ContentTypeHeaderKey, cType)
}

// ContentType sets the response writer's
// header "Content-Type" to the 'cType'.
func (ctx *Context) ContentType(cType string) {
	if cType == "" {
		return
	}

	if _, wroteOnce := ctx.values.GetEntry(contentTypeContextKey); wroteOnce {
		return
	}

	// 1. if it's path or a filename or an extension,
	// then take the content type from that,
	// ^ No, it's not always a file,e .g. vnd.$type
	// if strings.Contains(cType, ".") {
	// 	ext := filepath.Ext(cType)
	// 	cType = mime.TypeByExtension(ext)
	// }
	// if doesn't contain a charset already then append it
	if shouldAppendCharset(cType) {
		if !strings.Contains(cType, "charset") {
			cType += "; charset=" + ctx.app.ConfigurationReadOnly().GetCharset()
		}
	}

	ctx.writer.Header().Set(ContentTypeHeaderKey, cType)
}

// GetContentType returns the response writer's
// header value of "Content-Type".
func (ctx *Context) GetContentType() string {
	return ctx.writer.Header().Get(ContentTypeHeaderKey)
}

// GetContentTypeRequested returns the request's
// trim-ed(without the charset and priority values)
// header value of "Content-Type".
func (ctx *Context) GetContentTypeRequested() string {
	// could use mime.ParseMediaType too.
	return TrimHeaderValue(ctx.GetHeader(ContentTypeHeaderKey))
}

// GetContentLength returns the request's
// header value of "Content-Length".
func (ctx *Context) GetContentLength() int64 {
	if v := ctx.GetHeader(ContentLengthHeaderKey); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	return 0
}

// StatusCode sets the status code header to the response.
// Look .GetStatusCode & .FireStatusCode too.
//
// Note that you must set status code before write response body (except when recorder is used).
func (ctx *Context) StatusCode(statusCode int) {
	ctx.writer.WriteHeader(statusCode)
}

// NotFound emits an error 404 to the client, using the specific custom error error handler.
// Note that you may need to call ctx.StopExecution() if you don't want the next handlers
// to be executed. Next handlers are being executed on iris because you can alt the
// error code and change it to a more specific one, i.e
// users := app.Party("/users")
// users.Done(func(ctx iris.Context){ if ctx.GetStatusCode() == 400 { /*  custom error code for /users */ }})
func (ctx *Context) NotFound() {
	ctx.StatusCode(http.StatusNotFound)
}

// GetStatusCode returns the current status code of the response.
// Look StatusCode too.
func (ctx *Context) GetStatusCode() int {
	return ctx.writer.StatusCode()
}

//  +------------------------------------------------------------+
//  | Various Request and Post Data                              |
//  +------------------------------------------------------------+

func (ctx *Context) getQuery() url.Values {
	if ctx.query == nil {
		ctx.query = ctx.request.URL.Query()
	}

	return ctx.query
}

// URLParamExists returns true if the url parameter exists, otherwise false.
func (ctx *Context) URLParamExists(name string) bool {
	_, exists := ctx.getQuery()[name]
	return exists
}

// URLParamDefault returns the get parameter from a request, if not found then "def" is returned.
func (ctx *Context) URLParamDefault(name string, def string) string {
	if v := ctx.getQuery().Get(name); v != "" {
		return v
	}

	return def
}

// URLParam returns the get parameter from a request, if any.
func (ctx *Context) URLParam(name string) string {
	return ctx.URLParamDefault(name, "")
}

// URLParamSlice a shortcut of ctx.Request().URL.Query()[name].
// Like `URLParam` but it returns all values instead of a single string separated by commas.
// Returns the values of a url query of the given "name" as string slice, e.g.
// ?names=john&names=doe&names=kataras and ?names=john,doe,kataras will return [ john doe kataras].
//
// Note that, this method skips any empty entries.
//
// See `URLParamsSorted` for sorted values.
func (ctx *Context) URLParamSlice(name string) []string {
	values := ctx.getQuery()[name]
	n := len(values)
	if n == 0 {
		return values
	}

	var sep string
	if sepPtr := ctx.app.ConfigurationReadOnly().GetURLParamSeparator(); sepPtr != nil {
		sep = *sepPtr
	}

	normalizedValues := make([]string, 0, n)
	for _, v := range values {
		if v == "" {
			continue
		}

		if sep != "" {
			values := strings.Split(v, sep)
			normalizedValues = append(normalizedValues, values...)
			continue
		}

		normalizedValues = append(normalizedValues, v)
	}

	return normalizedValues
}

// URLParamTrim returns the url query parameter with trailing white spaces removed from a request.
func (ctx *Context) URLParamTrim(name string) string {
	return strings.TrimSpace(ctx.URLParam(name))
}

// URLParamEscape returns the escaped url query parameter from a request.
func (ctx *Context) URLParamEscape(name string) string {
	return DecodeQuery(ctx.URLParam(name))
}

// ErrNotFound is the type error which API users can make use of
// to check if a `Context` action of a `Handler` is type of Not Found,
// e.g. URL Query Parameters.
// Example:
//
// n, err := context.URLParamInt("url_query_param_name")
//
//	if errors.Is(err, context.ErrNotFound) {
//		// [handle error...]
//	}
//
// Another usage would be `err == context.ErrNotFound`
// HOWEVER prefer use the new `errors.Is` as API details may change in the future.
var ErrNotFound = errors.New("not found")

// URLParamInt returns the url query parameter as int value from a request,
// returns -1 and an error if parse failed or not found.
func (ctx *Context) URLParamInt(name string) (int, error) {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return -1, err
		}
		return n, nil
	}

	return -1, ErrNotFound
}

// URLParamIntDefault returns the url query parameter as int value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *Context) URLParamIntDefault(name string, def int) int {
	v, err := ctx.URLParamInt(name)
	if err != nil {
		return def
	}

	return v
}

// URLParamInt32Default returns the url query parameter as int32 value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *Context) URLParamInt32Default(name string, def int32) int32 {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return def
		}

		return int32(n)
	}

	return def
}

// URLParamInt64 returns the url query parameter as int64 value from a request,
// returns -1 and an error if parse failed or not found.
func (ctx *Context) URLParamInt64(name string) (int64, error) {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return -1, err
		}
		return n, nil
	}

	return -1, ErrNotFound
}

// URLParamInt64Default returns the url query parameter as int64 value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *Context) URLParamInt64Default(name string, def int64) int64 {
	v, err := ctx.URLParamInt64(name)
	if err != nil {
		return def
	}

	return v
}

// URLParamUint64 returns the url query parameter as uint64 value from a request.
// Returns 0 on parse errors or when the URL parameter does not exist in the Query.
func (ctx *Context) URLParamUint64(name string) uint64 {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	}

	return 0
}

// URLParamFloat64 returns the url query parameter as float64 value from a request,
// returns an error and -1 if parse failed.
func (ctx *Context) URLParamFloat64(name string) (float64, error) {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return -1, err
		}
		return n, nil
	}

	return -1, ErrNotFound
}

// URLParamFloat64Default returns the url query parameter as float64 value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *Context) URLParamFloat64Default(name string, def float64) float64 {
	v, err := ctx.URLParamFloat64(name)
	if err != nil {
		return def
	}

	return v
}

// URLParamBool returns the url query parameter as boolean value from a request,
// returns an error if parse failed.
func (ctx *Context) URLParamBool(name string) (bool, error) {
	return strconv.ParseBool(ctx.URLParam(name))
}

// URLParamBoolDefault returns the url query parameter as boolean value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *Context) URLParamBoolDefault(name string, def bool) bool {
	v, err := ctx.URLParamBool(name)
	if err != nil {
		return def
	}

	return v
}

// URLParams returns a map of URL Query parameters.
// If the value of a URL parameter is a slice,
// then it is joined as one separated by comma.
// It returns an empty map on empty URL query.
//
// See URLParamsSorted too.
func (ctx *Context) URLParams() map[string]string {
	q := ctx.getQuery()
	values := make(map[string]string, len(q))

	for k, v := range q {
		values[k] = strings.Join(v, ",")
	}

	return values
}

// URLParamsSorted returns a sorted (by key) slice
// of key-value entries of the URL Query parameters.
func (ctx *Context) URLParamsSorted() []memstore.StringEntry {
	q := ctx.getQuery()
	n := len(q)
	if n == 0 {
		return nil
	}

	keys := make([]string, 0, n)
	for key := range q {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	entries := make([]memstore.StringEntry, 0, n)
	for _, key := range keys {
		value := q[key]
		entries = append(entries, memstore.StringEntry{
			Key:   key,
			Value: strings.Join(value, ","),
		})
	}

	return entries
}

// ResetQuery clears the GET URL Query request, temporary, cache.
// Any new URLParamXXX calls will receive the new parsed values.
func (ctx *Context) ResetQuery() {
	ctx.query = nil
}

// No need anymore, net/http checks for the Form already.
// func (ctx *Context) askParseForm() error {
// 	if ctx.request.Form == nil {
// 		if err := ctx.request.ParseForm(); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// FormValueDefault returns a single parsed form value by its "name",
// including both the URL field's query parameters and the POST or PUT form data.
//
// Returns the "def" if not found.
func (ctx *Context) FormValueDefault(name string, def string) string {
	if form, has := ctx.form(); has {
		if v := form[name]; len(v) > 0 {
			return v[0]
		}
	}
	return def
}

// FormValueDefault retruns a single parsed form value.
func FormValueDefault(r *http.Request, name string, def string, postMaxMemory int64, resetBody bool) string {
	if form, has := GetForm(r, postMaxMemory, resetBody); has {
		if v := form[name]; len(v) > 0 {
			return v[0]
		}
	}
	return def
}

// FormValue returns a single parsed form value by its "name",
// including both the URL field's query parameters and the POST or PUT form data.
func (ctx *Context) FormValue(name string) string {
	return ctx.FormValueDefault(name, "")
}

// FormValues returns the parsed form data, including both the URL
// field's query parameters and the POST or PUT form data.
//
// The default form's memory maximum size is 32MB, it can be changed by the
// `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
// NOTE: A check for nil is necessary.
func (ctx *Context) FormValues() map[string][]string {
	form, _ := ctx.form()
	return form
}

// Form contains the parsed form data, including both the URL
// field's query parameters and the POST or PUT form data.
func (ctx *Context) form() (form map[string][]string, found bool) {
	return GetForm(ctx.request, ctx.app.ConfigurationReadOnly().GetPostMaxMemory(), ctx.app.ConfigurationReadOnly().GetDisableBodyConsumptionOnUnmarshal())
}

// GetForm returns the request form (url queries, post or multipart) values.
func GetForm(r *http.Request, postMaxMemory int64, resetBody bool) (form map[string][]string, found bool) {
	/*
		net/http/request.go#1219
		for k, v := range f.Value {
			r.Form[k] = append(r.Form[k], v...)
			// r.PostForm should also be populated. See Issue 9305.
			r.PostForm[k] = append(r.PostForm[k], v...)
		}
	*/

	if form := r.Form; len(form) > 0 {
		return form, true
	}

	if form := r.PostForm; len(form) > 0 {
		return form, true
	}

	if m := r.MultipartForm; m != nil {
		if len(m.Value) > 0 {
			return m.Value, true
		}
	}

	if resetBody {
		// on POST, PUT and PATCH it will read the form values from request body otherwise from URL queries.
		if m := r.Method; m == "POST" || m == "PUT" || m == "PATCH" {
			body, restoreBody, err := GetBody(r, resetBody)
			if err != nil {
				return nil, false
			}
			setBody(r, body)    // so the ctx.request.Body works
			defer restoreBody() // so the next GetForm calls work.

			// r.Body = io.NopCloser(io.TeeReader(r.Body, buf))
		} else {
			resetBody = false
		}
	}

	// ParseMultipartForm calls `request.ParseForm` automatically
	// therefore we don't need to call it here, although it doesn't hurt.
	// After one call to ParseMultipartForm or ParseForm,
	// subsequent calls have no effect, are idempotent.
	err := r.ParseMultipartForm(postMaxMemory)
	// if resetBody {
	// 	r.Body = io.NopCloser(bytes.NewBuffer(bodyCopy))
	// }
	if err != nil && err != http.ErrNotMultipart {
		return nil, false
	}

	if form := r.Form; len(form) > 0 {
		return form, true
	}

	if form := r.PostForm; len(form) > 0 {
		return form, true
	}

	if m := r.MultipartForm; m != nil {
		if len(m.Value) > 0 {
			return m.Value, true
		}
	}

	return nil, false
}

// PostValues returns all the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name" as a string slice.
//
// The default form's memory maximum size is 32MB, it can be changed by the
// `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
//
// In addition, it reports whether the form was empty
// or when the "name" does not exist
// or whether the available values are empty.
// It strips any empty key-values from the slice before return.
//
// Look ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
// See `PostValueMany` method too.
func (ctx *Context) PostValues(name string) ([]string, error) {
	_, ok := ctx.form()
	if !ok {
		if !ctx.app.ConfigurationReadOnly().GetFireEmptyFormError() {
			return nil, nil
		}

		return nil, ErrEmptyForm // empty form.
	}

	values, ok := ctx.request.PostForm[name]
	if !ok {
		return nil, ErrNotFound // field does not exist
	}

	if len(values) == 0 ||
		// Fast check for its first empty value (see below).
		strings.TrimSpace(values[0]) == "" {
		return nil, fmt.Errorf("%w: %s", ErrEmptyFormField, name)
	}

	for _, value := range values {
		if value == "" { // if at least one empty value, then perform the strip from the beginning.
			result := make([]string, 0, len(values))
			for _, value := range values {
				if strings.TrimSpace(value) != "" {
					result = append(result, value) // we store the value as it is, not space-trimmed.
				}
			}

			if len(result) == 0 {
				return nil, fmt.Errorf("%w: %s", ErrEmptyFormField, name)
			}

			return result, nil
		}
	}

	return values, nil
}

// PostValueMany is like `PostValues` method, it returns the post data of a given key.
// In addition to `PostValues` though, the returned value is a single string
// separated by commas on multiple values.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueMany(name string) (string, error) {
	values, err := ctx.PostValues(name)
	if err != nil || len(values) == 0 {
		return "", err
	}

	return strings.Join(values, ","), nil
}

// PostValueDefault returns the last parsed form data from POST, PATCH,
// or PUT body parameters based on a "name".
//
// If not found then "def" is returned instead.
func (ctx *Context) PostValueDefault(name string, def string) string {
	values, err := ctx.PostValues(name)
	if err != nil || len(values) == 0 {
		return def // it returns "def" even if it's empty here.
	}

	return values[len(values)-1]
}

// PostValueString same as `PostValue` method but it reports
// an error if the value with key equals to "name" does not exist.
func (ctx *Context) PostValueString(name string) (string, error) {
	values, err := ctx.PostValues(name)
	if err != nil {
		return "", err
	}

	if len(values) == 0 { // just in case.
		return "", ErrEmptyForm
	}

	return values[len(values)-1], nil
}

// PostValue returns the last parsed form data from POST, PATCH,
// or PUT body parameters based on a "name".
//
// See `PostValueMany` too.
func (ctx *Context) PostValue(name string) string {
	return ctx.PostValueDefault(name, "")
}

// PostValueTrim returns the last parsed form data from POST, PATCH,
// or PUT body parameters based on a "name",  without trailing spaces.
func (ctx *Context) PostValueTrim(name string) string {
	return strings.TrimSpace(ctx.PostValue(name))
}

// PostValueUint returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as unassigned number.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueUint(name string) (uint, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseUint(value)
}

// PostValueUint returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as unassigned number.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueUint8(name string) (uint8, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseUint8(value)
}

// PostValueUint returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as unassigned number.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueUint16(name string) (uint16, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseUint16(value)
}

// PostValueUint returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as unassigned number.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueUint32(name string) (uint32, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseUint32(value)
}

// PostValueUint returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as unassigned number.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueUint64(name string) (uint64, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseUint64(value)
}

// PostValueInt returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as signed number.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueInt(name string) (int, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseInt(value)
}

// PostValueIntDefault same as PostValueInt but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueIntDefault(name string, def int) int {
	value, err := ctx.PostValueInt(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueInt8 returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as int8.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueInt8(name string) (int8, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseInt8(value)
}

// PostValueInt8Default same as PostValueInt8 but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueInt8Default(name string, def int8) int8 {
	value, err := ctx.PostValueInt8(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueInt16 returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as int16.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueInt16(name string) (int16, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseInt16(value)
}

// PostValueInt16Default same as PostValueInt16 but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueInt16Default(name string, def int16) int16 {
	value, err := ctx.PostValueInt16(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueInt32 returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as int32.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueInt32(name string) (int32, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseInt32(value)
}

// PostValueInt32Default same as PostValueInt32 but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueInt32Default(name string, def int32) int32 {
	value, err := ctx.PostValueInt32(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueInt64 returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as int64.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueInt64(name string) (int64, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseInt64(value)
}

// PostValueInt64Default same as PostValueInt64 but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueInt64Default(name string, def int64) int64 {
	value, err := ctx.PostValueInt64(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueFloat32 returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as float32.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueFloat32(name string) (float32, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseFloat32(value)
}

// PostValueFloat32Default same as PostValueFloat32 but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueFloat32Default(name string, def float32) float32 {
	value, err := ctx.PostValueFloat32(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueFloat64 returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as float64.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueFloat64(name string) (float64, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseFloat64(value)
}

// PostValueFloat64Default same as PostValueFloat64 but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueFloat64Default(name string, def float64) float64 {
	value, err := ctx.PostValueFloat64(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueComplex64 returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as complex64.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueComplex64(name string) (complex64, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseComplex64(value)
}

// PostValueComplex64Default same as PostValueComplex64 but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueComplex64Default(name string, def complex64) complex64 {
	value, err := ctx.PostValueComplex64(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueComplex128 returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as complex128.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueComplex128(name string) (complex128, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseComplex128(value)
}

// PostValueComplex128Default same as PostValueComplex128 but if errored it returns
// the given "def" default value.
func (ctx *Context) PostValueComplex128Default(name string, def complex128) complex128 {
	value, err := ctx.PostValueComplex128(name)
	if err != nil {
		return def
	}

	return value
}

// PostValueBool returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as bool.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueBool(name string) (bool, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return false, err
	}

	return strParseBool(value)
}

// PostValueWeekday returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as time.Weekday.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueWeekday(name string) (time.Weekday, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return 0, err
	}

	return strParseWeekday(value)
}

// PostValueTime returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as time.Time with the given "layout".
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueTime(layout, name string) (time.Time, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return time.Time{}, err
	}

	return strParseTime(layout, value)
}

// PostValueSimpleDate returns the last parsed form data matches the given "name" key
// from POST, PATCH, or PUT body request parameters as time.Time with "2006/01/02"
// or "2006-01-02" time layout.
//
// See ErrEmptyForm, ErrNotFound and ErrEmptyFormField respectfully.
func (ctx *Context) PostValueSimpleDate(name string) (time.Time, error) {
	value, err := ctx.PostValueString(name)
	if err != nil {
		return time.Time{}, err
	}

	return strParseSimpleDate(value)
}

// FormFile returns the first uploaded file that received from the client.
//
// The default form's memory maximum size is 32MB, it can be changed by the
// `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/file-server/upload-file
func (ctx *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	// we don't have access to see if the request is body stream
	// and then the ParseMultipartForm can be useless
	// here but do it in order to apply the post limit,
	// the internal request.FormFile will not do it if that's filled
	// and it's not a stream body.
	if err := ctx.request.ParseMultipartForm(ctx.app.ConfigurationReadOnly().GetPostMaxMemory()); err != nil {
		return nil, nil, err
	}

	return ctx.request.FormFile(key)
}

// FormFiles same as FormFile but may return multiple file inputs based on a key, e.g. "files[]".
func (ctx *Context) FormFiles(key string, before ...func(*Context, *multipart.FileHeader) bool) (files []multipart.File, headers []*multipart.FileHeader, err error) {
	err = ctx.request.ParseMultipartForm(ctx.app.ConfigurationReadOnly().GetPostMaxMemory())
	if err != nil {
		return
	}

	if ctx.request.MultipartForm != nil {
		fhs := ctx.request.MultipartForm.File
		if n := len(fhs); n > 0 {
			files = make([]multipart.File, 0, n)
			headers = make([]*multipart.FileHeader, 0, n)

		innerLoop:
			for _, header := range fhs[key] {
				header.Filename = filepath.Base(header.Filename)

				for _, b := range before {
					if !b(ctx, header) {
						continue innerLoop
					}
				}

				file, fErr := header.Open()
				if fErr != nil { // exit on first error but return the succeed.
					return files, headers, fErr
				}

				files = append(files, file)
				headers = append(headers, header)
			}
		}

		return
	}

	return nil, nil, http.ErrMissingFile
}

var (
	// ValidFileNameRegexp is used to validate the user input by using a regular expression.
	// See `Context.UploadFormFiles` method.
	ValidFilenameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
	// ValidExtensionRegexp acts as an allowlist of valid extensions. It's optional. Defaults to nil (all file extensions are allowed to be uploaded).
	// See `Context.UploadFormFiles` method.
	ValidExtensionRegexp *regexp.Regexp
)

// SafeFilename returns a safe filename based on the given name.
//   - Using filepath.Base and filepath.ToSlash: This ensures that only the base file name is used, without any directory components,
//     and converts all separators to slashes. This is a good practice to prevent directory traversal.
//   - Regular Expression for Filenames: The ValidFilenameRegexp ensures that filenames are restricted to a safe character set.
//     This helps prevent the use of special characters that could lead to path traversal or other types of injection attacks.
//   - Extension Validation: If you have a ValidExtensionRegexp, it would further ensure that the file has an expected and safe extension, which is another good practice.
//   - Canonical Path Check: By evaluating symlinks and ensuring that the destination path starts with the canonical destination directory, you’re adding.
//
// It returns the safe prefix directory (destination directory), the safe filename, a boolean indicating whether the filename is safe, and an error if any.
func SafeFilename(prefixDir string, name string) (string, string, bool, error) {
	// Security fix for go < 1.17.5:
	// Reported by Kirill Efimov (snyk.io) through security reports.
	filename := filepath.Base(filepath.ToSlash(name))

	// CWE-99.

	// Sanitize the user input by using a regular expression
	// and an allowlist of valid extensions
	isValidFilename := ValidFilenameRegexp.MatchString(filename)
	if !isValidFilename {
		// Reject the input as it is invalid or unsafe.
		return prefixDir, name, false, nil
	}

	if ValidExtensionRegexp != nil && !ValidExtensionRegexp.MatchString(filename) {
		// Reject the input as it is invalid or unsafe.
		return prefixDir, name, false, nil
	}

	var destPath string
	if prefixDir != "" {
		// Join the sanitized input with the destination directory.
		destPath = filepath.Join(prefixDir, filename)

		// Get the canonical path of the destination directory.
		canonicalDestDir, err := filepath.EvalSymlinks(prefixDir) // the prefix dir should exists.
		if err != nil {
			return prefixDir, name, false, fmt.Errorf("dest directory: %s: eval symlinks: %w", prefixDir, err)
		}

		// Check if the destination path is within the destination directory.
		if !strings.HasPrefix(destPath, canonicalDestDir) {
			// Reject the input as it is a path traversal attempt.
			return prefixDir, name, false, nil
		}
	}

	return destPath, filename, true, nil
}

// UploadFormFiles uploads any received file(s) from the client
// to the system physical location "destDirectory".
//
// The second optional argument "before" gives caller the chance to
// modify or cancel the *miltipart.FileHeader before saving to the disk,
// it can be used to change a file's name based on the current request,
// all FileHeader's options can be changed. You can ignore it if
// you don't need to use this capability before saving a file to the disk.
//
// Note that it doesn't check if request body streamed.
//
// Returns the copied length as int64 and
// a not nil error if at least one new file
// can't be created due to the operating system's permissions or
// http.ErrMissingFile if no file received.
//
// If you want to receive & accept files and manage them manually you can use the `context#FormFile`
// instead and create a copy function that suits your needs or use the `SaveFormFile` method,
// the below is for generic usage.
//
// The default form's memory maximum size is 32MB, it can be changed by
// the `WithPostMaxMemory` configurator or by `SetMaxRequestBodySize` or
// by the `LimitRequestBodySize` middleware (depends the use case).
//
// See `FormFile` and `FormFiles` to a more controlled way to receive a file.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/file-server/upload-files
func (ctx *Context) UploadFormFiles(destDirectory string, before ...func(*Context, *multipart.FileHeader) bool) (uploaded []*multipart.FileHeader, n int64, err error) {
	err = ctx.request.ParseMultipartForm(ctx.app.ConfigurationReadOnly().GetPostMaxMemory())
	if err != nil {
		return nil, 0, err
	}

	if ctx.request.MultipartForm != nil {
		if fhs := ctx.request.MultipartForm.File; fhs != nil {
			for _, files := range fhs {
			innerLoop:
				for _, file := range files {
					for _, b := range before {
						if !b(ctx, file) {
							continue innerLoop
						}
					}

					destPath, filename, ok, err := SafeFilename(destDirectory, file.Filename)
					if err != nil {
						return nil, 0, err
					}
					if !ok {
						continue
					}

					file.Filename = filename

					n0, err0 := ctx.SaveFormFile(file, destPath)
					if err0 != nil {
						return nil, 0, err0
					}
					n += n0

					uploaded = append(uploaded, file)
				}
			}
			return uploaded, n, nil
		}
	}

	return nil, 0, http.ErrMissingFile
}

// SaveFormFile saves a result of `FormFile` to the "dest" disk full path (directory + filename).
// See `FormFile` and `UploadFormFiles` too.
func (ctx *Context) SaveFormFile(fh *multipart.FileHeader, dest string) (int64, error) {
	src, err := fh.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()

	out, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	return io.Copy(out, src)
}

// AbsoluteURI parses the "s" and returns its absolute URI form.
func (ctx *Context) AbsoluteURI(s string) string {
	if s == "" {
		return ""
	}

	userInfo := ""
	if s[0] == '@' {
		endUserInfoIdx := strings.IndexByte(s, '/')
		if endUserInfoIdx > 0 && len(s) > endUserInfoIdx {
			userInfo = s[1:endUserInfoIdx] + "@"
			s = s[endUserInfoIdx:]
		}
	}

	if s[0] == '/' {
		scheme := ctx.request.URL.Scheme
		if scheme == "" {
			if ctx.request.TLS != nil {
				scheme = "https:"
			} else {
				scheme = "http:"
			}
		}

		host := ctx.Host()

		return scheme + "//" + userInfo + host + path.Clean(s)
	}

	if u, err := url.Parse(s); err == nil {
		r := ctx.request

		if u.Scheme == "" && u.Host == "" {
			oldpath := r.URL.Path
			if oldpath == "" {
				oldpath = "/"
			}

			if s == "" || s[0] != '/' {
				olddir, _ := path.Split(oldpath)
				s = olddir + s
			}

			var query string
			if i := strings.Index(s, "?"); i != -1 {
				s, query = s[:i], s[i:]
			}

			// clean up but preserve trailing slash
			trailing := strings.HasSuffix(s, "/")
			s = path.Clean(s)
			if trailing && !strings.HasSuffix(s, "/") {
				s += "/"
			}
			s += query
		}
	}

	return s
}

// Redirect sends a redirect response to the client
// of an absolute or relative target URL.
// It accepts 2 input arguments, a string and an optional integer.
// The first parameter is the target url to redirect.
// The second one is the HTTP status code should be sent
// among redirection response,
// If the second parameter is missing, then it defaults to 302 (StatusFound).
// It can be set to 301 (Permant redirect), StatusTemporaryRedirect(307)
// or 303 (StatusSeeOther) if POST method.
func (ctx *Context) Redirect(urlToRedirect string, statusHeader ...int) {
	ctx.StopExecution()
	// get the previous status code given by the end-developer.
	status := ctx.GetStatusCode()
	if status < 300 { // the previous is not a RCF-valid redirect status.
		status = 0
	}

	if len(statusHeader) > 0 {
		// check if status code is passed via receivers.
		if s := statusHeader[0]; s > 0 {
			status = s
		}
	}
	if status == 0 {
		// if status remains zero then default it.
		// a 'temporary-redirect-like' which works better than for our purpose
		status = http.StatusFound
	}

	http.Redirect(ctx.writer, ctx.request, urlToRedirect, status)
}

//  +------------------------------------------------------------+
//  | Body Readers                                               |
//  +------------------------------------------------------------+

// SetMaxRequestBodySize sets a limit to the request body size
// should be called before reading the request body from the client.
func (ctx *Context) SetMaxRequestBodySize(limitOverBytes int64) {
	ctx.request.Body = http.MaxBytesReader(ctx.writer, ctx.request.Body, limitOverBytes)
}

var emptyFunc = func() {}

// GetBody reads and returns the request body.
func GetBody(r *http.Request, resetBody bool) ([]byte, func(), error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, err
	}

	if resetBody {
		// * remember, Request.Body has no Bytes(), we have to consume them first
		// and after re-set them to the body, this is the only solution.
		return data, func() {
			setBody(r, data)
		}, nil
	}

	return data, emptyFunc, nil
}

func setBody(r *http.Request, data []byte) {
	r.Body = io.NopCloser(bytes.NewBuffer(data))
}

const disableRequestBodyConsumptionContextKey = "iris.request.body.record"

// RecordRequestBody same as the Application's DisableBodyConsumptionOnUnmarshal
// configuration field but acts only for the current request.
// It makes the request body readable more than once.
func (ctx *Context) RecordRequestBody(b bool) {
	ctx.values.Set(disableRequestBodyConsumptionContextKey, b)
}

// IsRecordingBody reports whether the request body can be readen multiple times.
func (ctx *Context) IsRecordingBody() bool {
	if ctx.app.ConfigurationReadOnly().GetDisableBodyConsumptionOnUnmarshal() {
		return true
	}

	value, _ := ctx.values.GetBool(disableRequestBodyConsumptionContextKey)
	return value
}

// GetBody reads and returns the request body.
// The default behavior for the http request reader is to consume the data readen
// but you can change that behavior by passing the `WithoutBodyConsumptionOnUnmarshal` Iris option
// or by calling the `RecordRequestBody` method.
//
// However, whenever you can use the `ctx.Request().Body` instead.
func (ctx *Context) GetBody() ([]byte, error) {
	body, release, err := GetBody(ctx.request, ctx.IsRecordingBody())
	if err != nil {
		return nil, err
	}
	release()
	return body, nil
}

// Validator is the validator for request body on Context methods such as
// ReadJSON, ReadMsgPack, ReadXML, ReadYAML, ReadForm, ReadQuery, ReadBody and e.t.c.
type Validator interface {
	Struct(interface{}) error
	// If community asks for more than a struct validation on JSON, XML, MsgPack, Form, Query and e.t.c
	// then we should add more methods here, alternative approach would be to have a
	// `Validator:Validate(interface{}) error` and a map[reflect.Kind]Validator instead.
}

// UnmarshalBody reads the request's body and binds it to a value or pointer of any type
// Examples of usage: context.ReadJSON, context.ReadXML.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-custom-via-unmarshaler/main.go
func (ctx *Context) UnmarshalBody(outPtr interface{}, unmarshaler Unmarshaler) error {
	if ctx.request.Body == nil {
		return fmt.Errorf("unmarshal: empty body: %w", ErrNotFound)
	}

	rawData, err := ctx.GetBody()
	if err != nil {
		return err
	}

	if decoderWithCtx, ok := outPtr.(BodyDecoderWithContext); ok {
		return decoderWithCtx.DecodeContext(ctx.request.Context(), rawData)
	}

	// check if the v contains its own decode
	// in this case the v should be a pointer also,
	// but this is up to the user's custom Decode implementation*
	//
	// See 'BodyDecoder' for more.
	if decoder, isDecoder := outPtr.(BodyDecoder); isDecoder {
		return decoder.Decode(rawData)
	}

	// // check if v is already a pointer, if yes then pass as it's
	// if reflect.TypeOf(v).Kind() == reflect.Ptr {
	// 	return unmarshaler.Unmarshal(rawData, v)
	// } <- no need for that, ReadJSON is documented enough to receive a pointer,
	// we don't need to reduce the performance here by using the reflect.TypeOf method.

	// f the v doesn't contains a self-body decoder use the custom unmarshaler to bind the body.
	err = unmarshaler.Unmarshal(rawData, outPtr)
	if err != nil {
		return err
	}

	return ctx.app.Validate(outPtr)
}

// JSONReader holds the JSON decode options of the `Context.ReadJSON, ReadBody` methods.
type JSONReader struct { // Note(@kataras): struct instead of optional funcs to keep consistently with the encoder options.
	// DisallowUnknownFields causes the json decoder to return an error when the destination
	// is a struct and the input contains object keys which do not match any
	// non-ignored, exported fields in the destination.
	DisallowUnknownFields bool
	// If set to true then a bit faster json decoder is used instead,
	// note that if this is true then it overrides
	// the Application's EnableOptimizations configuration field.
	Optimize bool
	// This field only applies to the ReadJSONStream.
	// The Optimize field has no effect when this is true.
	// If set to true the request body stream MUST start with a `[`
	// and end with `]` literals, example:
	//  [
	//   {"username":"john"},
	//   {"username": "makis"},
	//   {"username": "george"}
	//  ]
	// Defaults to false: decodes a json object one by one, example:
	//  {"username":"john"}
	//  {"username": "makis"}
	//  {"username": "george"}
	ArrayStream bool
}

var ReadJSON = func(ctx *Context, outPtr interface{}, opts ...JSONReader) error {
	var body io.Reader

	if ctx.IsRecordingBody() {
		data, err := io.ReadAll(ctx.request.Body)
		if err != nil {
			return err
		}
		setBody(ctx.request, data)
		body = bytes.NewReader(data)
	} else {
		body = ctx.request.Body
	}

	decoder := json.NewDecoder(body)
	// decoder := gojson.NewDecoder(ctx.Request().Body)
	if len(opts) > 0 {
		options := opts[0]

		if options.DisallowUnknownFields {
			decoder.DisallowUnknownFields()
		}
	}

	if err := decoder.Decode(&outPtr); err != nil {
		return err
	}

	return ctx.app.Validate(outPtr)
}

// ReadJSON reads JSON from request's body and binds it to a value of any json-valid type.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-json/main.go
func (ctx *Context) ReadJSON(outPtr interface{}, opts ...JSONReader) error {
	return ReadJSON(ctx, outPtr, opts...)
}

// ReadJSONStream is an alternative of ReadJSON which can reduce the memory load
// by reading only one json object every time.
// It buffers just the content required for a single json object instead of the entire string,
// and discards that once it reaches an end of value that can be decoded into the provided struct
// inside the onDecode's DecodeFunc.
//
// It accepts a function which accepts the json Decode function and returns an error.
// The second variadic argument is optional and can be used to customize the decoder even further.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-json-stream/main.go
func (ctx *Context) ReadJSONStream(onDecode func(DecodeFunc) error, opts ...JSONReader) error {
	decoder := json.NewDecoder(ctx.request.Body)

	if len(opts) > 0 && opts[0].ArrayStream {
		_, err := decoder.Token() // read open bracket.
		if err != nil {
			return err
		}

		for decoder.More() { // hile the array contains values.
			if err = onDecode(decoder.Decode); err != nil {
				return err
			}
		}

		_, err = decoder.Token() // read closing bracket.
		return err
	}

	// while the array contains values
	for decoder.More() {
		if err := onDecode(decoder.Decode); err != nil {
			return err
		}
	}

	return nil
}

// ReadXML reads XML from request's body and binds it to a value of any xml-valid type.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-xml/main.go
func (ctx *Context) ReadXML(outPtr interface{}) error {
	return ctx.UnmarshalBody(outPtr, UnmarshalerFunc(xml.Unmarshal))
}

// ReadYAML reads YAML from request's body and binds it to the "outPtr" value.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-yaml/main.go
func (ctx *Context) ReadYAML(outPtr interface{}) error {
	return ctx.UnmarshalBody(outPtr, UnmarshalerFunc(yaml.Unmarshal))
}

var (
	// IsErrEmptyJSON reports whether the given "err" is caused by a
	// Client.ReadJSON call when the request body was empty or
	// didn't start with { or [.
	IsErrEmptyJSON = func(err error) bool {
		if err == nil {
			return false
		}

		if errors.Is(err, io.EOF) {
			return true
		}

		if v, ok := err.(*json.SyntaxError); ok {
			// standard go json encoder error.
			return v.Offset == 0 && v.Error() == "unexpected end of JSON input"
		}

		errMsg := err.Error()
		// 3rd party pacakges:
		return strings.Contains(errMsg, "readObjectStart: expect {") || strings.Contains(errMsg, "readArrayStart: expect [")
	}

	// IsErrPath can be used at `context#ReadForm` and `context#ReadQuery`.
	// It reports whether the incoming error
	// can be ignored when server allows unknown post values to be sent by the client.
	//
	// A shortcut for the `schema#IsErrPath`.
	IsErrPath = schema.IsErrPath

	// IsErrPathCRSFToken reports whether the given "err" is caused
	// by unknown key error on "csrf.token". See `context#ReadForm` for more.
	IsErrPathCRSFToken = func(err error) bool {
		if err == nil || CSRFTokenFormKey == "" {
			return false
		}

		if m, ok := err.(schema.MultiError); ok {
			if csrfErr, hasCSRFToken := m[CSRFTokenFormKey]; hasCSRFToken {
				_, is := csrfErr.(schema.UnknownKeyError)
				return is

			}
		}

		return false
	}

	// ErrEmptyForm is returned by
	// - `context#ReadForm`
	// - `context#ReadQuery`
	// - `context#ReadBody`
	// when the request data (form, query and body respectfully) is empty.
	ErrEmptyForm = errors.New("empty form")

	// ErrEmptyFormField reports whether a specific field exists but it's empty.
	// Usage: errors.Is(err, ErrEmptyFormField)
	// See postValue method. It's only returned on parsed post value methods.
	ErrEmptyFormField = errors.New("empty form field")

	// ConnectionCloseErrorSubstr if at least one of the given
	// substrings are found in a net.OpError:os.SyscallError error type
	// on `IsErrConnectionReset` then the function will report true.
	ConnectionCloseErrorSubstr = []string{
		"broken pipe",
		"connection reset by peer",
	}

	// IsErrConnectionClosed reports whether the given "err"
	// is caused because of a broken connection.
	IsErrConnectionClosed = func(err error) bool {
		if err == nil {
			return false
		}

		if opErr, ok := err.(*net.OpError); ok {
			if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
				errStr := strings.ToLower(syscallErr.Error())
				for _, s := range ConnectionCloseErrorSubstr {
					if strings.Contains(errStr, s) {
						return true
					}
				}
			}
		}

		return false
	}
)

// CSRFTokenFormKey the CSRF token key of the form data.
//
// See ReadForm method for more.
const CSRFTokenFormKey = "csrf.token"

// ReadForm binds the request body of a form to the "formObject".
// It supports any kind of type, including custom structs.
// It will return nothing if request data are empty.
// The struct field tag is "form".
// Note that it will return nil error on empty form data if `Configuration.FireEmptyFormError`
// is false (as defaulted) in this case the caller should check the pointer to
// see if something was actually binded.
//
// If a client sent an unknown field, this method will return an error,
// in order to ignore that error use the `err != nil && !iris.IsErrPath(err)`.
//
// As of 15 Aug 2022, ReadForm does not return an error over unknown CSRF token form key,
// to change this behavior globally, set the `context.CSRFTokenFormKey` to an empty value.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-form/main.go
func (ctx *Context) ReadForm(formObject interface{}) error {
	values := ctx.FormValues()
	if len(values) == 0 {
		if ctx.app.ConfigurationReadOnly().GetFireEmptyFormError() {
			return ErrEmptyForm
		}
		return nil
	}

	err := schema.DecodeForm(values, formObject)
	if err != nil && !IsErrPathCRSFToken(err) {
		return err
	}

	return ctx.app.Validate(formObject)
}

type (
	// MultipartRelated is the result of the context.ReadMultipartRelated method.
	MultipartRelated struct {
		// ContentIDs keeps an ordered list of all the
		// content-ids of the multipart related request.
		ContentIDs []string
		// Contents keeps each part's information by Content-ID.
		// Contents holds each of the multipart/related part's data.
		Contents map[string]MultipartRelatedContent
	}

	// MultipartRelatedContent holds a multipart/related part's id, header and body.
	MultipartRelatedContent struct {
		// ID holds the Content-ID.
		ID string
		// Headers holds the part's request headers.
		Headers map[string][]string
		// Body holds the part's body.
		Body []byte
	}
)

// ReadMultipartRelated returns a structure which contain
// information about each part (id, headers, body).
//
// Read more at: https://www.ietf.org/rfc/rfc2387.txt.
//
// Example request (2387/5.2 Text/X-Okie):
// Content-Type: Multipart/Related; boundary=example-2;
// start="<950118.AEBH@XIson.com>"
// type="Text/x-Okie"
//
// --example-2
// Content-Type: Text/x-Okie; charset=iso-8859-1;
// declaration="<950118.AEB0@XIson.com>"
// Content-ID: <950118.AEBH@XIson.com>
// Content-Description: Document
//
// {doc}
// This picture was taken by an automatic camera mounted ...
// {image file=cid:950118.AECB@XIson.com}
// {para}
// Now this is an enlargement of the area ...
// {image file=cid:950118:AFDH@XIson.com}
// {/doc}
// --example-2
// Content-Type: image/jpeg
// Content-ID: <950118.AFDH@XIson.com>
// Content-Transfer-Encoding: BASE64
// Content-Description: Picture A
//
// [encoded jpeg image]
// --example-2
// Content-Type: image/jpeg
// Content-ID: <950118.AECB@XIson.com>
// Content-Transfer-Encoding: BASE64
// Content-Description: Picture B
//
// [encoded jpeg image]
// --example-2--
func (ctx *Context) ReadMultipartRelated() (MultipartRelated, error) {
	contentType, params, err := mime.ParseMediaType(ctx.GetHeader(ContentTypeHeaderKey))
	if err != nil {
		return MultipartRelated{}, err
	}

	if !strings.HasPrefix(contentType, ContentMultipartRelatedHeaderValue) {
		return MultipartRelated{}, ErrEmptyForm
	}

	var (
		contentIDs []string
		contents   = make(map[string]MultipartRelatedContent)
	)

	if ctx.IsRecordingBody() {
		// * remember, Request.Body has no Bytes(), we have to consume them first
		// and after re-set them to the body, this is the only solution.
		body, restoreBody, err := GetBody(ctx.request, true)
		if err != nil {
			return MultipartRelated{}, fmt.Errorf("multipart related: body copy because of iris.Configuration.DisableBodyConsumptionOnUnmarshal: %w", err)
		}
		setBody(ctx.request, body) // so the ctx.request.Body works
		defer restoreBody()        // so the next ctx.GetBody calls work.
	}

	multipartReader := multipart.NewReader(ctx.request.Body, params["boundary"])
	for {
		part, err := multipartReader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}

			return MultipartRelated{}, fmt.Errorf("multipart related: next part: %w", err)
		}
		defer part.Close()

		b, err := io.ReadAll(part)
		if err != nil {
			return MultipartRelated{}, fmt.Errorf("multipart related: next part: read: %w", err)
		}

		contentID := part.Header.Get("Content-ID")
		contentIDs = append(contentIDs, contentID)
		contents[contentID] = MultipartRelatedContent{ // replace if same Content-ID appears, which it shouldn't.
			ID:      contentID,
			Headers: http.Header(part.Header),
			Body:    b,
		}
	}

	if len(contents) != len(contentIDs) {
		contentIDs = distinctStrings(contentIDs)
	}

	result := MultipartRelated{
		ContentIDs: contentIDs,
		Contents:   contents,
	}
	return result, nil
}

func distinctStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, val := range values {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	return result
}

// ReadQuery binds URL Query to "ptr". The struct field tag is "url".
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-query/main.go
func (ctx *Context) ReadQuery(ptr interface{}) error {
	values := ctx.getQuery()
	if len(values) == 0 {
		if ctx.app.ConfigurationReadOnly().GetFireEmptyFormError() {
			return ErrEmptyForm
		}
		return nil
	}

	err := schema.DecodeQuery(values, ptr)
	if err != nil {
		return err
	}

	return ctx.app.Validate(ptr)
}

// ReadHeaders binds request headers to "ptr". The struct field tag is "header".
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-headers/main.go
func (ctx *Context) ReadHeaders(ptr interface{}) error {
	err := schema.DecodeHeaders(ctx.request.Header, ptr)
	if err != nil {
		return err
	}

	return ctx.app.Validate(ptr)
}

// ReadParams binds URI Dynamic Path Parameters to "ptr". The struct field tag is "param".
//
// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-params/main.go
func (ctx *Context) ReadParams(ptr interface{}) error {
	n := ctx.params.Len()
	if n == 0 {
		return nil
	}

	values := make(map[string][]string, n)
	ctx.params.Visit(func(key string, value string) {
		// []string on path parameter, e.g.
		// /.../{tail:path}
		// Tail []string `param:"tail"`
		values[key] = strings.Split(value, "/")
	})

	err := schema.DecodeParams(values, ptr)
	if err != nil {
		return err
	}

	return ctx.app.Validate(ptr)
}

// ReadURL is a shortcut of ReadParams and ReadQuery.
// It binds dynamic path parameters and URL query parameters
// to the "ptr" pointer struct value.
// The struct fields may contain "url" or "param" binding tags.
// If a validator exists then it validates the result too.
func (ctx *Context) ReadURL(ptr interface{}) error {
	values := make(map[string][]string, ctx.params.Len())
	ctx.params.Visit(func(key string, value string) {
		values[key] = strings.Split(value, "/")
	})

	for k, v := range ctx.getQuery() {
		values[k] = append(values[k], v...)
	}

	// Decode using all available binding tags (url, header, param).
	err := schema.Decode(values, ptr)
	if err != nil {
		return err
	}

	return ctx.app.Validate(ptr)
}

// ReadProtobuf binds the body to the "ptr" of a proto Message and returns any error.
// Look `ReadJSONProtobuf` too.
func (ctx *Context) ReadProtobuf(ptr proto.Message) error {
	rawData, err := ctx.GetBody()
	if err != nil {
		return err
	}

	return proto.Unmarshal(rawData, ptr)
}

// ProtoUnmarshalOptions is a type alias for protojson.UnmarshalOptions.
type ProtoUnmarshalOptions = protojson.UnmarshalOptions

var defaultProtobufUnmarshalOptions ProtoUnmarshalOptions

// ReadJSONProtobuf reads a JSON body request into the given "ptr" proto.Message.
// Look `ReadProtobuf` too.
func (ctx *Context) ReadJSONProtobuf(ptr proto.Message, opts ...ProtoUnmarshalOptions) error {
	rawData, err := ctx.GetBody()
	if err != nil {
		return err
	}

	opt := defaultProtobufUnmarshalOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	return opt.Unmarshal(rawData, ptr)
}

// ReadMsgPack binds the request body of msgpack format to the "ptr" and returns any error.
func (ctx *Context) ReadMsgPack(ptr interface{}) error {
	rawData, err := ctx.GetBody()
	if err != nil {
		return err
	}

	err = msgpack.Unmarshal(rawData, ptr)
	if err != nil {
		return err
	}

	return ctx.app.Validate(ptr)
}

// ReadBody binds the request body to the "ptr" depending on the HTTP Method and the Request's Content-Type.
// If a GET method request then it reads from a form (or URL Query), otherwise
// it tries to match (depending on the request content-type) the data format e.g.
// JSON, Protobuf, MsgPack, XML, YAML, MultipartForm and binds the result to the "ptr".
// As a special case if the "ptr" was a pointer to string or []byte
// then it will bind it to the request body as it is.
func (ctx *Context) ReadBody(ptr interface{}) error {
	// If the ptr is string or byte, read the body as it's.
	switch v := ptr.(type) {
	case *string:
		b, err := ctx.GetBody()
		if err != nil {
			return err
		}

		*v = string(b)
	case *[]byte:
		b, err := ctx.GetBody()
		if err != nil {
			return err
		}

		copy(*v, b)
	}

	if ctx.Method() == http.MethodGet {
		if ctx.Request().URL.RawQuery != "" {
			// try read from query.
			return ctx.ReadQuery(ptr)
		}

		// otherwise use the ReadForm,
		// it's actually the same except
		// ReadQuery will not fire errors on:
		// 1. unknown or empty url query parameters
		// 2. empty query or form (if FireEmptyFormError is enabled).
		return ctx.ReadForm(ptr)
	}

	switch ctx.GetContentTypeRequested() {
	case ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue:
		return ctx.ReadXML(ptr)
		// "%v reflect.Indirect(reflect.ValueOf(ptr)).Interface())
	case ContentYAMLHeaderValue, ContentYAMLTextHeaderValue:
		return ctx.ReadYAML(ptr)
	case ContentFormHeaderValue, ContentFormMultipartHeaderValue:
		return ctx.ReadForm(ptr)
	case ContentMultipartRelatedHeaderValue:
		return fmt.Errorf("context: read body: cannot bind multipart/related: use ReadMultipartRelated instead")
	case ContentJSONHeaderValue:
		return ctx.ReadJSON(ptr)
	case ContentProtobufHeaderValue:
		msg, ok := ptr.(proto.Message)
		if !ok {
			return ErrContentNotSupported
		}

		return ctx.ReadProtobuf(msg)
	case ContentMsgPackHeaderValue, ContentMsgPack2HeaderValue:
		return ctx.ReadMsgPack(ptr)
	default:
		if ctx.Request().URL.RawQuery != "" {
			// try read from query.
			return ctx.ReadQuery(ptr)
		}

		// otherwise default to JSON.
		return ctx.ReadJSON(ptr)
	}
}

//  +------------------------------------------------------------+
//  | Body (raw) Writers                                         |
//  +------------------------------------------------------------+

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
//
// It reports any write errors back to the caller, Application.SetContentErrorHandler does NOT apply here
// as this is a lower-level method which must be remain as it is.
func (ctx *Context) Write(rawBody []byte) (int, error) {
	return ctx.writer.Write(rawBody)
}

// Writef formats according to a format specifier and writes to the response.
//
// Returns the number of bytes written and any write error encountered.
func (ctx *Context) Writef(format string, a ...interface{}) (n int, err error) {
	/* if len(a) == 0 {
	 	return ctx.WriteString(format)
	} ^ No, let it complain about arguments, because go test will do even if the app is running.
	Users should use WriteString instead of (format, args)
	when format may contain go-sprintf reserved chars (e.g. %).*/

	return fmt.Fprintf(ctx.writer, format, a...)
}

// WriteString writes a simple string to the response.
//
// Returns the number of bytes written and any write error encountered.
func (ctx *Context) WriteString(body string) (n int, err error) {
	return io.WriteString(ctx.writer, body)
}

const (
	// ContentTypeHeaderKey is the header key of "Content-Type".
	ContentTypeHeaderKey = "Content-Type"

	// LastModifiedHeaderKey is the header key of "Last-Modified".
	LastModifiedHeaderKey = "Last-Modified"
	// IfModifiedSinceHeaderKey is the header key of "If-Modified-Since".
	IfModifiedSinceHeaderKey = "If-Modified-Since"
	// CacheControlHeaderKey is the header key of "Cache-Control".
	CacheControlHeaderKey = "Cache-Control"
	// ETagHeaderKey is the header key of "ETag".
	ETagHeaderKey = "ETag"

	// ContentDispositionHeaderKey is the header key of "Content-Disposition".
	ContentDispositionHeaderKey = "Content-Disposition"
	// ContentLengthHeaderKey is the header key of "Content-Length"
	ContentLengthHeaderKey = "Content-Length"
	// ContentEncodingHeaderKey is the header key of "Content-Encoding".
	ContentEncodingHeaderKey = "Content-Encoding"
	// GzipHeaderValue is the header value of "gzip".
	GzipHeaderValue = "gzip"
	// AcceptEncodingHeaderKey is the header key of "Accept-Encoding".
	AcceptEncodingHeaderKey = "Accept-Encoding"
	// VaryHeaderKey is the header key of "Vary".
	VaryHeaderKey = "Vary"
)

var unixEpochTime = time.Unix(0, 0)

// IsZeroTime reports whether t is obviously unspecified (either zero or Unix()=0).
func IsZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

// ParseTime parses a time header (such as the Date: header),
// trying each forth formats (or three if Application's configuration's TimeFormat is defaulted)
// that are allowed by HTTP/1.1:
// Application's configuration's TimeFormat or/and http.TimeFormat,
// time.RFC850, and time.ANSIC.
//
// Look `context#FormatTime` for the opossite operation (Time to string).
var ParseTime = func(ctx *Context, text string) (t time.Time, err error) {
	t, err = time.Parse(ctx.Application().ConfigurationReadOnly().GetTimeFormat(), text)
	if err != nil {
		return http.ParseTime(text)
	}

	return
}

// FormatTime returns a textual representation of the time value formatted
// according to the Application's configuration's TimeFormat field
// which defines the format.
//
// Look `context#ParseTime` for the opossite operation (string to Time).
var FormatTime = func(ctx *Context, t time.Time) string {
	return t.Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
}

// SetLastModified sets the "Last-Modified" based on the "modtime" input.
// If "modtime" is zero then it does nothing.
//
// It's mostly internally on core/router and context packages.
func (ctx *Context) SetLastModified(modtime time.Time) {
	if !IsZeroTime(modtime) {
		ctx.Header(LastModifiedHeaderKey, FormatTime(ctx, modtime.UTC())) // or modtime.UTC()?
	}
}

// ErrPreconditionFailed may be returned from `Context` methods
// that has to perform one or more client side preconditions before the actual check, e.g. `CheckIfModifiedSince`.
// Usage:
// ok, err := context.CheckIfModifiedSince(modTime)
//
//	if err != nil {
//	   if errors.Is(err, context.ErrPreconditionFailed) {
//	        [handle missing client conditions,such as not valid request method...]
//	    }else {
//	        [the error is probably a time parse error...]
//	   }
//	}
var ErrPreconditionFailed = errors.New("precondition failed")

// CheckIfModifiedSince checks if the response is modified since the "modtime".
// Note that it has nothing to do with server-side caching.
// It does those checks by checking if the "If-Modified-Since" request header
// sent by client or a previous server response header
// (e.g with WriteWithExpiration or HandleDir or Favicon etc.)
// is a valid one and it's before the "modtime".
//
// A check for !modtime && err == nil is necessary to make sure that
// it's not modified since, because it may return false but without even
// had the chance to check the client-side (request) header due to some errors,
// like the HTTP Method is not "GET" or "HEAD" or if the "modtime" is zero
// or if parsing time from the header failed. See `ErrPreconditionFailed` too.
//
// It's mostly used internally, e.g. `context#WriteWithExpiration`.
func (ctx *Context) CheckIfModifiedSince(modtime time.Time) (bool, error) {
	if method := ctx.Method(); method != http.MethodGet && method != http.MethodHead {
		return false, fmt.Errorf("method: %w", ErrPreconditionFailed)
	}
	ims := ctx.GetHeader(IfModifiedSinceHeaderKey)
	if ims == "" || IsZeroTime(modtime) {
		return false, fmt.Errorf("zero time: %w", ErrPreconditionFailed)
	}
	t, err := ParseTime(ctx, ims)
	if err != nil {
		return false, err
	}
	// sub-second precision, so
	// use mtime < t+1s instead of mtime <= t to check for unmodified.
	if modtime.UTC().Before(t.Add(1 * time.Second)) {
		return false, nil
	}
	return true, nil
}

// WriteNotModified sends a 304 "Not Modified" status code to the client,
// it makes sure that the content type, the content length headers
// and any "ETag" are removed before the response sent.
//
// It's mostly used internally on core/router/fs.go and context methods.
func (ctx *Context) WriteNotModified() {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g.," Last-Modified" might be useful if the
	// response does not have an ETag field).
	h := ctx.ResponseWriter().Header()
	delete(h, ContentTypeHeaderKey)
	delete(h, ContentLengthHeaderKey)
	if h.Get(ETagHeaderKey) != "" {
		delete(h, LastModifiedHeaderKey)
	}
	ctx.StatusCode(http.StatusNotModified)
}

// WriteWithExpiration works like `Write` but it will check if a resource is modified,
// based on the "modtime" input argument,
// otherwise sends a 304 status code in order to let the client-side render the cached content.
func (ctx *Context) WriteWithExpiration(body []byte, modtime time.Time) (int, error) {
	if modified, err := ctx.CheckIfModifiedSince(modtime); !modified && err == nil {
		ctx.WriteNotModified()
		return 0, nil
	}

	ctx.SetLastModified(modtime)
	return ctx.writer.Write(body)
}

// StreamWriter registers the given stream writer for populating
// response body.
//
// Access to context's and/or its' members is forbidden from writer.
//
// This function may be used in the following cases:
//
//   - if response body is too big (more than iris.LimitRequestBodySize(if set)).
//   - if response body is streamed from slow external sources.
//   - if response body must be streamed to the client in chunks.
//     (aka `http server push`).
func (ctx *Context) StreamWriter(writer func(w io.Writer) error) error {
	cancelCtx := ctx.Request().Context()
	notifyClosed := cancelCtx.Done()

	for {
		select {
		// response writer forced to close, exit.
		case <-notifyClosed:
			return cancelCtx.Err()
		default:
			if err := writer(ctx.writer); err != nil {
				return err
			}
			ctx.writer.Flush()
		}
	}
}

//  +------------------------------------------------------------+
//  | Body Writers with compression                              |
//  +------------------------------------------------------------+

// ClientSupportsEncoding reports whether the
// client expects one of the given "encodings" compression.
//
// Note, this method just reports back the first valid encoding it sees,
// meaning that request accept-encoding offers don't matter here.
// See `CompressWriter` too.
func (ctx *Context) ClientSupportsEncoding(encodings ...string) bool {
	if len(encodings) == 0 {
		return false
	}

	if h := ctx.GetHeader(AcceptEncodingHeaderKey); h != "" {
		for _, v := range strings.Split(h, ",") {
			for _, encoding := range encodings {
				if strings.Contains(v, encoding) {
					return true
				}
			}
		}
	}

	return false
}

// CompressWriter enables or disables the compress response writer.
// if the client expects a valid compression algorithm then this
// will change the response writer to a compress writer instead.
// All future write and rich write methods will respect this option.
// Usage:
//
//	app.Use(func(ctx iris.Context){
//		err := ctx.CompressWriter(true)
//		ctx.Next()
//	})
//
// The recommendation is to compress data as much as possible and therefore to use this field,
// but some types of resources, such as jpeg images, are already compressed.
// Sometimes, using additional compression doesn't reduce payload size and
// can even make the payload longer.
func (ctx *Context) CompressWriter(enable bool) error {
	switch w := ctx.writer.(type) {
	case *CompressResponseWriter:
		if enable {
			return nil
		}

		w.Disabled = true
	case *ResponseRecorder:
		if enable {
			// If it's a recorder which already wraps the compress, exit.
			if _, ok := w.ResponseWriter.(*CompressResponseWriter); ok {
				return nil
			}

			// Keep the Recorder as ctx.writer.
			// Wrap the existing net/http response writer
			// with the compressed writer and
			// replace the recorder's response writer
			// reference with that compressed one.
			// Fixes an issue when Record is called before CompressWriter.
			cw, err := AcquireCompressResponseWriter(w.ResponseWriter, ctx.request, -1)
			if err != nil {
				return err
			}
			w.ResponseWriter = cw
		} else {
			cw, ok := w.ResponseWriter.(*CompressResponseWriter)
			if ok {
				cw.Disabled = true
			}
		}
	default:
		if !enable {
			return nil
		}

		cw, err := AcquireCompressResponseWriter(w, ctx.request, -1)
		if err != nil {
			return err
		}
		ctx.writer = cw
	}

	return nil
}

// CompressReader accepts a boolean, which, if set to true
// it wraps the request body reader with a reader which decompresses request data before read.
// If the "enable" input argument is false then the request body will reset to the default one.
//
// Useful when incoming request data are compressed.
// All future calls of `ctx.GetBody/ReadXXX/UnmarshalBody` methods will respect this option.
//
// Usage:
//
//	app.Use(func(ctx iris.Context){
//		err := ctx.CompressReader(true)
//		ctx.Next()
//	})
//
// More:
//
//	if cr, ok := ctx.Request().Body.(*CompressReader); ok {
//		cr.Src // the original request body
//	 cr.Encoding // the compression algorithm.
//	}
//
// It returns `ErrRequestNotCompressed` if client's request data are not compressed
// (or empty)
// or `ErrNotSupportedCompression` if server missing the decompression algorithm.
func (ctx *Context) CompressReader(enable bool) error {
	cr, ok := ctx.request.Body.(*CompressReader)
	if enable {
		if ok {
			// already called.
			return nil
		}

		encoding := ctx.GetHeader(ContentEncodingHeaderKey)
		if encoding == IDENTITY {
			// no transformation whatsoever, return nil error and
			// don't wrap the body reader.
			return nil
		}

		r, err := NewCompressReader(ctx.request.Body, encoding)
		if err != nil {
			return err
		}
		ctx.request.Body = r
	} else if ok {
		ctx.request.Body = cr.Src
	}

	return nil
}

//  +------------------------------------------------------------+
//  | Rich Body Content Writers/Renderers                        |
//  +------------------------------------------------------------+

// ViewEngine registers a view engine for the current chain of handlers.
// It overrides any previously registered engines, including the application's root ones.
// Note that, because performance is everything,
// the "engine" MUST be already ready-to-use,
// meaning that its `Load` method should be called once before this method call.
//
// To register a view engine per-group of groups too see `Party.RegisterView` instead.
func (ctx *Context) ViewEngine(engine ViewEngine) {
	ctx.values.Set(ctx.app.ConfigurationReadOnly().GetViewEngineContextKey(), engine)
}

// ViewLayout sets the "layout" option if and when .View
// is being called afterwards, in the same request.
// Useful when need to set or/and change a layout based on the previous handlers in the chain.
//
// Note that the 'layoutTmplFile' argument can be set to iris.NoLayout
// to disable the layout for a specific view render action,
// it disables the engine's configuration's layout property.
//
// Look .ViewData and .View too.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/view/context-view-data/
func (ctx *Context) ViewLayout(layoutTmplFile string) {
	ctx.values.Set(ctx.app.ConfigurationReadOnly().GetViewLayoutContextKey(), layoutTmplFile)
}

// ViewData saves one or more key-value pair in order to be passed if and when .View
// is being called afterwards, in the same request.
// Useful when need to set or/and change template data from previous hanadlers in the chain.
//
// If .View's "binding" argument is not nil and it's not a type of map
// then these data are being ignored, binding has the priority, so the main route's handler can still decide.
// If binding is a map or iris.Map then these data are being added to the view data
// and passed to the template.
//
// After .View, the data are not destroyed, in order to be re-used if needed (again, in the same request as everything else),
// to clear the view data, developers can call:
// ctx.Set(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey(), nil)
//
// If 'key' is empty then the value is added as it's (struct or map) and developer is unable to add other value.
//
// Look .ViewLayout and .View too.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/view/context-view-data/
func (ctx *Context) ViewData(key string, value interface{}) {
	viewDataContextKey := ctx.app.ConfigurationReadOnly().GetViewDataContextKey()
	if key == "" {
		ctx.values.Set(viewDataContextKey, value)
		return
	}

	v := ctx.values.Get(viewDataContextKey)
	if v == nil {
		ctx.values.Set(viewDataContextKey, Map{key: value})
		return
	}

	if data, ok := v.(Map); ok {
		data[key] = value
	}
}

// GetViewData returns the values registered by `context#ViewData`.
// The return value is `map[string]interface{}`, this means that
// if a custom struct registered to ViewData then this function
// will try to parse it to map, if failed then the return value is nil
// A check for nil is always a good practise if different
// kind of values or no data are registered via `ViewData`.
//
// Similarly to `viewData := ctx.Values().Get("iris.view.data")` or
// `viewData := ctx.Values().Get(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey())`.
func (ctx *Context) GetViewData() map[string]interface{} {
	if v := ctx.values.Get(ctx.app.ConfigurationReadOnly().GetViewDataContextKey()); v != nil {
		// if pure map[string]interface{}
		if viewData, ok := v.(Map); ok {
			return viewData
		}

		// if struct, convert it to map[string]interface{}
		if structs.IsStruct(v) {
			return structs.Map(v)
		}
	}

	// if no values found, then return nil
	return nil
}

// FallbackViewProvider is an interface which can be registered to the `Party.FallbackView`
// or `Context.FallbackView` methods to handle fallback views.
// See FallbackView, FallbackViewLayout and FallbackViewFunc.
type FallbackViewProvider interface {
	FallbackView(ctx *Context, err ErrViewNotExist) error
} /* Notes(@kataras): If ever requested, this fallback logic (of ctx, error) can go to all necessary methods.
   I've designed with a bit more complexity here instead of a simple filename fallback in order to give
   the freedom to the developer to do whatever he/she wants with that template/layout not exists error,
   e.g. have a list of fallbacks views to loop through until succeed or fire a different error than the default.
   We also provide some helpers for common fallback actions (FallbackView, FallbackViewLayout).
   This naming was chosen in order to be easy to follow up with the previous view-relative context features.
   Also note that here we catch a specific error, we want the developer
   to be aware of the rest template errors (e.g. when a template having parsing issues).
*/

// FallbackViewFunc is a function that can be registered
// to handle view fallbacks. It accepts the Context and
// a special error which contains information about the previous template error.
// It implements the FallbackViewProvider interface.
//
// See `Context.View` method.
type FallbackViewFunc func(ctx *Context, err ErrViewNotExist) error

// FallbackView completes the FallbackViewProvider interface.
func (fn FallbackViewFunc) FallbackView(ctx *Context, err ErrViewNotExist) error {
	return fn(ctx, err)
}

var (
	_ FallbackViewProvider = FallbackView("")
	_ FallbackViewProvider = FallbackViewLayout("")
)

// FallbackView is a helper to register a single template filename as a fallback
// when the provided tempate filename was not found.
type FallbackView string

// FallbackView completes the FallbackViewProvider interface.
func (f FallbackView) FallbackView(ctx *Context, err ErrViewNotExist) error {
	if err.IsLayout { // Not responsible to render layouts.
		return err
	}

	// ctx.StatusCode(200) // Let's keep the previous status code here, developer can change it anyways.
	return ctx.View(string(f), err.Data)
}

// FallbackViewLayout is a helper to register a single template filename as a fallback
// layout when the provided layout filename was not found.
type FallbackViewLayout string

// FallbackView completes the FallbackViewProvider interface.
func (f FallbackViewLayout) FallbackView(ctx *Context, err ErrViewNotExist) error {
	if !err.IsLayout {
		// Responsible to render layouts only.
		return err
	}

	ctx.ViewLayout(string(f))
	return ctx.View(err.Name, err.Data)
}

const fallbackViewOnce = "iris.fallback.view.once"

func (ctx *Context) fireFallbackViewOnce(err ErrViewNotExist) error {
	// Note(@kataras): this is our way to keep the same View method for
	// both fallback and normal views, remember, we export the whole
	// Context functionality to the end-developer through the fallback view provider.
	if ctx.values.Get(fallbackViewOnce) != nil {
		return err
	}

	v := ctx.values.Get(ctx.app.ConfigurationReadOnly().GetFallbackViewContextKey())
	if v == nil {
		return err
	}

	providers, ok := v.([]FallbackViewProvider)
	if !ok {
		return err
	}

	ctx.values.Set(fallbackViewOnce, struct{}{})

	var pErr error
	for _, provider := range providers {
		pErr = provider.FallbackView(ctx, err)
		if pErr != nil {
			if vErr, ok := pErr.(ErrViewNotExist); ok {
				// This fallback view does not exist or it's not responsible to handle,
				// try the next.
				pErr = vErr
				continue
			}
		}

		// If OK then we found the correct fallback.
		// If the error was a parse error and not a template not found
		// then exit and report the pErr error.
		break
	}

	return pErr
}

// FallbackView registers one or more fallback views for a template or a template layout.
// When View cannot find the given filename to execute then this "provider"
// is responsible to handle the error or render a different view.
//
// Usage:
//
//	FallbackView(iris.FallbackView("fallback.html"))
//	FallbackView(iris.FallbackViewLayout("layouts/fallback.html"))
//	OR
//	FallbackView(iris.FallbackViewFunc(ctx iris.Context, err iris.ErrViewNotExist) error {
//	  err.Name is the previous template name.
//	  err.IsLayout reports whether the failure came from the layout template.
//	  err.Data is the template data provided to the previous View call.
//	  [...custom logic e.g. ctx.View("fallback", err.Data)]
//	})
func (ctx *Context) FallbackView(providers ...FallbackViewProvider) {
	key := ctx.app.ConfigurationReadOnly().GetFallbackViewContextKey()
	if key == "" {
		return
	}

	v := ctx.values.Get(key)
	if v == nil {
		ctx.values.Set(key, providers)
		return
	}

	// Can register more than one.
	storedProviders, ok := v.([]FallbackViewProvider)
	if !ok {
		return
	}

	storedProviders = append(storedProviders, providers...)
	ctx.values.Set(key, storedProviders)
}

// View renders a template based on the registered view engine(s).
// First argument accepts the filename, relative to the view engine's Directory and Extension,
// i.e: if directory is "./templates" and want to render the "./templates/users/index.html"
// then you pass the "users/index.html" as the filename argument.
//
// The second optional argument can receive a single "view model".
// If "optionalViewModel" exists, even if it's nil, overrides any previous `ViewData` calls.
// If second argument is missing then binds the data through previous `ViewData` calls (e.g. middleware).
//
// Look .ViewData and .ViewLayout too.
//
// Examples: https://github.com/kataras/iris/tree/main/_examples/view
func (ctx *Context) View(filename string, optionalViewModel ...interface{}) error {
	ctx.ContentType(ContentHTMLHeaderValue)

	err := ctx.renderView(filename, optionalViewModel...)
	if err != nil {
		if errNotExists, ok := err.(ErrViewNotExist); ok {
			err = ctx.fireFallbackViewOnce(errNotExists)
		}
	}

	if err != nil {
		if ctx.IsDebug() {
			// send the error back to the client, when debug mode.
			ctx.StopWithError(http.StatusInternalServerError, err)
		} else {
			ctx.SetErrPrivate(err)
			ctx.StopWithStatus(http.StatusInternalServerError)
		}
	}

	return err
}

func (ctx *Context) renderView(filename string, optionalViewModel ...interface{}) error {
	cfg := ctx.app.ConfigurationReadOnly()
	layout := ctx.values.GetString(cfg.GetViewLayoutContextKey())

	var bindingData interface{}
	if len(optionalViewModel) > 0 /* Don't do it: can break a lot of servers: && optionalViewModel[0] != nil */ {
		// a nil can override the existing data or model sent by `ViewData`.
		bindingData = optionalViewModel[0]
	} else {
		bindingData = ctx.values.Get(cfg.GetViewDataContextKey())
	}

	if key := cfg.GetViewEngineContextKey(); key != "" {
		if engineV := ctx.values.Get(key); engineV != nil {
			if engine, ok := engineV.(ViewEngine); ok {
				return engine.ExecuteWriter(ctx, filename, layout, bindingData)
			}
		}
	}

	return ctx.app.View(ctx, filename, layout, bindingData)
}

const (
	// ContentBinaryHeaderValue header value for binary data.
	ContentBinaryHeaderValue = "application/octet-stream"
	// ContentWebassemblyHeaderValue header value for web assembly files.
	ContentWebassemblyHeaderValue = "application/wasm"
	// ContentHTMLHeaderValue is the  string of text/html response header's content type value.
	ContentHTMLHeaderValue = "text/html"
	// ContentJSONHeaderValue header value for JSON data.
	ContentJSONHeaderValue = "application/json"
	// ContentJSONProblemHeaderValue header value for JSON API problem error.
	// Read more at: https://tools.ietf.org/html/rfc7807
	ContentJSONProblemHeaderValue = "application/problem+json"
	// ContentXMLProblemHeaderValue header value for XML API problem error.
	// Read more at: https://tools.ietf.org/html/rfc7807
	ContentXMLProblemHeaderValue = "application/problem+xml"
	// ContentJavascriptHeaderValue header value for JSONP & Javascript data.
	ContentJavascriptHeaderValue = "text/javascript"
	// ContentTextHeaderValue header value for Text data.
	ContentTextHeaderValue = "text/plain"
	// ContentXMLHeaderValue header value for XML data.
	ContentXMLHeaderValue = "text/xml"
	// ContentXMLUnreadableHeaderValue obselete header value for XML.
	ContentXMLUnreadableHeaderValue = "application/xml"
	// ContentMarkdownHeaderValue custom key/content type, the real is the text/html.
	ContentMarkdownHeaderValue = "text/markdown"
	// ContentYAMLHeaderValue header value for YAML data.
	ContentYAMLHeaderValue = "application/x-yaml"
	// ContentYAMLTextHeaderValue header value for YAML plain text.
	ContentYAMLTextHeaderValue = "text/yaml"
	// ContentProtobufHeaderValue header value for Protobuf messages data.
	ContentProtobufHeaderValue = "application/x-protobuf"
	// ContentMsgPackHeaderValue header value for MsgPack data.
	ContentMsgPackHeaderValue = "application/msgpack"
	// ContentMsgPack2HeaderValue alternative header value for MsgPack data.
	ContentMsgPack2HeaderValue = "application/x-msgpack"
	// ContentFormHeaderValue header value for post form data.
	ContentFormHeaderValue = "application/x-www-form-urlencoded"
	// ContentFormMultipartHeaderValue header value for post multipart form data.
	ContentFormMultipartHeaderValue = "multipart/form-data"
	// ContentMultipartRelatedHeaderValue header value for multipart related data.
	ContentMultipartRelatedHeaderValue = "multipart/related"
	// ContentGRPCHeaderValue Content-Type header value for gRPC.
	ContentGRPCHeaderValue = "application/grpc"
)

// Binary writes out the raw bytes as binary data.
func (ctx *Context) Binary(data []byte) (int, error) {
	ctx.ContentType(ContentBinaryHeaderValue)
	return ctx.Write(data)
}

// Text writes out a string as plain text.
func (ctx *Context) Text(format string, args ...interface{}) (int, error) {
	ctx.ContentType(ContentTextHeaderValue)
	return ctx.Writef(format, args...)
}

// HTML writes out a string as text/html.
func (ctx *Context) HTML(format string, args ...interface{}) (int, error) {
	ctx.ContentType(ContentHTMLHeaderValue)
	return ctx.Writef(format, args...)
}

// ProtoMarshalOptions is a type alias for protojson.MarshalOptions.
type ProtoMarshalOptions = protojson.MarshalOptions

// JSON contains the options for the JSON (Context's) Renderer.
type JSON struct {
	// content-specific
	UnescapeHTML bool   `yaml:"UnescapeHTML"`
	Indent       string `yaml:"Indent"`
	Prefix       string `yaml:"Prefix"`
	ASCII        bool   `yaml:"ASCII"`  // if true writes with unicode to ASCII content.
	Secure       bool   `yaml:"Secure"` // if true then it prepends a "while(1);" when Go slice (to JSON Array) value.
	// proto.Message specific marshal options.
	Proto ProtoMarshalOptions `yaml:"ProtoMarshalOptions"`
	// If true and json writing failed then the error handler is skipped
	// and it just returns to the caller.
	//
	// See StopWithJSON and x/errors package.
	OmitErrorHandler bool `yaml:"OmitErrorHandler"`
}

// DefaultJSONOptions is the optional settings that are being used
// inside `Context.JSON`.
var DefaultJSONOptions = JSON{}

// IsDefault reports whether this JSON options structure holds the default values.
func (j *JSON) IsDefault() bool {
	return j.UnescapeHTML == DefaultJSONOptions.UnescapeHTML &&
		j.Indent == DefaultJSONOptions.Indent &&
		j.Prefix == DefaultJSONOptions.Prefix &&
		j.ASCII == DefaultJSONOptions.ASCII &&
		j.Secure == DefaultJSONOptions.Secure &&
		j.Proto == DefaultJSONOptions.Proto
}

// JSONP contains the options for the JSONP (Context's) Renderer.
type JSONP struct {
	// content-specific
	Indent           string
	Callback         string
	OmitErrorHandler bool // See JSON.OmitErrorHandler.
}

// XML contains the options for the XML (Context's) Renderer.
type XML struct {
	// content-specific
	Indent           string
	Prefix           string
	OmitErrorHandler bool // See JSON.OmitErrorHandler.
}

// Markdown contains the options for the Markdown (Context's) Renderer.
type Markdown struct {
	// content-specific
	Sanitize         bool
	OmitErrorHandler bool // See JSON.OmitErrorHandler.
	//
	// Library-specific.
	// E.g. Flags: html.CommonFlags | html.HrefTargetBlank
	RenderOptions html.RendererOptions
}

var (
	newLineB = []byte("\n")
	// the html codes for unescaping.
	ltHex = []byte("\\u003c")
	lt    = []byte("<")

	gtHex = []byte("\\u003e")
	gt    = []byte(">")

	andHex = []byte("\\u0026")
	and    = []byte("&")

	// secure JSON.
	jsonArrayPrefix  = []byte("[")
	jsonArraySuffix  = []byte("]")
	secureJSONPrefix = []byte("while(1);")
)

func (ctx *Context) handleSpecialJSONResponseValue(v interface{}, options *JSON) (bool, int, error) {
	if ctx.app.ConfigurationReadOnly().GetEnableProtoJSON() {
		if m, ok := v.(proto.Message); ok {
			protoJSON := ProtoMarshalOptions{}
			if options != nil {
				protoJSON = options.Proto
			}

			result, err := protoJSON.Marshal(m)
			if err != nil {
				return true, 0, err
			}

			n, err := ctx.writer.Write(result)
			return true, n, err
		}
	}

	if ctx.app.ConfigurationReadOnly().GetEnableEasyJSON() {
		if easyObject, ok := v.(easyjson.Marshaler); ok {
			noEscapeHTML := false
			if options != nil {
				noEscapeHTML = !options.UnescapeHTML
			}
			jw := jwriter.Writer{NoEscapeHTML: noEscapeHTML}
			easyObject.MarshalEasyJSON(&jw)

			n, err := jw.DumpTo(ctx.writer)
			return true, n, err
		}
	}

	return false, 0, nil
}

// WriteJSON marshals the given interface object and writes the JSON response to the 'writer'.
var WriteJSON = func(ctx *Context, v interface{}, options *JSON) error {
	if !options.Secure && !options.ASCII && options.Prefix == "" {
		// jsoniterConfig := jsoniter.Config{
		// 	EscapeHTML:    !options.UnescapeHTML,
		// 	IndentionStep: 4,
		// }.Froze()
		// enc := jsoniterConfig.NewEncoder(ctx.writer)
		// err = enc.Encode(v)
		//
		// enc := gojson.NewEncoder(ctx.writer)
		// enc.SetEscapeHTML(!options.UnescapeHTML)
		// enc.SetIndent(options.Prefix, options.Indent)
		// err = enc.EncodeContext(ctx, v)
		enc := json.NewEncoder(ctx.writer)
		enc.SetEscapeHTML(!options.UnescapeHTML)
		enc.SetIndent(options.Prefix, options.Indent)

		return enc.Encode(v)
	}

	var (
		result []byte
		err    error
	)

	if indent := options.Indent; indent != "" {
		result, err = json.MarshalIndent(v, "", indent)
		result = append(result, newLineB...)
	} else {
		result, err = json.Marshal(v)
	}

	if err != nil {
		return err
	}

	prependSecure := false
	if options.Secure {
		if bytes.HasPrefix(result, jsonArrayPrefix) {
			if options.Indent == "" {
				prependSecure = bytes.HasSuffix(result, jsonArraySuffix)
			} else {
				prependSecure = bytes.HasSuffix(bytes.TrimRightFunc(result, func(r rune) bool {
					return r == '\n' || r == '\r'
				}), jsonArraySuffix)
			}
		}
	}

	if options.UnescapeHTML {
		result = bytes.ReplaceAll(result, ltHex, lt)
		result = bytes.ReplaceAll(result, gtHex, gt)
		result = bytes.ReplaceAll(result, andHex, and)
	}

	if prependSecure {
		result = append(secureJSONPrefix, result...)
	}

	if options.ASCII {
		if len(result) > 0 {
			buf := new(bytes.Buffer)
			for _, s := range string(result) {
				char := string(s)
				if s >= 128 {
					char = fmt.Sprintf("\\u%04x", int64(s))
				}
				buf.WriteString(char)
			}

			result = buf.Bytes()
		}
	}

	if prefix := options.Prefix; prefix != "" {
		result = append(stringToBytes(prefix), result...)
	}

	_, err = ctx.Write(result)
	return err
}

// See https://golang.org/src/strings/builder.go#L45
// func bytesToString(b []byte) string {
// 	return unsafe.String(unsafe.SliceData(b), len(b))
// }

func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

type (
	// ErrorHandler describes a context error handler which applies on
	// JSON, JSONP, Protobuf, MsgPack, XML, YAML and Markdown write errors.
	//
	// It does NOT modify or handle the result of Context.GetErr at all,
	// use a custom middleware instead if you want to handle the handler-provided custom errors (Context.SetErr)
	//
	// An ErrorHandler can be registered once via Application.SetErrorHandler method to override the default behavior.
	// The default behavior is to simply send status internal code error
	// without a body back to the client.
	//
	// See Application.SetContextErrorHandler for more.
	ErrorHandler interface {
		HandleContextError(ctx *Context, err error)
	}
	// ErrorHandlerFunc a function shortcut for ErrorHandler interface.
	ErrorHandlerFunc func(ctx *Context, err error)
)

// HandleContextError completes the ErrorHandler interface.
func (h ErrorHandlerFunc) HandleContextError(ctx *Context, err error) {
	h(ctx, err)
}

func (ctx *Context) handleContextError(err error) {
	if err == nil {
		return
	}

	if errHandler := ctx.app.GetContextErrorHandler(); errHandler != nil {
		errHandler.HandleContextError(ctx, err)
	} else {
		if ctx.IsDebug() {
			ctx.app.Logger().Error(err)
		}
		ctx.StatusCode(http.StatusInternalServerError)
	}

	// keep the error non nil so the caller has control over further actions.
}

// Render writes the response headers and calls the given renderer "r" to render data.
// This method can be used while migrating from other frameworks.
func (ctx *Context) Render(statusCode int, r interface {
	// Render should push data with custom content type to the client.
	Render(http.ResponseWriter) error
	// WriteContentType writes custom content type to the response.
	WriteContentType(w http.ResponseWriter)
}) {
	ctx.StatusCode(statusCode)

	if statusCode >= 100 && statusCode <= 199 || statusCode == http.StatusNoContent || statusCode == http.StatusNotModified {
		r.WriteContentType(ctx.writer)
		return
	}

	if err := r.Render(ctx.writer); err != nil {
		ctx.StopWithError(http.StatusInternalServerError, err)
	}
}

// Component is the interface which all components must implement.
// A component is a struct which can be rendered to a writer.
// It's being used by the `Context.RenderComponent` method.
// An example of compatible Component is a templ.Component.
type Component interface {
	Render(context.Context, io.Writer) error
}

// RenderComponent renders a component to the client.
// It sets the "Content-Type" header to "text/html; charset=utf-8".
// It reports any component render errors back to the caller.
// Look the Application.SetContextErrorHandler to override the
// default status code 500 with a custom error response.
func (ctx *Context) RenderComponent(component Component) error {
	ctx.ContentType("text/html; charset=utf-8")
	err := component.Render(ctx.Request().Context(), ctx.ResponseWriter())
	if err != nil {
		ctx.handleContextError(err)
	}
	return err
}

// JSON marshals the given "v" value to JSON and writes the response to the client.
// Look the Configuration.EnableProtoJSON and EnableEasyJSON too.
//
// It reports any JSON parser or write errors back to the caller.
// Look the Application.SetContextErrorHandler to override the
// default status code 500 with a custom error response.
//
// Customize the behavior of every `Context.JSON“ can be achieved
// by modifying the package-level `WriteJSON` function on program initilization.
func (ctx *Context) JSON(v interface{}, opts ...JSON) (err error) {
	var options *JSON
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		// If no options are given safely read the already-initialized value.
		options = &DefaultJSONOptions
	}

	if err = ctx.writeJSON(v, options); err != nil {
		// if no options are given or OmitErrorHandler is false
		// then call the error handler (which may lead to a cycle if the error handler fails to write JSON too).
		if !options.OmitErrorHandler {
			ctx.handleContextError(err)
		}
	}

	return
}

func (ctx *Context) writeJSON(v interface{}, options *JSON) error {
	ctx.ContentType(ContentJSONHeaderValue)

	// After content type given and before everything else, try handle proto or easyjson, no matter the performance mode.
	if handled, _, err := ctx.handleSpecialJSONResponseValue(v, options); handled {
		return err
	}

	return WriteJSON(ctx, v, options)
}

var finishCallbackB = []byte(");")

// WriteJSONP marshals the given interface object and writes the JSONP response to the writer.
var WriteJSONP = func(ctx *Context, v interface{}, options *JSONP) (err error) {
	if callback := options.Callback; callback != "" {
		_, err = ctx.Write(stringToBytes(callback + "("))
		if err != nil {
			return err
		}
		defer func() {
			if err == nil {
				ctx.Write(finishCallbackB)
			}
		}()
	}

	err = WriteJSON(ctx, v, &JSON{
		Indent:           options.Indent,
		OmitErrorHandler: options.OmitErrorHandler,
	})
	return err
}

// DefaultJSONPOptions is the optional settings that are being used
// inside `ctx.JSONP`.
var DefaultJSONPOptions = JSONP{}

// JSONP marshals the given "v" value to JSON and sends the response to the client.
//
// It reports any JSON parser or write errors back to the caller.
// Look the Application.SetContextErrorHandler to override the
// default status code 500 with a custom error response.
func (ctx *Context) JSONP(v interface{}, opts ...JSONP) (err error) {
	var options *JSONP
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		options = &DefaultJSONPOptions
	}

	ctx.ContentType(ContentJavascriptHeaderValue)
	if err = WriteJSONP(ctx, v, options); err != nil {
		if !options.OmitErrorHandler {
			ctx.handleContextError(err)
		}
	}

	return
}

type xmlMapEntry struct {
	XMLName xml.Name
	Value   interface{} `xml:",chardata"`
}

// XMLMap wraps a map[string]interface{} to compatible xml marshaler,
// in order to be able to render maps as XML on the `Context.XML` method.
//
// Example: `Context.XML(XMLMap("Root", map[string]interface{}{...})`.
func XMLMap(elementName string, v Map) xml.Marshaler {
	return xmlMap{
		entries:     v,
		elementName: elementName,
	}
}

type xmlMap struct {
	entries     Map
	elementName string
}

// MarshalXML marshals a map to XML.
func (m xmlMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(m.entries) == 0 {
		return nil
	}

	start.Name = xml.Name{Local: m.elementName}
	err := e.EncodeToken(start)
	if err != nil {
		return err
	}

	for k, v := range m.entries {
		err = e.Encode(xmlMapEntry{XMLName: xml.Name{Local: k}, Value: v})
		if err != nil {
			return err
		}
	}

	return e.EncodeToken(start.End())
}

// WriteXML marshals the given interface object and writes the XML response to the writer.
var WriteXML = func(ctx *Context, v interface{}, options *XML) error {
	if prefix := options.Prefix; prefix != "" {
		_, err := ctx.Write(stringToBytes(prefix))
		if err != nil {
			return err
		}
	}

	encoder := xml.NewEncoder(ctx.writer)
	encoder.Indent("", options.Indent)
	if err := encoder.Encode(v); err != nil {
		return err
	}

	return encoder.Flush()
}

// DefaultXMLOptions is the optional settings that are being used
// from `ctx.XML`.
var DefaultXMLOptions = XML{}

// XML marshals the given interface object and writes the XML response to the client.
// To render maps as XML see the `XMLMap` package-level function.
//
// It reports any XML parser or write errors back to the caller.
// Look the Application.SetContextErrorHandler to override the
// default status code 500 with a custom error response.
func (ctx *Context) XML(v interface{}, opts ...XML) (err error) {
	var options *XML
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		options = &DefaultXMLOptions
	}

	ctx.ContentType(ContentXMLHeaderValue)
	if err = WriteXML(ctx, v, options); err != nil {
		if !options.OmitErrorHandler {
			ctx.handleContextError(err)
		}
	}

	return
}

// Problem writes a JSON or XML problem response.
// Order of Problem fields are not always rendered the same.
//
// Behaves exactly like the `Context.JSON` method
// but with default ProblemOptions.JSON indent of " " and
// a response content type of "application/problem+json" instead.
//
// Use the options.RenderXML and XML fields to change this behavior and
// send a response of content type "application/problem+xml" instead.
//
// Read more at: https://github.com/kataras/iris/blob/main/_examples/routing/http-errors.
func (ctx *Context) Problem(v interface{}, opts ...ProblemOptions) error {
	options := DefaultProblemOptions
	if len(opts) > 0 {
		options = opts[0]
		// Currently apply only if custom options passsed, otherwise,
		// with the current settings, it's not required.
		// This may change in the future though.
		options.Apply(ctx)
	}

	if p, ok := v.(Problem); ok {
		// if !p.Validate() {
		// 	ctx.StatusCode(http.StatusInternalServerError)
		// 	return ErrNotValidProblem
		// }
		p.updateURIsToAbs(ctx)
		code, _ := p.getStatus()
		if code == 0 { // get the current status code and set it to the problem.
			code = ctx.GetStatusCode()
			ctx.StatusCode(code)
		} else {
			// send the problem's status code
			ctx.StatusCode(code)
		}

		if options.RenderXML {
			ctx.contentTypeOnce(ContentXMLProblemHeaderValue, "")
			// Problem is an xml Marshaler already, don't use `XMLMap`.
			return ctx.XML(v, options.XML)
		}
	}

	ctx.contentTypeOnce(ContentJSONProblemHeaderValue, "")
	return ctx.writeJSON(v, &options.JSON)
}

var sanitizer = bluemonday.UGCPolicy()

// WriteMarkdown parses the markdown to html and writes these contents to the writer.
var WriteMarkdown = func(ctx *Context, markdownB []byte, options *Markdown) error {
	out := markdown.NormalizeNewlines(markdownB)

	renderer := html.NewRenderer(options.RenderOptions)
	doc := markdown.Parse(out, nil)
	out = markdown.Render(doc, renderer)
	if options.Sanitize {
		out = sanitizer.SanitizeBytes(out)
	}

	_, err := ctx.Write(out)
	return err
}

// DefaultMarkdownOptions is the optional settings that are being used
// from `WriteMarkdown` and `ctx.Markdown`.
var DefaultMarkdownOptions = Markdown{}

// Markdown parses the markdown to html and renders its result to the client.
//
// It reports any Markdown parser or write errors back to the caller.
// Look the Application.SetContextErrorHandler to override the
// default status code 500 with a custom error response.
func (ctx *Context) Markdown(markdownB []byte, opts ...Markdown) (err error) {
	var options *Markdown
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		options = &DefaultMarkdownOptions
	}

	ctx.ContentType(ContentHTMLHeaderValue)
	if err = WriteMarkdown(ctx, markdownB, options); err != nil {
		if !options.OmitErrorHandler {
			ctx.handleContextError(err)
		}
	}

	return
}

// WriteYAML sends YAML response to the client.
var WriteYAML = func(ctx *Context, v interface{}, indentSpace int) error {
	encoder := yaml.NewEncoder(ctx.writer)
	encoder.SetIndent(indentSpace)

	if err := encoder.Encode(v); err != nil {
		return err
	}

	return encoder.Close()
}

// YAML marshals the given "v" value using the yaml marshaler and writes the result to the client.
//
// It reports any YAML parser or write errors back to the caller.
// Look the Application.SetContextErrorHandler to override the
// default status code 500 with a custom error response.
func (ctx *Context) YAML(v interface{}) error {
	ctx.ContentType(ContentYAMLHeaderValue)

	err := WriteYAML(ctx, v, 0)
	if err != nil {
		ctx.handleContextError(err)
		return err
	}

	return nil
}

// TextYAML calls the Context.YAML method but with the text/yaml content type instead.
func (ctx *Context) TextYAML(v interface{}) error {
	ctx.ContentType(ContentYAMLTextHeaderValue)

	err := WriteYAML(ctx, v, 4)
	if err != nil {
		ctx.handleContextError(err)
		return err
	}

	return nil
}

// Protobuf marshals the given "v" value of proto Message and writes its result to the client.
//
// It reports any protobuf parser or write errors back to the caller.
// Look the Application.SetContextErrorHandler to override the
// default status code 500 with a custom error response.
func (ctx *Context) Protobuf(v proto.Message) (int, error) {
	out, err := proto.Marshal(v)
	if err != nil {
		ctx.handleContextError(err)
		return 0, err
	}

	ctx.ContentType(ContentProtobufHeaderValue)
	n, err := ctx.Write(out)
	if err != nil {
		ctx.handleContextError(err)
	}

	return n, err
}

// MsgPack marshals the given "v" value of msgpack format and writes its result to the client.
//
// It reports any message pack or write errors back to the caller.
// Look the Application.SetContextErrorHandler to override the
// default status code 500 with a custom error response.
func (ctx *Context) MsgPack(v interface{}) (int, error) {
	out, err := msgpack.Marshal(v)
	if err != nil {
		ctx.handleContextError(err)
	}

	ctx.ContentType(ContentMsgPackHeaderValue)
	n, err := ctx.Write(out)
	if err != nil {
		ctx.handleContextError(err)
	}

	return n, err
}

//  +-----------------------------------------------------------------------+
//  | Content Νegotiation                                                   |
//  | https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation |                                       |
//  +-----------------------------------------------------------------------+

// ErrContentNotSupported returns from the `Negotiate` method
// when server responds with 406.
var ErrContentNotSupported = errors.New("unsupported content")

// ContentSelector is the interface which structs can implement
// to manually choose a content based on the negotiated mime (content type).
// It can be passed to the `Context.Negotiate` method.
//
// See the `N` struct too.
type ContentSelector interface {
	SelectContent(mime string) interface{}
}

// ContentNegotiator is the interface which structs can implement
// to override the `Context.Negotiate` default implementation and
// manually respond to the client based on a manuall call of `Context.Negotiation().Build()`
// to get the final negotiated mime and charset.
// It can be passed to the `Context.Negotiate` method.
type ContentNegotiator interface {
	// mime and charset can be retrieved by:
	// mime, charset := Context.Negotiation().Build()
	// Pass this method to `Context.Negotiate` method
	// to write custom content.
	// Overriding the existing behavior of Context.Negotiate for selecting values based on
	// content types, although it can accept any custom mime type with []byte.
	// Content type is already set.
	// Use it with caution, 99.9% you don't need this but it's here for extreme cases.
	Negotiate(ctx *Context) (int, error)
}

// N is a struct which can be passed on the `Context.Negotiate` method.
// It contains fields which should be filled based on the `Context.Negotiation()`
// server side values. If no matched mime then its "Other" field will be sent,
// which should be a string or []byte.
// It completes the `ContentSelector` interface.
type N struct {
	Text, HTML string
	Markdown   []byte
	Binary     []byte

	JSON     interface{}
	Problem  Problem
	JSONP    interface{}
	XML      interface{}
	YAML     interface{}
	Protobuf interface{}
	MsgPack  interface{}

	Other []byte // custom content types.
}

var _ ContentSelector = N{}

// SelectContent returns a content based on the matched negotiated "mime".
func (n N) SelectContent(mime string) interface{} {
	switch mime {
	case ContentTextHeaderValue:
		return n.Text
	case ContentHTMLHeaderValue:
		return n.HTML
	case ContentMarkdownHeaderValue:
		return n.Markdown
	case ContentBinaryHeaderValue:
		return n.Binary
	case ContentJSONHeaderValue:
		return n.JSON
	case ContentJSONProblemHeaderValue:
		return n.Problem
	case ContentJavascriptHeaderValue:
		return n.JSONP
	case ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue:
		return n.XML
	case ContentYAMLHeaderValue:
		return n.YAML
	case ContentProtobufHeaderValue:
		return n.Protobuf
	case ContentMsgPackHeaderValue, ContentMsgPack2HeaderValue:
		return n.MsgPack
	default:
		return n.Other
	}
}

const negotiationContextKey = "iris.negotiation_builder"

// Negotiation creates once and returns the negotiation builder
// to build server-side available prioritized content
// for specific content type(s), charset(s) and encoding algorithm(s).
//
// See `Negotiate` method too.
func (ctx *Context) Negotiation() *NegotiationBuilder {
	if n := ctx.values.Get(negotiationContextKey); n != nil {
		return n.(*NegotiationBuilder)
	}

	acceptBuilder := NegotiationAcceptBuilder{}
	acceptBuilder.accept = parseHeader(ctx.GetHeader("Accept"))
	acceptBuilder.charset = parseHeader(ctx.GetHeader("Accept-Charset"))

	n := &NegotiationBuilder{Accept: acceptBuilder}

	ctx.values.Set(negotiationContextKey, n)

	return n
}

func parseHeader(headerValue string) []string {
	in := strings.Split(headerValue, ",")
	out := make([]string, 0, len(in))

	for _, value := range in {
		// remove any spaces and quality values such as ;q=0.8.
		v := strings.TrimSpace(strings.Split(value, ";")[0])
		if v != "" {
			out = append(out, v)
		}
	}

	return out
}

// Negotiate used for serving different representations of a resource at the same URI.
//
// The "v" can be a single `N` struct value.
// The "v" can be any value completes the `ContentSelector` interface.
// The "v" can be any value completes the `ContentNegotiator` interface.
// The "v" can be any value of struct(JSON, JSONP, XML, YAML, Protobuf, MsgPack) or
// string(TEXT, HTML) or []byte(Markdown, Binary) or []byte with any matched mime type.
//
// If the "v" is nil, the `Context.Negotitation()` builder's
// content will be used instead, otherwise "v" overrides builder's content
// (server mime types are still retrieved by its registered, supported, mime list)
//
// Set mime type priorities by `Negotiation().JSON().XML().HTML()...`.
// Set charset priorities by `Negotiation().Charset(...)`.
// Set encoding algorithm priorities by `Negotiation().Encoding(...)`.
// Modify the accepted by
// `Negotiation().Accept./Override()/.XML().JSON().Charset(...).Encoding(...)...`.
//
// It returns `ErrContentNotSupported` when not matched mime type(s).
//
// Resources:
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Charset
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Encoding
//
// Supports the above without quality values.
//
// Read more at: https://github.com/kataras/iris/tree/main/_examples/response-writer/content-negotiation
func (ctx *Context) Negotiate(v interface{}) (int, error) {
	contentType, charset, encoding, content := ctx.Negotiation().Build()
	if v == nil {
		v = content
	}

	if contentType == "" {
		// If the server cannot serve any matching set,
		// it SHOULD send back a 406 (Not Acceptable) error code.
		ctx.StatusCode(http.StatusNotAcceptable)
		return -1, ErrContentNotSupported
	}

	if charset == "" {
		charset = ctx.app.ConfigurationReadOnly().GetCharset()
	}

	if encoding != "" {
		ctx.CompressWriter(true)
	}

	ctx.contentTypeOnce(contentType, charset)

	if n, ok := v.(ContentNegotiator); ok {
		return n.Negotiate(ctx)
	}

	if s, ok := v.(ContentSelector); ok {
		v = s.SelectContent(contentType)
	}

	// switch v := value.(type) {
	// case []byte:
	// 	if contentType == ContentMarkdownHeaderValue {
	// 		return ctx.Markdown(v)
	// 	}

	// 	return ctx.Write(v)
	// case string:
	// 	return ctx.WriteString(v)
	// default:
	// make it switch by content-type only, but we lose custom mime types capability that way:
	//                                                 ^ solved with []byte on default case and
	//                                                 ^ N.Other and
	//                                                 ^ ContentSelector and ContentNegotiator interfaces.

	switch contentType {
	case ContentTextHeaderValue, ContentHTMLHeaderValue:
		return ctx.WriteString(v.(string))
	case ContentMarkdownHeaderValue:
		err := ctx.Markdown(v.([]byte))
		if err != nil {
			return 0, err
		}

		return ctx.writer.Written(), nil
	case ContentJSONHeaderValue:
		err := ctx.JSON(v)
		if err != nil {
			return 0, err
		}

		return ctx.writer.Written(), nil
	case ContentJSONProblemHeaderValue, ContentXMLProblemHeaderValue:
		err := ctx.Problem(v)
		if err != nil {
			return 0, err
		}

		return ctx.writer.Written(), nil
	case ContentJavascriptHeaderValue:
		err := ctx.JSONP(v)
		if err != nil {
			return 0, err
		}

		return ctx.writer.Written(), nil
	case ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue:
		err := ctx.XML(v)
		if err != nil {
			return 0, err
		}

		return ctx.writer.Written(), nil
	case ContentYAMLHeaderValue:
		err := ctx.YAML(v)
		if err != nil {
			return 0, err
		}

		return ctx.writer.Written(), nil
	case ContentYAMLTextHeaderValue:
		err := ctx.TextYAML(v)
		if err != nil {
			return 0, err
		}

		return ctx.writer.Written(), nil
	case ContentProtobufHeaderValue:
		msg, ok := v.(proto.Message)
		if !ok {
			return -1, ErrContentNotSupported
		}

		return ctx.Protobuf(msg)
	case ContentMsgPackHeaderValue, ContentMsgPack2HeaderValue:
		return ctx.MsgPack(v)
	default:
		// maybe "Other" or v is []byte or string but not a built-in framework mime,
		// for custom content types,
		// panic if not correct usage.
		switch vv := v.(type) {
		case []byte:
			return ctx.Write(vv)
		case string:
			return ctx.WriteString(vv)
		default:
			ctx.StatusCode(http.StatusNotAcceptable)
			return -1, ErrContentNotSupported
		}
	}
}

// NegotiationBuilder returns from the `Context.Negotitation`
// and can be used inside chain of handlers to build server-side
// mime type(s), charset(s) and encoding algorithm(s)
// that should match with the client's
// Accept, Accept-Charset and Accept-Encoding headers (by-default).
// To modify the client's accept use its "Accept" field
// which it's the `NegotitationAcceptBuilder`.
//
// See the `Negotiate` method too.
type NegotiationBuilder struct {
	Accept NegotiationAcceptBuilder

	mime     []string               // we need order.
	contents map[string]interface{} // map to the "mime" and content should be rendered if that mime requested.
	charset  []string
	encoding []string
}

// MIME registers a mime type and optionally the value that should be rendered
// through `Context.Negotiate` when this mime type is accepted by client.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) MIME(mime string, content interface{}) *NegotiationBuilder {
	mimes := parseHeader(mime) // if contains more than one sep by commas ",".
	if content == nil {
		n.mime = append(n.mime, mimes...)
		return n
	}

	if n.contents == nil {
		n.contents = make(map[string]interface{})
	}

	for _, m := range mimes {
		n.mime = append(n.mime, m)
		n.contents[m] = content
	}

	return n
}

// Text registers the "text/plain" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/plain" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Text(v ...string) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentTextHeaderValue, content)
}

// HTML registers the "text/html" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/html" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) HTML(v ...string) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentHTMLHeaderValue, content)
}

// Markdown registers the "text/markdown" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/markdown" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Markdown(v ...[]byte) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v
	}
	return n.MIME(ContentMarkdownHeaderValue, content)
}

// Binary registers the "application/octet-stream" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/octet-stream" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Binary(v ...[]byte) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentBinaryHeaderValue, content)
}

// JSON registers the "application/json" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/json" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) JSON(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentJSONHeaderValue, content)
}

// Problem registers the "application/problem+json" or "application/problem+xml" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/problem+json" or the "application/problem+xml" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Problem(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentJSONProblemHeaderValue+","+ContentXMLProblemHeaderValue, content)
}

// JSONP registers the "text/javascript" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "javascript/javascript" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) JSONP(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentJavascriptHeaderValue, content)
}

// XML registers the "text/xml" and "application/xml" content types and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts one of the "text/xml" or "application/xml" content types.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) XML(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentXMLHeaderValue+","+ContentXMLUnreadableHeaderValue, content)
}

// YAML registers the "application/x-yaml" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/x-yaml" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) YAML(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentYAMLHeaderValue, content)
}

// TextYAML registers the "text/yaml" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/x-yaml" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) TextYAML(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentYAMLTextHeaderValue, content)
}

// Protobuf registers the "application/x-protobuf" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/x-protobuf" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Protobuf(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentProtobufHeaderValue, content)
}

// MsgPack registers the "application/x-msgpack" and "application/msgpack" content types and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts one of the "application/x-msgpack" or "application/msgpack" content types.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) MsgPack(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentMsgPackHeaderValue+","+ContentMsgPack2HeaderValue, content)
}

// Any registers a wildcard that can match any client's accept content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Any(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME("*", content)
}

// Charset overrides the application's config's charset (which defaults to "utf-8")
// that a client should match for
// (through Accept-Charset header or custom through `NegotitationBuilder.Accept.Override().Charset(...)` call).
// Do not set it if you don't know what you're doing.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Charset(charset ...string) *NegotiationBuilder {
	n.charset = append(n.charset, charset...)
	return n
}

// Encoding registers one or more encoding algorithms by name, i.e gzip, deflate, br, snappy, s2.
// that a client should match for (through Accept-Encoding header).
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Encoding(encoding ...string) *NegotiationBuilder {
	n.encoding = append(n.encoding, encoding...)
	return n
}

// EncodingGzip registers the "gzip" encoding algorithm
// that a client should match for (through Accept-Encoding header or call of Accept.Encoding(enc)).
//
// It will make resources to served by "gzip" if Accept-Encoding contains the "gzip" as well.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) EncodingGzip() *NegotiationBuilder {
	return n.Encoding(GzipHeaderValue)
}

// Build calculates the client's and server's mime type(s), charset(s) and encoding
// and returns the final content type, charset and encoding that server should render
// to the client. It does not clear the fields, use the `Clear` method if neeeded.
//
// The returned "content" can be nil if the matched "contentType" does not provide any value,
// in that case the `Context.Negotiate(v)` must be called with a non-nil value.
func (n *NegotiationBuilder) Build() (contentType, charset, encoding string, content interface{}) {
	contentType = negotiationMatch(n.Accept.accept, n.mime)
	charset = negotiationMatch(n.Accept.charset, n.charset)
	encoding = negotiationMatch(n.Accept.encoding, n.encoding)

	if n.contents != nil {
		if data, ok := n.contents[contentType]; ok {
			content = data
		}
	}

	return
}

// Clear clears the prioritized mime type(s), charset(s) and any contents
// relative to those mime type(s).
// The "Accept" field is stay as it is, use its `Override` method
// to clear out the client's accepted mime type(s) and charset(s).
func (n *NegotiationBuilder) Clear() *NegotiationBuilder {
	n.mime = n.mime[0:0]
	n.contents = nil
	n.charset = n.charset[0:0]
	return n
}

// NegotiationAcceptBuilder builds the accepted mime types and charset
//
// and "Accept-Charset" headers respectfully.
// The default values are set by the client side, server can append or override those.
// The end result will be challenged with runtime preffered set of content types and charsets.
//
// See the `Negotiate` method too.
type NegotiationAcceptBuilder struct {
	// initialized with "Accept" request header values.
	accept []string
	// initialized with "Accept-Charset" request header. and if was empty then the
	// application's default (which defaults to utf-8).
	charset []string
	// initialized with "Accept-Encoding" request header values.
	encoding []string

	// To support override in request life cycle.
	// We need slice when data is the same format
	// for one or more mime types,
	// i.e text/xml and obselete application/xml.
	lastAccept   []string
	lastCharset  []string
	lastEncoding []string
}

// Override clears the default values for accept and accept charset.
// Returns itself.
func (n *NegotiationAcceptBuilder) Override() *NegotiationAcceptBuilder {
	// when called first.
	n.accept = n.accept[0:0]
	n.charset = n.charset[0:0]
	n.encoding = n.encoding[0:0]

	// when called after.
	if len(n.lastAccept) > 0 {
		n.accept = append(n.accept, n.lastAccept...)
		n.lastAccept = n.lastAccept[0:0]
	}

	if len(n.lastCharset) > 0 {
		n.charset = append(n.charset, n.lastCharset...)
		n.lastCharset = n.lastCharset[0:0]
	}

	if len(n.lastEncoding) > 0 {
		n.encoding = append(n.encoding, n.lastEncoding...)
		n.lastEncoding = n.lastEncoding[0:0]
	}

	return n
}

// MIME adds accepted client's mime type(s).
// Returns itself.
func (n *NegotiationAcceptBuilder) MIME(mimeType ...string) *NegotiationAcceptBuilder {
	n.lastAccept = mimeType
	n.accept = append(n.accept, mimeType...)
	return n
}

// Text adds the "text/plain" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Text() *NegotiationAcceptBuilder {
	return n.MIME(ContentTextHeaderValue)
}

// HTML adds the "text/html" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) HTML() *NegotiationAcceptBuilder {
	return n.MIME(ContentHTMLHeaderValue)
}

// Markdown adds the "text/markdown" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Markdown() *NegotiationAcceptBuilder {
	return n.MIME(ContentMarkdownHeaderValue)
}

// Binary adds the "application/octet-stream" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Binary() *NegotiationAcceptBuilder {
	return n.MIME(ContentBinaryHeaderValue)
}

// JSON adds the "application/json" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) JSON() *NegotiationAcceptBuilder {
	return n.MIME(ContentJSONHeaderValue)
}

// Problem adds the "application/problem+json" and "application/problem-xml"
// as accepted client content types.
// Returns itself.
func (n *NegotiationAcceptBuilder) Problem() *NegotiationAcceptBuilder {
	return n.MIME(ContentJSONProblemHeaderValue, ContentXMLProblemHeaderValue)
}

// JSONP adds the "text/javascript" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) JSONP() *NegotiationAcceptBuilder {
	return n.MIME(ContentJavascriptHeaderValue)
}

// XML adds the "text/xml" and "application/xml" as accepted client content types.
// Returns itself.
func (n *NegotiationAcceptBuilder) XML() *NegotiationAcceptBuilder {
	return n.MIME(ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue)
}

// YAML adds the "application/x-yaml" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) YAML() *NegotiationAcceptBuilder {
	return n.MIME(ContentYAMLHeaderValue)
}

// TextYAML adds the "text/yaml" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) TextYAML() *NegotiationAcceptBuilder {
	return n.MIME(ContentYAMLTextHeaderValue)
}

// Protobuf adds the "application/x-protobuf" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Protobuf() *NegotiationAcceptBuilder {
	return n.MIME(ContentYAMLHeaderValue)
}

// MsgPack adds the "application/msgpack" and "application/x-msgpack" as accepted client content types.
// Returns itself.
func (n *NegotiationAcceptBuilder) MsgPack() *NegotiationAcceptBuilder {
	return n.MIME(ContentYAMLHeaderValue)
}

// Charset adds one or more client accepted charsets.
// Returns itself.
func (n *NegotiationAcceptBuilder) Charset(charset ...string) *NegotiationAcceptBuilder {
	n.lastCharset = charset
	n.charset = append(n.charset, charset...)

	return n
}

// Encoding adds one or more client accepted encoding algorithms.
// Returns itself.
func (n *NegotiationAcceptBuilder) Encoding(encoding ...string) *NegotiationAcceptBuilder {
	n.lastEncoding = encoding
	n.encoding = append(n.encoding, encoding...)

	return n
}

// EncodingGzip adds the "gzip" as accepted encoding.
// Returns itself.
func (n *NegotiationAcceptBuilder) EncodingGzip() *NegotiationAcceptBuilder {
	return n.Encoding(GzipHeaderValue)
}

//  +------------------------------------------------------------+
//  | Serve files                                                |
//  +------------------------------------------------------------+

// ServeContent replies to the request using the content in the
// provided ReadSeeker. The main benefit of ServeContent over io.Copy
// is that it handles Range requests properly, sets the MIME type, and
// handles If-Match, If-Unmodified-Since, If-None-Match, If-Modified-Since,
// and If-Range requests.
//
// If the response's Content-Type header is not set, ServeContent
// first tries to deduce the type from name's file extension.
//
// The name is otherwise unused; in particular it can be empty and is
// never sent in the response.
//
// If modtime is not the zero time or Unix epoch, ServeContent
// includes it in a Last-Modified header in the response. If the
// request includes an If-Modified-Since header, ServeContent uses
// modtime to decide whether the content needs to be sent at all.
//
// The content's Seek method must work: ServeContent uses
// a seek to the end of the content to determine its size.
//
// If the caller has set w's ETag header formatted per RFC 7232, section 2.3,
// ServeContent uses it to handle requests using If-Match, If-None-Match, or If-Range.
//
// Note that *os.File implements the io.ReadSeeker interface.
// Note that compression can be registered
// through `ctx.CompressWriter(true)` or `app.Use(iris.Compression)`.
func (ctx *Context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time) {
	ctx.ServeContentWithRate(content, filename, modtime, 0, 0)
}

// rateReadSeeker is a io.ReadSeeker that is rate limited by
// the given token bucket. Each token in the bucket
// represents one byte. See "golang.org/x/time/rate" package.
type rateReadSeeker struct {
	io.ReadSeeker
	ctx     context.Context
	limiter *rate.Limiter
}

func (rs *rateReadSeeker) Read(buf []byte) (int, error) {
	n, err := rs.ReadSeeker.Read(buf)
	if n <= 0 {
		return n, err
	}
	err = rs.limiter.WaitN(rs.ctx, n)
	return n, err
}

// ServeContentWithRate same as `ServeContent` but it can throttle the speed of reading
// and though writing the "content" to the client.
func (ctx *Context) ServeContentWithRate(content io.ReadSeeker, filename string, modtime time.Time, limit float64, burst int) {
	if limit > 0 {
		content = &rateReadSeeker{
			ReadSeeker: content,
			ctx:        ctx.request.Context(),
			limiter:    rate.NewLimiter(rate.Limit(limit), burst),
		}
	}

	http.ServeContent(ctx.writer, ctx.request, filename, modtime, content)
}

// ServeFile replies to the request with the contents of the named
// file or directory.
//
// If the provided file or directory name is a relative path, it is
// interpreted relative to the current directory and may ascend to
// parent directories. If the provided name is constructed from user
// input, it should be sanitized before calling `ServeFile`.
//
// Use it when you want to serve assets like css and javascript files.
// If client should confirm and save the file use the `SendFile` instead.
// Note that compression can be registered
// through `ctx.CompressWriter(true)` or `app.Use(iris.Compression)`.
func (ctx *Context) ServeFile(filename string) error {
	return ctx.ServeFileWithRate(filename, 0, 0)
}

// ServeFileWithRate same as `ServeFile` but it can throttle the speed of reading
// and though writing the file to the client.
func (ctx *Context) ServeFileWithRate(filename string, limit float64, burst int) error {
	f, err := os.Open(filename)
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		code := http.StatusInternalServerError
		if os.IsNotExist(err) {
			code = http.StatusNotFound
		}

		if os.IsPermission(err) {
			code = http.StatusForbidden
		}

		ctx.StatusCode(code)
		return err
	}

	if st.IsDir() {
		return ctx.ServeFile(path.Join(filename, "index.html"))
	}

	ctx.ServeContentWithRate(f, st.Name(), st.ModTime(), limit, burst)
	return nil
}

// SendFile sends a file as an attachment, that is downloaded and saved locally from client.
// Note that compression can be registered
// through `ctx.CompressWriter(true)` or `app.Use(iris.Compression)`.
// Use `ServeFile` if a file should be served as a page asset instead.
func (ctx *Context) SendFile(src string, destName string) error {
	return ctx.SendFileWithRate(src, destName, 0, 0)
}

// SendFileWithRate same as `SendFile` but it can throttle the speed of reading
// and though writing the file to the client.
func (ctx *Context) SendFileWithRate(src, destName string, limit float64, burst int) error {
	if destName == "" {
		destName = filepath.Base(src)
	}

	ctx.writer.Header().Set(ContentDispositionHeaderKey, MakeDisposition(destName))
	return ctx.ServeFileWithRate(src, limit, burst)
}

// MakeDisposition generates an HTTP Content-Disposition field-value.
// Similar solution followed by: Spring(Java), Symfony(PHP) and Ruby on Rails frameworks too.
//
// Fixes CVE-2020-5398. Reported by motoyasu-saburi.
func MakeDisposition(filename string) string {
	return `attachment; filename*=UTF-8''` + url.QueryEscape(filename)
} /*
// Found at: https://stackoverflow.com/questions/53069040/checking-a-string-contains-only-ascii-characters
// A faster (better, more idiomatic) version, which avoids unnecessary rune conversions.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
func isRFC5987AttrChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		c == '!' || c == '#' || c == '$' || c == '&' || c == '+' || c == '-' ||
		c == '.' || c == '^' || c == '_' || c == '`' || c == '|' || c == '~'
}
*/

//  +------------------------------------------------------------+
//  | Cookies                                                    |
//  +------------------------------------------------------------+

// Set of Cookie actions for `CookieOption`.
const (
	OpCookieGet uint8 = iota
	OpCookieSet
	OpCookieDel
)

// CookieOption is the type of function that is accepted on
// context's methods like `SetCookieKV`, `RemoveCookie` and `SetCookie`
// as their (last) variadic input argument to amend the to-be-sent cookie.
//
// The "op" is the operation code, 0 is GET, 1 is SET and 2 is REMOVE.
type CookieOption func(ctx *Context, c *http.Cookie, op uint8)

// CookieIncluded reports whether the "cookie.Name" is in the list of "cookieNames".
// Notes:
// If "cookieNames" slice is empty then it returns true,
// If "cookie.Name" is empty then it returns false.
func CookieIncluded(cookie *http.Cookie, cookieNames []string) bool {
	if cookie.Name == "" {
		return false
	}

	if len(cookieNames) > 0 {
		for _, name := range cookieNames {
			if cookie.Name == name {
				return true
			}
		}

		return false
	}

	return true
}

// var cookieNameSanitizer = strings.NewReplacer("\n", "-", "\r", "-")
//
// func sanitizeCookieName(n string) string {
// 	return cookieNameSanitizer.Replace(n)
// }

// CookieOverride is a CookieOption which overrides the cookie explicitly to the given "cookie".
//
// Usage:
// ctx.RemoveCookie("the_cookie_name", iris.CookieOverride(&http.Cookie{Domain: "example.com"}))
func CookieOverride(cookie *http.Cookie) CookieOption { // The "Cookie" word method name is reserved as it's used as an alias.
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieGet {
			return
		}

		*cookie = *c
	}
}

// CookieDomain is a CookieOption which sets the cookie's Domain field.
// If empty then the current domain is used.
//
// Usage:
// ctx.RemoveCookie("the_cookie_name", iris.CookieDomain("example.com"))
func CookieDomain(domain string) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieGet {
			return
		}

		c.Domain = domain
	}
}

// CookieAllowReclaim accepts the Context itself.
// If set it will add the cookie to (on `CookieSet`, `CookieSetKV`, `CookieUpsert`)
// or remove the cookie from (on `CookieRemove`) the Request object too.
func CookieAllowReclaim(cookieNames ...string) CookieOption {
	return func(ctx *Context, c *http.Cookie, op uint8) {
		if op == OpCookieGet {
			return
		}

		if !CookieIncluded(c, cookieNames) {
			return
		}

		switch op {
		case OpCookieSet:
			// perform upsert on request cookies or is it too much and not worth the cost?
			ctx.Request().AddCookie(c)
		case OpCookieDel:
			cookies := ctx.Request().Cookies()
			ctx.Request().Header.Del("Cookie")
			for i, v := range cookies {
				if v.Name != c.Name {
					ctx.Request().AddCookie(cookies[i])
				}
			}
		}
	}
}

// CookieAllowSubdomains set to the Cookie Options
// in order to allow subdomains to have access to the cookies.
// It sets the cookie's Domain field (if was empty) and
// it also sets the cookie's SameSite to lax mode too.
func CookieAllowSubdomains(cookieNames ...string) CookieOption {
	return func(ctx *Context, c *http.Cookie, _ uint8) {
		if c.Domain != "" {
			return // already set.
		}

		if !CookieIncluded(c, cookieNames) {
			return
		}

		c.Domain = ctx.Domain()
		c.SameSite = http.SameSiteLaxMode // allow subdomain sharing.
	}
}

// CookieSameSite sets a same-site rule for cookies to set.
// SameSite allows a server to define a cookie attribute making it impossible for
// the browser to send this cookie along with cross-site requests. The main
// goal is to mitigate the risk of cross-origin information leakage, and provide
// some protection against cross-site request forgery attacks.
//
// See https://tools.ietf.org/html/draft-ietf-httpbis-cookie-same-site-00 for details.
func CookieSameSite(sameSite http.SameSite) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieSet {
			c.SameSite = sameSite
		}
	}
}

// CookieSecure sets the cookie's Secure option if the current request's
// connection is using TLS. See `CookieHTTPOnly` too.
func CookieSecure(ctx *Context, c *http.Cookie, op uint8) {
	if op == OpCookieSet {
		if ctx.Request().TLS != nil {
			c.Secure = true
		}
	}
}

// CookieHTTPOnly is a `CookieOption`.
// Use it to set the cookie's HttpOnly field to false or true.
// HttpOnly field defaults to true for `RemoveCookie` and `SetCookieKV`.
// See `CookieSecure` too.
func CookieHTTPOnly(httpOnly bool) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieSet {
			c.HttpOnly = httpOnly
		}
	}
}

// CookiePath is a `CookieOption`.
// Use it to change the cookie's Path field.
func CookiePath(path string) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op > OpCookieGet { // on set and remove.
			c.Path = path
		}
	}
}

// CookieCleanPath is a `CookieOption`.
// Use it to clear the cookie's Path field, exactly the same as `CookiePath("")`.
func CookieCleanPath(_ *Context, c *http.Cookie, op uint8) {
	if op > OpCookieGet {
		c.Path = ""
	}
}

// CookieExpires is a `CookieOption`.
// Use it to change the cookie's Expires and MaxAge fields by passing the lifetime of the cookie.
func CookieExpires(durFromNow time.Duration) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieSet {
			c.Expires = time.Now().Add(durFromNow)
			c.MaxAge = int(durFromNow.Seconds())
		}
	}
}

// SecureCookie should encodes and decodes
// authenticated and optionally encrypted cookie values.
// See `CookieEncoding` package-level function.
type SecureCookie interface {
	// Encode should encode the cookie value.
	// Should accept the cookie's name as its first argument
	// and as second argument the cookie value ptr.
	// Should return an encoded value or an empty one if encode operation failed.
	// Should return an error if encode operation failed.
	//
	// Note: Errors are not printed, so you have to know what you're doing,
	// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	//
	// See `Decode` too.
	Encode(cookieName string, cookieValue interface{}) (string, error)
	// Decode should decode the cookie value.
	// Should accept the cookie's name as its first argument,
	// as second argument the encoded cookie value and as third argument the decoded value ptr.
	// Should return a decoded value or an empty one if decode operation failed.
	// Should return an error if decode operation failed.
	//
	// Note: Errors are not printed, so you have to know what you're doing,
	// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	//
	// See `Encode` too.
	Decode(cookieName string, cookieValue string, cookieValuePtr interface{}) error
}

// CookieEncoding accepts a value which implements `Encode` and `Decode` methods.
// It calls its `Encode` on `Context.SetCookie, UpsertCookie, and SetCookieKV` methods.
// And on `Context.GetCookie` method it calls its `Decode`.
// If "cookieNames" slice is not empty then only cookies
// with that `Name` will be encoded on set and decoded on get, that way you can encrypt
// specific cookie names (like the session id) and let the rest of the cookies "insecure".
//
// Example: https://github.com/kataras/iris/tree/main/_examples/cookies/securecookie
func CookieEncoding(encoding SecureCookie, cookieNames ...string) CookieOption {
	if encoding == nil {
		return func(_ *Context, _ *http.Cookie, _ uint8) {}
	}

	return func(ctx *Context, c *http.Cookie, op uint8) {
		if op == OpCookieDel {
			return
		}

		if !CookieIncluded(c, cookieNames) {
			return
		}

		switch op {
		case OpCookieSet:
			// Should encode, it's a write to the client operation.
			newVal, err := encoding.Encode(c.Name, c.Value)
			if err != nil {
				ctx.Application().Logger().Error(err)
				c.Value = ""
			} else {
				c.Value = newVal
			}

			return
		case OpCookieGet:
			// Should decode, it's a read from the client operation.
			if err := encoding.Decode(c.Name, c.Value, &c.Value); err != nil {
				c.Value = ""
			}
		}
	}
}

const cookieOptionsContextKey = "iris.cookie.options"

// AddCookieOptions adds cookie options for `SetCookie`,
// `SetCookieKV, UpsertCookie` and `RemoveCookie` methods
// for the current request. It can be called from a middleware before
// cookies sent or received from the next Handler in the chain.
//
// Available builtin Cookie options are:
//   - CookieOverride
//   - CookieDomain
//   - CookieAllowReclaim
//   - CookieAllowSubdomains
//   - CookieSecure
//   - CookieHTTPOnly
//   - CookieSameSite
//   - CookiePath
//   - CookieCleanPath
//   - CookieExpires
//   - CookieEncoding
//
// Example at: https://github.com/kataras/iris/tree/main/_examples/cookies/securecookie
func (ctx *Context) AddCookieOptions(options ...CookieOption) {
	if len(options) == 0 {
		return
	}

	if v := ctx.values.Get(cookieOptionsContextKey); v != nil {
		if opts, ok := v.([]CookieOption); ok {
			options = append(opts, options...)
		}
	}

	ctx.values.Set(cookieOptionsContextKey, options)
}

func (ctx *Context) applyCookieOptions(c *http.Cookie, op uint8, override []CookieOption) {
	if v := ctx.values.Get(cookieOptionsContextKey); v != nil {
		if options, ok := v.([]CookieOption); ok {
			for _, opt := range options {
				opt(ctx, c, op)
			}
		}
	}

	// The function's ones should be called last, so they can override
	// the stored ones (i.e by a prior middleware).
	for _, opt := range override {
		opt(ctx, c, op)
	}
}

// ClearCookieOptions clears any previously registered cookie options.
// See `AddCookieOptions` too.
func (ctx *Context) ClearCookieOptions() {
	ctx.values.Remove(cookieOptionsContextKey)
}

// SetCookie adds a cookie.
// Use of the "options" is not required, they can be used to amend the "cookie".
//
// Example: https://github.com/kataras/iris/tree/main/_examples/cookies/basic
func (ctx *Context) SetCookie(cookie *http.Cookie, options ...CookieOption) {
	ctx.applyCookieOptions(cookie, OpCookieSet, options)
	http.SetCookie(ctx.writer, cookie)
}

const setCookieHeaderKey = "Set-Cookie"

// UpsertCookie adds a cookie to the response like `SetCookie` does
// but it will also perform a replacement of the cookie
// if already set by a previous `SetCookie` call.
// It reports whether the cookie is new (true) or an existing one was updated (false).
func (ctx *Context) UpsertCookie(cookie *http.Cookie, options ...CookieOption) bool {
	ctx.applyCookieOptions(cookie, OpCookieSet, options)

	header := ctx.ResponseWriter().Header()
	if cookies := header[setCookieHeaderKey]; len(cookies) > 0 {
		s := cookie.Name + "=" // name=?value

		existingUpdated := false

		for i, c := range cookies {
			if strings.HasPrefix(c, s) {
				if existingUpdated { // fixes #1877
					// remove any duplicated.
					cookies[i] = ""
					header[setCookieHeaderKey] = cookies
					continue
				}
				// We need to update the Set-Cookie (to update the expiration or any other cookie's properties).
				// Probably the cookie is set and then updated in the first session creation
				// (e.g. UpdateExpiration, see https://github.com/kataras/iris/issues/1485).
				cookies[i] = cookie.String()
				header[setCookieHeaderKey] = cookies
				existingUpdated = true
			}
		}

		if existingUpdated {
			return false // existing one updated.
		}
	}

	header.Add(setCookieHeaderKey, cookie.String())
	return true
}

// SetCookieKVExpiration is 365 days by-default
// you can change it or simple, use the SetCookie for more control.
//
// See `CookieExpires` and `AddCookieOptions` for more.
var SetCookieKVExpiration = 8760 * time.Hour

// SetCookieKV adds a cookie, requires the name(string) and the value(string).
//
// By default it expires after 365 days and it is added to the root URL path,
// use the `CookieExpires` and `CookiePath` to modify them.
// Alternatively: ctx.SetCookie(&http.Cookie{...}) or ctx.AddCookieOptions(...)
//
// If you want to set custom the path:
// ctx.SetCookieKV(name, value, iris.CookiePath("/custom/path/cookie/will/be/stored"))
//
// If you want to be visible only to current request path:
// (note that client should be responsible for that if server sent an empty cookie's path, all browsers are compatible)
// ctx.SetCookieKV(name, value, iris.CookieCleanPath/iris.CookiePath(""))
// More:
//
//	iris.CookieExpires(time.Duration)
//	iris.CookieHTTPOnly(false)
//
// Examples: https://github.com/kataras/iris/tree/main/_examples/cookies/basic
func (ctx *Context) SetCookieKV(name, value string, options ...CookieOption) {
	c := &http.Cookie{}
	c.Path = "/"
	c.Name = name
	c.Value = url.QueryEscape(value)
	c.HttpOnly = true

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	c.Expires = time.Now().Add(SetCookieKVExpiration)
	c.MaxAge = int(time.Until(c.Expires).Seconds())

	ctx.SetCookie(c, options...)
}

// GetCookie returns cookie's value by its name
// returns empty string if nothing was found.
//
// If you want more than the value then:
// cookie, err := ctx.GetRequestCookie("name")
//
// Example: https://github.com/kataras/iris/tree/main/_examples/cookies/basic
func (ctx *Context) GetCookie(name string, options ...CookieOption) string {
	c, err := ctx.GetRequestCookie(name, options...)
	if err != nil {
		return ""
	}

	value, _ := url.QueryUnescape(c.Value)
	return value
}

// GetRequestCookie returns the request cookie including any context's cookie options (stored or given by this method).
func (ctx *Context) GetRequestCookie(name string, options ...CookieOption) (*http.Cookie, error) {
	c, err := ctx.request.Cookie(name)
	if err != nil {
		return nil, err
	}

	ctx.applyCookieOptions(c, OpCookieGet, options)
	return c, nil
}

var (
	// CookieExpireDelete may be set on Cookie.Expire for expiring the given cookie.
	CookieExpireDelete = memstore.ExpireDelete

	// CookieExpireUnlimited indicates that does expires after 24 years.
	CookieExpireUnlimited = time.Now().AddDate(24, 10, 10)
)

// RemoveCookie deletes a cookie by its name and path = "/".
// Tip: change the cookie's path to the current one by: RemoveCookie("the_cookie_name", iris.CookieCleanPath)
//
// If you intend to remove a cookie with a specific domain and value, please ensure to pass these values explicitly:
//
//	ctx.RemoveCookie("the_cookie_name", iris.CookieDomain("example.com"), iris.CookiePath("/"))
//
// OR use a Cookie value instead:
//
//	ctx.RemoveCookie("the_cookie_name", iris.CookieOverride(&http.Cookie{Domain: "example.com", Path: "/"}))
//
// Example: https://github.com/kataras/iris/tree/main/_examples/cookies/basic
func (ctx *Context) RemoveCookie(name string, options ...CookieOption) {
	c := &http.Cookie{Path: "/"}
	// Send the cookie back to the client
	ctx.applyCookieOptions(c, OpCookieDel, options)
	c.Name = name
	c.Value = ""
	c.HttpOnly = true
	// Set the cookie expiration date to a past time
	c.Expires = CookieExpireDelete
	c.MaxAge = -1 // RFC says 1 second, but let's do it -1  to make sure is working.

	http.SetCookie(ctx.writer, c)
}

// VisitAllCookies takes a visitor function which is called
// on each (request's) cookies' name and value.
func (ctx *Context) VisitAllCookies(visitor func(name string, value string)) {
	for _, cookie := range ctx.request.Cookies() {
		visitor(cookie.Name, cookie.Value)
	}
}

var maxAgeExp = regexp.MustCompile(`maxage=(\d+)`)

// MaxAge returns the "cache-control" request header's value
// seconds as int64
// if header not found or parse failed then it returns -1.
func (ctx *Context) MaxAge() int64 {
	header := ctx.GetHeader(CacheControlHeaderKey)
	if header == "" {
		return -1
	}
	m := maxAgeExp.FindStringSubmatch(header)
	if len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			return int64(v)
		}
	}
	return -1
}

//  +------------------------------------------------------------+
//  | Advanced: Response Recorder                                |
//  +------------------------------------------------------------+

// Record transforms the context's basic and direct responseWriter to a *ResponseRecorder
// which can be used to reset the body, reset headers, get the body,
// get & set the status code at any time and more.
func (ctx *Context) Record() {
	switch w := ctx.writer.(type) {
	case *ResponseRecorder:
	default:
		recorder := AcquireResponseRecorder()
		recorder.BeginRecord(w)
		ctx.ResetResponseWriter(recorder)
	}
}

// Recorder returns the context's ResponseRecorder
// if not recording then it starts recording and returns the new context's ResponseRecorder
func (ctx *Context) Recorder() *ResponseRecorder {
	ctx.Record()
	return ctx.writer.(*ResponseRecorder)
}

// IsRecording returns the response recorder and a true value
// when the response writer is recording the status code, body, headers and so on,
// else returns nil and false.
func (ctx *Context) IsRecording() (*ResponseRecorder, bool) {
	// NOTE:
	// two return values in order to minimize the if statement:
	// if (Recording) then writer = Recorder()
	// instead we do: recorder,ok = Recording()
	rr, ok := ctx.writer.(*ResponseRecorder)
	return rr, ok
}

// Exec calls the framewrok's ServeHTTPC
// based on this context but with a changed method and path
// like it was requested by the user, but it is not.
//
// Offline means that the route is registered to the iris and have all features that a normal route has
// BUT it isn't available by browsing, its handlers executed only when other handler's context call them
// it can validate paths, has sessions, path parameters and all.
//
// You can find the Route by app.GetRoute("theRouteName")
// you can set a route name as: myRoute := app.Get("/mypath", handler)("theRouteName")
// that will set a name to the route and returns its RouteInfo instance for further usage.
//
// It doesn't changes the global state, if a route was "offline" it remains offline.
//
// app.None(...) and app.GetRoutes().Offline(route)/.Online(route, method)
//
// Example: https://github.com/kataras/iris/tree/main/_examples/routing/route-state
//
// User can get the response by simple using rec := ctx.Recorder(); rec.Body()/rec.StatusCode()/rec.Header().
//
// context's Values and the Session are kept in order to be able to communicate via the result route.
//
// It's for extreme use cases, 99% of the times will never be useful for you.
func (ctx *Context) Exec(method string, path string) {
	if path == "" {
		return
	}

	if method == "" {
		method = "GET"
	}

	// backup the handlers
	backupHandlers := ctx.handlers[0:]
	backupPos := ctx.currentHandlerIndex

	req := ctx.request
	// backup the request path information
	backupPath := req.URL.Path
	backupMethod := req.Method
	// don't backupValues := ctx.values.ReadOnly()
	// set the request to be align with the 'againstRequestPath'
	req.RequestURI = path
	req.URL.Path = path
	req.Method = method

	// [values stays]
	// reset handlers
	ctx.handlers = ctx.handlers[0:0]
	ctx.currentHandlerIndex = 0

	// execute the route from the (internal) context router
	// this way we keep the sessions and the values
	ctx.app.ServeHTTPC(ctx)

	// set the request back to its previous state
	req.RequestURI = backupPath
	req.URL.Path = backupPath
	req.Method = backupMethod

	// set back the old handlers and the last known index
	ctx.handlers = backupHandlers
	ctx.currentHandlerIndex = backupPos
}

// RouteExists reports whether a particular route exists
// It will search from the current subdomain of context's host, if not inside the root domain.
func (ctx *Context) RouteExists(method, path string) bool {
	return ctx.app.RouteExists(ctx, method, path)
}

const (
	reflectValueContextKey = "iris.context.reflect_value"
	// ControllerContextKey returns the context key from which
	// the `Context.Controller` method returns the store's value.
	ControllerContextKey = "iris.controller.reflect_value"
)

// ReflectValue caches and returns a []reflect.Value{reflect.ValueOf(ctx)}.
// It's just a helper to maintain variable inside the context itself.
func (ctx *Context) ReflectValue() []reflect.Value {
	if v := ctx.values.Get(reflectValueContextKey); v != nil {
		return v.([]reflect.Value)
	}

	v := []reflect.Value{reflect.ValueOf(ctx)}
	ctx.values.Set(reflectValueContextKey, v)
	return v
}

var emptyValue reflect.Value

// Controller returns a reflect Value of the custom Controller from which this handler executed.
// It will return a Kind() == reflect.Invalid if the handler was not executed from within a controller.
func (ctx *Context) Controller() reflect.Value {
	if v := ctx.values.Get(ControllerContextKey); v != nil {
		return v.(reflect.Value)
	}

	return emptyValue
}

// DependenciesContextKey is the context key for the context's value
// to keep the serve-time static dependencies raw values.
const DependenciesContextKey = "iris.dependencies"

// DependenciesMap is the type which context serve-time
// struct dependencies are stored with.
type DependenciesMap map[reflect.Type]reflect.Value

// RegisterDependency registers a struct or slice
// or pointer to struct dependency at request-time
// for the next handler in the chain. One value per type.
// Note that it's highly recommended to register
// your dependencies before server ran
// through Party.ConfigureContainer or mvc.Application.Register
// in sake of minimum performance cost.
//
// See `UnregisterDependency` too.
func (ctx *Context) RegisterDependency(v interface{}) {
	if v == nil {
		return
	}

	val, ok := v.(reflect.Value)
	if !ok {
		val = reflect.ValueOf(v)
	}

	cv := ctx.values.Get(DependenciesContextKey)
	if cv != nil {
		m, ok := cv.(DependenciesMap)
		if !ok {
			return
		}

		m[val.Type()] = val
		return
	}

	ctx.values.Set(DependenciesContextKey, DependenciesMap{
		val.Type(): val,
	})
}

// UnregisterDependency removes a dependency based on its type.
// Reports whether a dependency with that type was found and removed successfully.
func (ctx *Context) UnregisterDependency(typ reflect.Type) bool {
	cv := ctx.values.Get(DependenciesContextKey)
	if cv != nil {
		m, ok := cv.(DependenciesMap)
		if ok {
			delete(m, typ)
			return true
		}
	}

	return false
}

// Application returns the iris app instance which belongs to this context.
// Worth to notice that this function returns an interface
// of the Application, which contains methods that are safe
// to be executed at serve-time. The full app's fields
// and methods are not available here for the developer's safety.
func (ctx *Context) Application() Application {
	return ctx.app
}

// IsDebug reports whether the application runs with debug log level.
// It is a shortcut of Application.IsDebug().
func (ctx *Context) IsDebug() bool {
	return ctx.app.IsDebug()
}

// SetErr is just a helper that sets an error value
// as a context value, it does nothing more.
// Also, by-default this error's value is written to the client
// on failures when no registered error handler is available (see `Party.On(Any)ErrorCode`).
// See `GetErr` to retrieve it back.
//
// To remove an error simply pass nil.
//
// Note that, if you want to stop the chain
// with an error see the `StopWithError/StopWithPlainError` instead.
func (ctx *Context) SetErr(err error) {
	if err == nil {
		ctx.values.Remove(errorContextKey)
		return
	}

	ctx.values.Set(errorContextKey, err)
}

// GetErr is a helper which retrieves
// the error value stored by `SetErr`.
//
// Note that, if an error was stored by `SetErrPrivate`
// then it returns the underline/original error instead
// of the internal error wrapper.
func (ctx *Context) GetErr() error {
	_, err := ctx.GetErrPublic()
	return err
}

// ErrPrivate if provided then the error saved in context
// should NOT be visible to the client no matter what.
type ErrPrivate interface {
	error
	IrisPrivateError()
}

// An internal wrapper for the `SetErrPrivate` method.
type privateError struct{ error }

func (e privateError) IrisPrivateError() {}

// PrivateError accepts an error and returns a wrapped private one.
func PrivateError(err error) ErrPrivate {
	if err == nil {
		return nil
	}

	errPrivate, ok := err.(ErrPrivate)
	if !ok {
		errPrivate = privateError{err}
	}

	return errPrivate
}

const errorContextKey = "iris.context.error"

// SetErrPrivate sets an error that it's only accessible through `GetErr`
// and it should never be sent to the client.
//
// Same as ctx.SetErr with an error that completes the `ErrPrivate` interface.
// See `GetErrPublic` too.
func (ctx *Context) SetErrPrivate(err error) {
	ctx.SetErr(PrivateError(err))
}

// GetErrPublic reports whether the stored error
// can be displayed to the client without risking
// to expose security server implementation to the client.
//
// If the error is not nil, it is always the original one.
func (ctx *Context) GetErrPublic() (bool, error) {
	if v := ctx.values.Get(errorContextKey); v != nil {
		switch err := v.(type) {
		case privateError:
			// If it's an error set by SetErrPrivate then unwrap it.
			return false, err.error
		case ErrPrivate:
			return false, err
		case error:
			return true, err
		}
	}

	return false, nil
}

// ErrPanicRecovery may be returned from `Context` actions of a `Handler`
// which recovers from a manual panic.
type ErrPanicRecovery struct {
	ErrPrivate
	Cause                  interface{}
	Callers                []string // file:line callers.
	Stack                  []byte   // the full debug stack.
	RegisteredHandlers     []string // file:line of all registered handlers.
	CurrentHandlerFileLine string   // the handler panic came from.
	CurrentHandlerName     string   // the handler name panic came from.
	Request                string   // the http dumped request.
}

// Error implements the Go standard error type.
func (e *ErrPanicRecovery) Error() string {
	if e.Cause != nil {
		if err, ok := e.Cause.(error); ok {
			return err.Error()
		}
	}

	return fmt.Sprintf("%v\n%s\nRequest:\n%s", e.Cause, strings.Join(e.Callers, "\n"), e.Request)
}

// Is completes the internal errors.Is interface.
func (e *ErrPanicRecovery) Is(err error) bool {
	_, ok := IsErrPanicRecovery(err)
	return ok
}

func (e *ErrPanicRecovery) LogMessage() string {
	logMessage := fmt.Sprintf("Recovered from a route's Handler('%s')\n", e.CurrentHandlerName)
	logMessage += fmt.Sprint(e.Request)
	logMessage += fmt.Sprintf("%s\n", e.Cause)
	logMessage += fmt.Sprintf("%s\n", strings.Join(e.Callers, "\n"))

	return logMessage
}

// IsErrPanicRecovery reports whether the given "err" is a type of ErrPanicRecovery.
func IsErrPanicRecovery(err error) (*ErrPanicRecovery, bool) {
	if err == nil {
		return nil, false
	}
	v, ok := err.(*ErrPanicRecovery)
	return v, ok
}

// IsRecovered reports whether this handler has been recovered
// by the Iris recover middleware.
func (ctx *Context) IsRecovered() (*ErrPanicRecovery, bool) {
	if ctx.GetStatusCode() == http.StatusInternalServerError {
		// Panic error from recovery middleware is private.
		return IsErrPanicRecovery(ctx.GetErr())
	}

	return nil, false
}

const (
	funcsContextPrefixKey = "iris.funcs."
	funcLogoutContextKey  = "auth.logout_func"
)

// SetFunc registers a custom function to this Request.
// It's a helper to pass dynamic functions across handlers of the same chain.
// For a more complete solution please use Dependency Injection instead.
// This is just an easy to way to pass a function to the
// next handler like ctx.Values().Set/Get does.
// Sometimes is faster and easier to pass the object as a request value
// and cast it when you want to use one of its methods instead of using
// these `SetFunc` and `CallFunc` methods.
// This implementation is suitable for functions that may change inside the
// handler chain and the object holding the method is not predictable.
//
// The "name" argument is the custom name of the function,
// you will use its name to call it later on, e.g. "auth.myfunc".
//
// The second, "fn" argument is the raw function/method you want
// to pass through the next handler(s) of the chain.
//
// The last variadic input argument is optionally, if set
// then its arguments are passing into the function's input arguments,
// they should be always be the first ones to be accepted by the "fn" inputs,
// e.g. an object, a receiver or a static service.
//
// See its `CallFunc` to call the "fn" on the next handler.
//
// Example at:
// https://github.com/kataras/iris/tree/main/_examples/routing/writing-a-middleware/share-funcs
func (ctx *Context) SetFunc(name string, fn interface{}, persistenceArgs ...interface{}) {
	f := newFunc(name, fn, persistenceArgs...)
	ctx.values.Set(funcsContextPrefixKey+name, f)
}

// GetFunc returns the context function declaration which holds
// some information about the function registered under the given "name" by
// the `SetFunc` method.
func (ctx *Context) GetFunc(name string) (*Func, bool) {
	fn := ctx.values.Get(funcsContextPrefixKey + name)
	if fn == nil {
		return nil, false
	}

	return fn.(*Func), true
}

// CallFunc calls the function registered by the `SetFunc`.
// The input arguments MUST match the expected ones.
//
// If the registered function was just a handler
// or a handler which returns an error
// or a simple function
// or a simple function which returns an error
// then this operation will perform without any serious cost,
// otherwise reflection will be used instead, which may slow down the overall performance.
//
// Retruns ErrNotFound if the function was not registered.
//
// For a more complete solution without limiations navigate through
// the Iris Dependency Injection feature instead.
func (ctx *Context) CallFunc(name string, args ...interface{}) ([]reflect.Value, error) {
	fn, ok := ctx.GetFunc(name)
	if !ok || fn == nil {
		return nil, ErrNotFound
	}

	return fn.call(ctx, args...)
}

// SetLogoutFunc registers a custom logout function that will be
// available to use inside the next handler(s). The function
// may be registered multiple times but the last one is the valid.
// So a logout function may start with basic authentication
// and other middleware in the chain may change it to a custom sessions logout one.
// This method uses the `SetFunc` method under the hoods.
//
// See `Logout` method too.
func (ctx *Context) SetLogoutFunc(fn interface{}, persistenceArgs ...interface{}) {
	ctx.SetFunc(funcLogoutContextKey, fn, persistenceArgs...)
}

// Logout calls the registered logout function.
// Returns ErrNotFound if a logout function was not specified
// by a prior call of `SetLogoutFunc`.
func (ctx *Context) Logout(args ...interface{}) error {
	_, err := ctx.CallFunc(funcLogoutContextKey, args...)
	return err
}

const userContextKey = "iris.user"

// SetUser sets a value as a User for this request.
// It's used by auth middlewares as a common
// method to provide user information to the
// next handlers in the chain.
//
// The "i" input argument can be:
//   - A value which completes the User interface
//   - A map[string]interface{}.
//   - A value which does not complete the whole User interface
//   - A value which does not complete the User interface at all
//     (only its `User().GetRaw` method is available).
//
// Look the `User` method to retrieve it.
func (ctx *Context) SetUser(i interface{}) error {
	if i == nil {
		ctx.values.Remove(userContextKey)
		return nil
	}

	u, ok := i.(User)
	if !ok {
		if m, ok := i.(Map); ok { // it's a map, convert it to a User.
			u = UserMap(m)
		} else {
			// It's a structure, wrap it and let
			// runtime decide the features.
			p := newUserPartial(i)
			if p == nil {
				return ErrNotSupported
			}
			u = p
		}
	}

	ctx.values.Set(userContextKey, u)
	return nil
}

// User returns the registered User of this request.
// To get the original value (even if a value set by SetUser does not implement the User interface)
// use its GetRaw method.
// /
// See `SetUser` too.
func (ctx *Context) User() User {
	if v := ctx.values.Get(userContextKey); v != nil {
		if u, ok := v.(User); ok {
			return u
		}
	}

	return nil
}

// Ensure Iris Context implements the standard Context package, build-time.
var _ context.Context = (*Context)(nil)

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
//
// Shortcut of Request().Context().Deadline().
func (ctx *Context) Deadline() (deadline time.Time, ok bool) {
	return ctx.request.Context().Deadline()
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
// The close of the Done channel may happen asynchronously,
// after the cancel function returns.
//
// WithCancel arranges for Done to be closed when cancel is called;
// WithDeadline arranges for Done to be closed when the deadline
// expires; WithTimeout arranges for Done to be closed when the timeout
// elapses.
//
// Done is provided for use in select statements:
//
//	// Stream generates values with DoSomething and sends them to out
//	// until DoSomething returns an error or ctx.Done is closed.
//	func Stream(ctx context.Context, out chan<- Value) error {
//		for {
//			v, err := DoSomething(ctx)
//			if err != nil {
//				return err
//			}
//			select {
//			case <-ctx.Done():
//				return ctx.Err()
//			case out <- v:
//			}
//		}
//	}
//
// See https://blog.golang.org/pipelines for more examples of how to use
// a Done channel for cancellation.
//
// Shortcut of Request().Context().Done().
func (ctx *Context) Done() <-chan struct{} {
	return ctx.request.Context().Done()
}

// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
//
// Shortcut of Request().Context().Err().
func (ctx *Context) Err() error {
	return ctx.request.Context().Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
//
// Shortcut of Request().Context().Value(key interface{}) interface{}.
func (ctx *Context) Value(key interface{}) interface{} {
	if keyStr, ok := key.(string); ok { // check if the key is a type of string, which can be retrieved by the mem store.
		if entry, exists := ctx.values.GetEntry(keyStr); exists {
			return entry.ValueRaw
		}
	}
	// otherwise return the chained value.
	return ctx.request.Context().Value(key)
}

const idContextKey = "iris.context.id"

// SetID sets an ID, any value, to the Request Context.
// If possible the "id" should implement a `String() string` method
// so it can be rendered on `Context.String` method.
//
// See `GetID` and `middleware/requestid` too.
func (ctx *Context) SetID(id interface{}) {
	ctx.values.Set(idContextKey, id)
}

// GetID returns the Request Context's ID.
// It returns nil if not given by a prior `SetID` call.
// See `middleware/requestid` too.
func (ctx *Context) GetID() interface{} {
	return ctx.values.Get(idContextKey)
}

// String returns the string representation of this request.
//
// It returns the Context's ID given by a `SetID`call,
// followed by the client's IP and the method:uri.
func (ctx *Context) String() string {
	id := ctx.GetID()
	if id != nil {
		if stringer, ok := id.(fmt.Stringer); ok {
			id = stringer.String()
		}
	}

	return fmt.Sprintf("[%v] %s ▶ %s:%s", id, ctx.RemoteAddr(), ctx.Method(), ctx.Request().RequestURI)
}
