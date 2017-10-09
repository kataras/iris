package mvc

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/mvc/activator/methodfunc"
)

// build go1.9 only(go19.go)-->
// // Result is a response dispatcher.
// // All types that complete this interface
// // can be returned as values from the method functions.
// Result = methodfunc.Result
// <--
// No, let's just copy-paste in order to go 1.8 users have this type
// easy to be used from the root mvc package,
// sometimes duplication doesn't hurt.

// Result is a response dispatcher.
// All types that complete this interface
// can be returned as values from the method functions.
//
// Example at: https://github.com/kataras/iris/tree/master/_examples/mvc/using-method-result.
type Result interface { // NOTE: Should be always compatible with the methodfunc.Result.
	// Dispatch should sends the response to the context's response writer.
	Dispatch(ctx context.Context)
}

var defaultFailureResponse = Response{Code: methodfunc.DefaultErrStatusCode}

// Try will check if "fn" ran without any panics,
// using recovery,
// and return its result as the final response
// otherwise it returns the "failure" response if any,
// if not then a 400 bad request is being sent.
//
// Example usage at: https://github.com/kataras/iris/blob/master/mvc/method_result_test.go.
func Try(fn func() Result, failure ...Result) Result {
	var failed bool
	var actionResponse Result

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				failed = true
			}
		}()
		actionResponse = fn()
	}()

	if failed {
		if len(failure) > 0 {
			return failure[0]
		}
		return defaultFailureResponse
	}

	return actionResponse
}
