package iris

import (
	"net/http"

	"github.com/kataras/go-errors"
)

// errHandler returns na error with message: 'Passed argument is not func(*Context) neither an object which implements the iris.Default.Handler with Serve(ctx *Context)
// It seems to be a  +type Points to: +pointer.'
var errHandler = errors.New(`
Passed argument is not an iris.Handler (or func(*iris.Context)) neither one of these types:
  - http.Handler
  - func(w http.ResponseWriter, r *http.Request)
  - func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)
  ---------------------------------------------------------------------
It seems to be a  %T points to: %v`)

type (
	// Handler the main Iris Handler interface.
	Handler interface {
		Serve(ctx *Context) // iris-specific
	}

	// HandlerFunc type is an adapter to allow the use of
	// ordinary functions as HTTP handlers.  If f is a function
	// with the appropriate signature, HandlerFunc(f) is a
	// Handler that calls f.
	HandlerFunc func(*Context)
	// Middleware is just a slice of Handler []func(c *Context)
	Middleware []Handler
)

// Serve implements the Handler, is like ServeHTTP but for Iris
func (h HandlerFunc) Serve(ctx *Context) {
	h(ctx)
}

// ToNativeHandler converts an iris handler to http.Handler
func ToNativeHandler(s *Framework, h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Context.Run(w, r, func(ctx *Context) {
			h.Serve(ctx)
		})
	})
}

// ToHandler converts different style of handlers that you
// used to use (usually with third-party net/http middleware) to an iris.HandlerFunc.
//
// Supported types:
// - .ToHandler(h http.Handler)
// - .ToHandler(func(w http.ResponseWriter, r *http.Request))
// - .ToHandler(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc))
func ToHandler(handler interface{}) HandlerFunc {
	switch handler.(type) {
	case HandlerFunc:
		{
			//
			//it's already an iris handler
			//
			return handler.(HandlerFunc)
		}

	case http.Handler:
		//
		// handlerFunc.ServeHTTP(w,r)
		//
		{
			h := handler.(http.Handler)
			return func(ctx *Context) {
				h.ServeHTTP(ctx.ResponseWriter, ctx.Request)
			}
		}

	case func(http.ResponseWriter, *http.Request):
		{
			//
			// handlerFunc(w,r)
			//
			return ToHandler(http.HandlerFunc(handler.(func(http.ResponseWriter, *http.Request))))
		}

	case func(http.ResponseWriter, *http.Request, http.HandlerFunc):
		{
			//
			// handlerFunc(w,r, http.HandlerFunc)
			//
			return toHandlerNextHTTPHandlerFunc(handler.(func(http.ResponseWriter, *http.Request, http.HandlerFunc)))
		}

	default:
		{
			//
			// No valid handler passed
			//
			panic(errHandler.Format(handler, handler))
		}

	}

}

func toHandlerNextHTTPHandlerFunc(h func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)) HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		// take the next handler in route's chain
		nextIrisHandler := ctx.NextHandler()
		if nextIrisHandler != nil {
			executed := false // we need to watch this in order to StopExecution from all next handlers
			// if this next handler is not executed by the third-party net/http next-style middleware.
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextIrisHandler.Serve(ctx)
				executed = true
			})

			h(ctx.ResponseWriter, ctx.Request, nextHandler)

			// after third-party middleware's job:
			if executed {
				// if next is executed then increment the ctx.Pos manually
				// in order to the next handler not to be executed twice.
				ctx.Pos++
			} else {
				// otherwise StopExecution from all next handlers.
				ctx.StopExecution()
			}
			return
		}

		// if not next handler found then this is not a 'valid' middleware but
		// some middleware may don't care about next,
		// so we just execute the handler with an empty net.
		h(ctx.ResponseWriter, ctx.Request, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	})
}

// convertToHandlers just make []HandlerFunc to []Handler, although HandlerFunc and Handler are the same
// we need this on some cases we explicit want a interface Handler, it is useless for users.
func convertToHandlers(handlersFn []HandlerFunc) []Handler {
	hlen := len(handlersFn)
	mlist := make([]Handler, hlen)
	for i := 0; i < hlen; i++ {
		mlist[i] = Handler(handlersFn[i])
	}
	return mlist
}

// joinMiddleware uses to create a copy of all middleware and return them in order to use inside the node
func joinMiddleware(middleware1 Middleware, middleware2 Middleware) Middleware {
	nowLen := len(middleware1)
	totalLen := nowLen + len(middleware2)
	// create a new slice of middleware in order to store all handlers, the already handlers(middleware) and the new
	newMiddleware := make(Middleware, totalLen)
	//copy the already middleware to the just created
	copy(newMiddleware, middleware1)
	//start from there we finish, and store the new middleware too
	copy(newMiddleware[nowLen:], middleware2)
	return newMiddleware
}
