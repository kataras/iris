package context

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/kataras/iris/v14/context/encoding"

	"github.com/iris-contrib/schema"
)

// Map is just a type alias of the map[string]any type.
type Map = map[string]any

// Context is the middle-man server's "object" dealing with incoming requests.
//
// A New context is being acquired from a sync.Pool on each connection.
// The Context is the most important thing on the iris's http flow.
//
// Developers send responses to the client's request through a Context.
// Developers get request information from the client's request a Context.
//
// Context implements the standard library's context.Context, so a handler
// can pass it directly to code that takes one. Do that only for calls that
// finish before the handler returns: the Context is pooled and reused
// across requests, so a consumer that keeps it longer (an HTTP client's
// transport, a spawned goroutine, a database driver) ends up reading a
// released Context's state. For those, pass StdContext, or WithoutCancel
// for work that must survive the request's cancellation.
//
// Corresponding source code files:
//   - context.go
//   - context_request.go
//   - context_response.go
type Context struct {
	// the http.ResponseWriter wrapped by custom writer.
	writer ResponseWriter
	// the original http.Request
	request *http.Request
	// the current route registered to this request path.
	currentRoute RouteReadOnly
	options      *ContextOptions

	// the local key-value storage
	params encoding.PathParams    // url named parameters.
	values encoding.ContextValues // generic storage, middleware communication.
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

	// releaseHooks run, in reverse order, right before the response writer and
	// this Context are recycled. See OnRelease.
	releaseHooks []func()
}

// NewContext returns a new Context instance.
func NewContext(app Application) *Context {
	return &Context{app: app}
}

/* Not required, unless requested.
// SetApplication sets an Iris Application on-fly.
// Do NOT use it after HandleRequest is fired.
func (ctx *Context) SetApplication(app Application) {
	ctx.app = app
}
*/

// Clone returns a copy of the context that
// can be safely used outside the request's scope.
// Note that if the request-response lifecycle terminated
// or request canceled by the client (can be checked by `ctx.IsCanceled()`)
// then the response writer is totally useless.
// The request is copied through http.Request.Clone, keeping the same standard
// context, therefore modifying the clone's URL or Header does not affect the
// original request; the request's Body reader is still shared between the two.
func (ctx *Context) Clone() *Context {
	req := ctx.request.Clone(ctx.request.Context())
	cloned := &Context{
		app:                 ctx.app,
		values:              ctx.values.Clone(),
		params:              ctx.params.Clone(),
		writer:              ctx.writer.Clone(),
		options:             ctx.options.Clone(),
		request:             req,
		currentHandlerIndex: stopExecutionIndex,
		proceeded:           ctx.proceeded,
		currentRoute:        ctx.currentRoute,
	}
	// copyOptions(ctx, cloned)
	return cloned
}

// OnRelease registers fn to run once the handler chain has returned, right
// before the response writer and this Context go back to their pools. It runs
// on the request's own goroutine, so it may touch the Context, and it is the
// last point at which the response writer is still this request's.
//
// It exists for goroutines that write to the response after the handler has
// moved on - a heartbeat, a ticker that drains a batch - and would otherwise
// find the writer gone or, worse, serving the next request: the Context is
// recycled inside ServeHTTP, before net/http cancels the request context, so
// waiting on Request().Context().Done() is not enough. Stop such a goroutine
// here and wait for it.
//
// Hooks run in reverse registration order, like defer. They are dropped once
// run, so nothing carries over to the next request served by the same pooled
// Context. Keep them short: the connection is held until they return. A hook
// may register another, which runs in the same call; a hook must not register
// itself again, because RunReleaseHooks makes a bounded number of passes and
// drops what is left.
//
// It is unrelated to OnConnectionClose, which fires on a separate goroutine
// when the client goes away, possibly long before the handler chain returns.
func (ctx *Context) OnRelease(fn func()) {
	if fn == nil {
		return
	}

	ctx.releaseHooks = append(ctx.releaseHooks, fn)
}

