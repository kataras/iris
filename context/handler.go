package context

import (
	"reflect"
	"runtime"
)

// A Handler responds to an HTTP request.
// It writes reply headers and data to the Context.ResponseWriter() and then return.
// Returning signals that the request is finished;
// it is not valid to use the Context after or concurrently with the completion of the Handler call.
//
// Depending on the HTTP client software, HTTP protocol version,
// and any intermediaries between the client and the iris server,
// it may not be possible to read from the Context.Request().Body after writing to the context.ResponseWriter().
// Cautious handlers should read the Context.Request().Body first, and then reply.
//
// Except for reading the body, handlers should not modify the provided Context.
//
// If Handler panics, the server (the caller of Handler) assumes that the effect of the panic was isolated to the active request.
// It recovers the panic, logs a stack trace to the server error log, and hangs up the connection.
type Handler func(Context)

// Handlers is just a type of slice of []Handler.
//
// See `Handler` for more.
type Handlers []Handler

// HandlerName returns the name, the handler function informations.
// Same as `context.HandlerName`.
func HandlerName(h Handler) string {
	pc := reflect.ValueOf(h).Pointer()
	// l, n := runtime.FuncForPC(pc).FileLine(pc)
	// return fmt.Sprintf("%s:%d", l, n)
	return runtime.FuncForPC(pc).Name()
}

// Filter is just a type of func(Handler) bool which reports whether an action must be performed
// based on the incoming request.
//
// See `NewConditionalHandler` for more.
type Filter func(Context) bool

// NewConditionalHandler returns a single Handler which can be registered
// as a middleware.
// Filter is just a type of Handler which returns a boolean.
// Handlers here should act like middleware, they should contain `ctx.Next` to proceed
// to the next handler of the chain. Those "handlers" are registered to the per-request context.
//
//
// It checks the "filter" and if passed then
// it, correctly, executes the "handlers".
//
// If passed, this function makes sure that the Context's information
// about its per-request handler chain based on the new "handlers" is always updated.
//
// If not passed, then simply the Next handler(if any) is executed and "handlers" are ignored.
//
// Example can be found at: _examples/routing/conditional-chain.
func NewConditionalHandler(filter Filter, handlers ...Handler) Handler {
	return func(ctx Context) {
		if filter(ctx) {
			// Note that we don't want just to fire the incoming handlers, we must make sure
			// that it won't break any further handler chain
			// information that may be required for the next handlers.
			//
			// The below code makes sure that this conditional handler does not break
			// the ability that iris provides to its end-devs
			// to check and modify the per-request handlers chain at runtime.
			currIdx := ctx.HandlerIndex(-1)
			currHandlers := ctx.Handlers()
			if currIdx == len(currHandlers)-1 {
				// if this is the last handler of the chain
				// just add to the last the new handlers and call Next to fire those.
				ctx.AddHandler(handlers...)
				ctx.Next()
				return
			}
			// otherwise insert the new handlers in the middle of the current executed chain and the next chain.
			newHandlers := append(currHandlers[:currIdx], append(handlers, currHandlers[currIdx+1:]...)...)
			ctx.SetHandlers(newHandlers)
			ctx.Next()
			return
		}
		// if not pass, then just execute the next.
		ctx.Next()
	}
}
