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