// maxReleaseHookPasses bounds RunReleaseHooks. Each pass runs the hooks
// registered so far, most recent first, and hooks registered meanwhile get the
// next pass; a hook that re-arms itself would otherwise spin the request
// goroutine forever.
const maxReleaseHookPasses = 8

// RunReleaseHooks runs and drops the OnRelease callbacks, most recent first.
// A hook registered by a running hook runs in the same call, so a stream that
// closes a coalescer at release can register the coalescer's own cleanup. The
// number of passes is bounded (see maxReleaseHookPasses); a hook that keeps
// registering itself is dropped after that, with a debug log line.
//
// The router calls it after the handler chain has returned and before it
// releases the response writer and the Context; a host that acquires Contexts
// from the Pool itself must do the same, or a hook registered by one request
// runs for the next one. It is a no-op on a Context that Pool.ForceRelease has
// already recycled, because ForceRelease drops the hooks.
func (ctx *Context) RunReleaseHooks() {
	for pass := 0; len(ctx.releaseHooks) > 0; pass++ {
		if pass == maxReleaseHookPasses {
			if ctx.app != nil {
				ctx.app.Logger().Debugf("Context.RunReleaseHooks: dropping %d hook(s) still registered after %d passes: a release hook must not re-arm itself",
					len(ctx.releaseHooks), pass)
			}

			clear(ctx.releaseHooks)
			ctx.releaseHooks = ctx.releaseHooks[:0]
			return
		}

		hooks := ctx.releaseHooks
		ctx.releaseHooks = nil
		for i := len(hooks) - 1; i >= 0; i-- {
			hooks[i]()
		}

		clear(hooks)
		if ctx.releaseHooks == nil {
			ctx.releaseHooks = hooks[:0] // keep the backing array for the next request.
		}
	}
}

// SetCurrentRoute sets the route internally,
// See `CurrentRoute()` method too.
// It's being initialized by the Router, do not call it.
// See the `SetHandlers` and `AddHandler` methods to simulate a request.
func (ctx *Context) SetCurrentRoute(route RouteReadOnly) {
	ctx.currentRoute = route
}

