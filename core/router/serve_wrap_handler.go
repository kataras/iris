package router

import "net/http"

// WrapperFunc is used as an expected input parameter signature
// for the WrapRouter. It's a "low-level" signature which is compatible
// with the net/http.
// It's being used to run or no run the router based on a custom logic.
type WrapperFunc func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc)

// wrapHandler returns a new http.Handler that wraps the "mainHandler" with the "wrapperFunc".
//
// Application is responsible to make it to work on more than one wrappers
// via composition or func clojure, see "joinWrapperFunc".
func wrapHandler(mainHandlerFunc http.Handler, wrapperFunc WrapperFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wrapperFunc(w, r, mainHandlerFunc.ServeHTTP)
	}
}

func joinWrapperFunc(original WrapperFunc, wrapperFunc WrapperFunc) WrapperFunc {
	if wrapperFunc == nil {
		return original
	}

	if original != nil {
		// wrap into one function, from bottom to top, end to begin.
		nextWrapper := wrapperFunc
		prevWrapper := original
		wrapperFunc = func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			if next != nil {
				nexthttpFunc := http.HandlerFunc(func(_w http.ResponseWriter, _r *http.Request) {
					prevWrapper(_w, _r, next)
				})
				nextWrapper(w, r, nexthttpFunc)
			}
		}
	}

	return wrapperFunc
}
