package handlerconv

import (
	"net/http"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
)

var errHandler = errors.New(`
	Passed argument is not a func(context.Context) neither one of these types:
	- http.Handler
	- func(w http.ResponseWriter, r *http.Request)
	- func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)
	---------------------------------------------------------------------
	It seems to be a  %T points to: %v`)

// FromStd converts native http.Handler & http.HandlerFunc to context.Handler.
//
// Supported form types:
// 		 .FromStd(h http.Handler)
// 		 .FromStd(func(w http.ResponseWriter, r *http.Request))
// 		 .FromStd(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc))
func FromStd(handler interface{}) context.Handler {
	switch handler.(type) {
	case context.Handler:
		{
			//
			//it's already a iris handler
			//
			return handler.(context.Handler)
		}

	case http.Handler:
		//
		// handlerFunc.ServeHTTP(w,r)
		//
		{
			h := handler.(http.Handler)
			return func(ctx context.Context) {
				h.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
			}
		}

	case func(http.ResponseWriter, *http.Request):
		{
			//
			// handlerFunc(w,r)
			//
			return FromStd(http.HandlerFunc(handler.(func(http.ResponseWriter, *http.Request))))
		}

	case func(http.ResponseWriter, *http.Request, http.HandlerFunc):
		{
			//
			// handlerFunc(w,r, http.HandlerFunc)
			//
			return FromStdWithNext(handler.(func(http.ResponseWriter, *http.Request, http.HandlerFunc)))
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

// FromStdWithNext receives a standar handler - middleware form - and returns a
// compatible context.Handler wrapper.
func FromStdWithNext(h func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)) context.Handler {
	return func(ctx context.Context) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx.Next()
		})

		h(ctx.ResponseWriter(), ctx.Request(), next)
	}
}