// CurrentRoute returns the current "read-only" route that
// was registered to this request's path.
func (ctx *Context) CurrentRoute() RouteReadOnly {
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

// AddHandler appends handler(s) to this request's chain at serve-time.
// The handlers run after the ones already registered on the route and are
// not persisted back to the router: the next request starts from the
// route's own chain again.
//
// See SetHandlers to replace the chain instead, and NewConditionalHandler
// to insert handlers in the middle of it.
func (ctx *Context) AddHandler(handlers ...Handler) {
	if len(handlers) == 0 {
		return
	}

	// The full-slice expression is required: ctx.handlers may alias the route's
	// registered Handlers (see Do), so an append with spare capacity would write
	// into that shared array and leak these serve-time handlers into every later
	// request on the same route.
	n := len(ctx.handlers)
	ctx.handlers = append(ctx.handlers[:n:n], handlers...)
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

// Proceed executes the given "handlers" without handler index increment.
func (ctx *Context) Proceed(handlers ...Handler) bool {
	for _, h := range handlers {
		_, ok := ctx.proceedAndReportIfStopped(h)
		if !ok {
			return false
		}
	}

	return true
}

// proceedAndReportIfStopped runs one handler for Proceed without advancing
// the handler index. The first result reports whether "h" called
// StopExecution, the second whether the chain may continue after it.
func (ctx *Context) proceedAndReportIfStopped(h Handler) (bool, bool) {
	ctx.proceeded = internalPauseExecutionIndex // in order to stop advance the currentHandlerIndex.

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

// currentHandler returns the handler being executed, or nil when there is none:
// the chain is empty, or StopExecution moved the index out of range.
//
// The nil case is reached in practice from the recovery middleware, which asks
// for the current handler's name while unwinding a panic. Panicking here would
// escape the deferred recover that is already running.
func (ctx *Context) currentHandler() Handler {
	if ctx.currentHandlerIndex < 0 || ctx.currentHandlerIndex >= len(ctx.handlers) {
		return nil
	}

	return ctx.handlers[ctx.currentHandlerIndex]
}

// HandlerName returns the current handler's name, helpful for debugging.
// It returns an empty string if the execution has stopped or no handler is running.
func (ctx *Context) HandlerName() string {
	h := ctx.currentHandler()
	if h == nil {
		return ""
	}

	return HandlerName(h)
}

// HandlerFileLine returns the current running handler's function source file and line information.
// Useful mostly when debugging.
// It returns an empty file and a zero line if the execution has stopped or no handler is running.
func (ctx *Context) HandlerFileLine() (file string, line int) {
	h := ctx.currentHandler()
	if h == nil {
		return "", 0
	}

	return HandlerFileLine(h)
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

// ReadValues binds this context's values to "ptr". The struct field tag is "context".
func (ctx *Context) ReadValues(ptr any) error {
	n := ctx.values.Len()
	if n == 0 {
		return nil
	}

	values := ctx.values.ToMap()

	err := schema.DecodeContextValues(values, ptr)
	if err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(ptr)
}

// SetErr is just a helper that sets an error value
// as a context value, it does nothing more.
// Also, by-default this error's value is written to the client
// on failures when no registered error handler is available (see `Router.On(Any)ErrorCode`).
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
	_, err := ctx.ErrPublic()
	return err
}

// ErrPrivate if provided then the error saved in context
// should NOT be visible to the client no matter what.
type ErrPrivate interface {
	error
	IrisPrivateError()
}

// An internal wrapper for the `SetErrPrivate` method and `PrivateError` package-level function.
type privateError struct{ error }

// IrisPrivateError reports that this error should not be visible to the client.
// It's a marker interface. It does nothing.
func (e privateError) IrisPrivateError() {}

// Error implements the Go standard error type.
func (err privateError) Error() string {
	if err.error == nil {
		return ""
	}

	return err.error.Error()
}

// PrivateError accepts an error and returns
// an `ErrPrivate` by wrapping the given "err" with the internal privateError struct.
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

// PrivateErrorf accepts an error and returns a wrapped one
// which includes the "err" and an empty private error, by using fmt.Errorf,
// so the actual error message does contain the "err" message
// but this final, wrapped, error can be marked as an `ErrPrivate` one.
//
// See rest/errors#UnwrapAll and rest/errors#err for more.
func PrivateErrorf(err error) error {
	if err == nil {
		return nil
	}

	if errPrivate, ok := err.(ErrPrivate); ok {
		return errPrivate
	}

	return fmt.Errorf("%w%w", err, privateError{})
}

const errorContextKey = "iris.context.error"

// SetErrPrivate sets an error that it's only accessible through `GetErr`
// and it should never be sent to the client.
//
// Same as ctx.SetErr with an error that completes the `ErrPrivate` interface.
// See `ErrPublic` too.
func (ctx *Context) SetErrPrivate(err error) {
	ctx.SetErr(PrivateError(err))
}

// ErrPublic reports whether the stored error
// can be displayed to the client without risking
// to expose security server implementation to the client.
//
// If the error is not nil, it is always the original one.
func (ctx *Context) ErrPublic() (bool, error) {
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
	Cause                  any
	Callers                []string // file:line callers.
	Stack                  []byte   // the full debug stack.
	RegisteredHandlers     []string // file:line of all registered handlers.
	CurrentHandlerFileLine string   // the handler panic came from.
	CurrentHandlerName     string   // the handler name panic came from.
	Request                string   // the http dumped request.
}

// Error implements the Go standard error type.
//
// It reports the cause and the callers but not the dumped request. This error is
// private (see ErrPrivate) so the default renderer never sends it to a client,
// but a custom error handler that formats it would otherwise put the request's
// headers in the response. Use LogMessage for the full form.
func (e *ErrPanicRecovery) Error() string {
	if e.Cause != nil {
		if err, ok := e.Cause.(error); ok {
			return err.Error()
		}
	}

	return fmt.Sprintf("%v\n%s", e.Cause, strings.Join(e.Callers, "\n"))
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
	funcLogoutContextKey = "auth.logout_func"
)

// LogoutHandler is a function which should be responsible
// to clear the authentication information of a request.
type LogoutHandler = func(*Context) error

// SetLogoutFunc registers a custom logout handler that will be
// available to use inside the next handler(s). The function
// may be registered multiple times but the last one is the valid.
// So a logout function may start with basic authentication
// and other middleware in the chain may change it to a custom sessions logout one.
//
// See `Logout` method too.
func (ctx *Context) SetLogoutFunc(fn LogoutHandler) {
	if fn == nil {
		ctx.Values().Remove(funcLogoutContextKey)
		return
	}

	ctx.Values().Set(funcLogoutContextKey, fn)
}

// Logout calls the registered logout function, if any.
func (ctx *Context) Logout() error {
	// Type-assert the stored value: reinterpreting the interface's memory
	// through unsafe.Pointer reads its type descriptor instead of the func
	// value and jumps to an invalid code address.
	if logout, ok := ctx.Values().Get(funcLogoutContextKey).(LogoutHandler); ok && logout != nil {
		return logout(ctx)
	}

	return nil
}

// UserContextKey is the context key for the context's value to keep the authorized user information.
const UserContextKey = "iris.user"

// SetUser sets a value as a User for this request.
// It's used by auth middlewares as a common
// method to provide user information to the
// next handlers in the chain.
//
// The "i" input argument can be:
//   - A value which completes the User interface
//   - A map[string]any.
//   - A value which does not complete the whole User interface
//   - A value which does not complete the User interface at all
//     (only its `User().GetRaw` method is available).
//
// Look the `User` method to retrieve it.
func (ctx *Context) SetUser(i any) error {
	if i == nil {
		ctx.values.Remove(UserContextKey)
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

	ctx.values.Set(UserContextKey, u)
	return nil
}

// User returns the registered User of this request.
// To get the original value (even if a value set by SetUser does not implement the User interface)
// use its GetRaw method.
//
// SECURITY: do not write the returned value to the response. A User carries the
// credentials it authenticated with: SimpleUser has a Password and a Token
// field, and a value wrapped by SetUser is serialized whole. Either way
// ctx.JSON(ctx.User()) returns those to the caller. Pick the fields you mean to
// expose:
//
//	u := ctx.User()
//	id, _ := u.GetID()
//	username, _ := u.GetUsername()
//	ctx.JSON(iris.Map{"id": id, "username": username})
//
// See `SetUser` too.
func (ctx *Context) User() User {
	if v := ctx.values.Get(UserContextKey); v != nil {
		if u, ok := v.(User); ok {
			return u
		}
	}

	return nil
}

// Ensure Iris Context implements the standard Context package, build-time.
var _ context.Context = (*Context)(nil)

// releasedDone is the Done channel of a Context that has been released back
// to its pool: already closed, so a holder sees a canceled context. See Done.
var releasedDone = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
//
// Shortcut of Request().Context().Deadline().
// A released Context reports no deadline, see Done.
func (ctx *Context) Deadline() (deadline time.Time, ok bool) {
	if r := ctx.request; r != nil {
		return r.Context().Deadline()
	}

	return time.Time{}, false
}

// StdContext returns the request's standard library context: the one
// net/http created for this request, carrying its deadline and its
// cancellation on client disconnect or request completion.
//
// Pass this, not the Iris Context itself, to any API that takes a
// context.Context and may hold on to it past the handler's return: HTTP
// clients (the transport keeps the context for cancel propagation after
// the response is delivered), database drivers, spawned goroutines. The
// Iris Context also implements context.Context, but it is pooled and
// reused across requests; a consumer that touches it after the handler
// returned races with the pool's release of it. The returned standard
// context is safe to keep.
//
// For work that must survive the request's cancellation, use
// WithoutCancel instead.
//
// Shortcut of Request().Context().
func (ctx *Context) StdContext() context.Context {
	return ctx.request.Context()
}

// WithoutCancel returns a standard context derived from the request's context
// that is never canceled when the client disconnects or the request finishes,
// while its values are preserved (standard context.WithoutCancel semantics).
// Use it for must-finish work spawned from a handler:
//
//	go persist(ctx.WithoutCancel(), record)
//
// Do not retain the Iris Context itself past the handler: it is pooled and
// reused across requests; the returned standard context is safe to keep.
func (ctx *Context) WithoutCancel() context.Context {
	return context.WithoutCancel(ctx.request.Context())
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
//
// Once the router has released this Context to its pool (the handler chain
// returned), it behaves as a canceled standard context: Done is closed, Err
// returns context.Canceled and Value only consults the request-scoped Values
// store, which is empty by then. This keeps code that holds on to the Context
// past the handler's return from dereferencing the request it no longer owns.
// pgx is one such holder: when the acquirer gives up mid-dial, jackc/puddle
// finishes dialing with the acquirer's context for Value lookups, and the Go
// resolver's context.WithDeadline then calls Value on it. The pooled Context
// may already serve another request by then, so this is a safety net, not a
// contract: pass StdContext() or WithoutCancel() to anything that may outlive
// the handler.
func (ctx *Context) Done() <-chan struct{} {
	if r := ctx.request; r != nil {
		return r.Context().Done()
	}

	return releasedDone
}

// Err reports why the request's context was canceled, if it was.
//
// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
//
// Shortcut of Request().Context().Err().
// A released Context returns context.Canceled, see Done.
func (ctx *Context) Err() error {
	if r := ctx.request; r != nil {
		return r.Context().Err()
	}

	return context.Canceled
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
//
// For a string key it first looks the key up in the Context.Values() store
// and returns that entry when it exists, otherwise it falls back to
// Request().Context().Value(key).
func (ctx *Context) Value(key any) any {
	if keyStr, ok := key.(string); ok { // check if the key is a type of string, which can be retrieved by the mem store.
		if entry, exists := ctx.values.GetEntry(keyStr); exists {
			return entry.Value
		}
	}
	// otherwise return the chained value.
	if r := ctx.request; r != nil {
		return r.Context().Value(key)
	}

	return nil // released Context, see Done.
}

const idContextKey = "iris.context.id"

// SetID sets an ID, any value, to the Request Context.
// If possible the "id" should implement a `String() string` method
// so it can be rendered on `Context.String` method.
//
// See `ID` and `middleware/requestid` too.
func (ctx *Context) SetID(id any) {
	ctx.values.Set(idContextKey, id)
}

// ID returns the Request Context's ID.
// It returns nil if not given by a prior `SetID` call.
// See `middleware/requestid` too.
func (ctx *Context) ID() any {
	return ctx.values.Get(idContextKey)
}

// String returns the string representation of this request.
//
// It returns the Context's ID given by a `SetID`call,
// followed by the client's IP and the method:uri.
//
// The address and the URI are client-controlled, so newlines and other control
// characters are stripped: this value is written to logs, where an unescaped
// newline would let a caller forge log entries.
func (ctx *Context) String() string {
	id := ctx.ID()
	if id != nil {
		if stringer, ok := id.(fmt.Stringer); ok {
			id = stringer.String()
		}
	}

	return fmt.Sprintf("[%v] %s ▶ %s:%s", id,
		stripControlChars(ctx.RemoteAddr()),
		ctx.Method(),
		stripControlChars(ctx.Request().RequestURI))
}

// stripControlChars replaces control characters, most importantly CR and LF,
// with '?'. Use it on client-controlled text that is written to a log line.
func stripControlChars(s string) string {
	if strings.IndexFunc(s, unicode.IsControl) < 0 {
		return s
	}

	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}

		return r
	}, s)
}
