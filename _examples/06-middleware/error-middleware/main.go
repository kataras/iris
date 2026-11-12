// Middleware that returns an error instead of writing a response.
//
// Middleware doing real work - a plan lookup, a database call - reads better
// when it can return an error instead of hand-writing a response.
// errors.FuncMiddleware from rest/errors converts a func(iris.Context) error
// into a normal iris.Handler:
//
//   - a nil return calls ctx.Next() and the chain continues;
//   - a non-nil return goes through the central error pipeline (mapping,
//     logging, JSON rendering) and the request stops there.
//
// The error's code name picks the status and the JSON "code":
// errors.Unauthenticated renders 401 UNAUTHENTICATED, errors.PermissionDenied
// 403 PERMISSION_DENIED, errors.InvalidArgument 400 INVALID_ARGUMENT. An
// unrecognized error becomes 500 INTERNAL and its text is logged rather than
// returned. errors.FuncHandler is the same conversion for a terminal handler
// (no Next on success), and errors.FuncHandlers converts a whole slice.
//
// Authentication is NOT the thing to hand-write here: middleware/apikey
// already compares the presented key against the configured ones in constant
// time and renders the same 401 envelope, so it runs first and this example's
// own middleware is left with the question it alone can answer, which
// principal may read the data.
//
// Learn: turn a func(iris.Context) error into middleware or a handler with
// errors.FuncMiddleware and errors.FuncHandler, and let the error code pick the
// status.
// Before: ../conditional
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/data                                # 401 UNAUTHENTICATED, no key
//	curl -i -H "X-API-Key: nope" http://localhost:8080/data           # 401 UNAUTHENTICATED, unknown key
//	curl -i -H "X-API-Key: trial-key" http://localhost:8080/data      # 403 PERMISSION_DENIED
//	curl -i -H "X-API-Key: demo-key" http://localhost:8080/data       # 200
//	curl -i -H "X-API-Key: demo-key" http://localhost:8080/failing    # 500 INTERNAL
//
// Next: ../cors
package main

import (
	"fmt"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/apikey"
	"github.com/kataras/iris/v14/rest/errors"
)

// readers are the principals allowed past requireReadAccess. Every other
// configured key authenticates but reads nothing.
var readers = map[string]bool{"demo": true}

func main() {
	app := iris.New()

	// Authentication. A bare key on a custom header, so Scheme is NoScheme;
	// the default is "Authorization: Bearer <key>". A missing or unknown key
	// is answered here, before anything below runs.
	app.Use(apikey.New(apikey.Options{
		Header: "X-API-Key",
		Scheme: apikey.NoScheme,
		Keys: map[string]string{
			"demo":  "demo-key",
			"trial": "trial-key",
		},
	}))

	// Authorization, as error-returning middleware registered like any other
	// handler.
	app.Use(errors.FuncMiddleware(requireReadAccess))

	// The terminal-handler counterpart.
	app.Get("/data", errors.FuncHandler(listData))
	app.Get("/failing", errors.FuncHandler(brokenDependency))

	app.Listen(":8080")
}

// requireReadAccess decides what the key is allowed to do. Returning an error
// is all it takes to stop the request with a rendered JSON body - no StopWith*
// call and no manual status code here.
func requireReadAccess(ctx iris.Context) error {
	// apikey.Get names the principal whose key matched. It is never empty
	// below the middleware, so the second result is the answer to "did this
	// request go through apikey at all", not to "is this a real caller".
	principal, _ := apikey.Get(ctx)

	if !readers[principal] {
		return errors.PermissionDenied.Wrap(
			fmt.Errorf("%s may not read the data", principal))
	}

	return nil // continue the chain.
}

func listData(ctx iris.Context) error {
	principal, _ := apikey.Get(ctx)

	ctx.JSON(iris.Map{
		"principal": principal,
		"items":     []string{"alpha", "beta"},
	})

	return nil
}

// brokenDependency returns a plain, unwrapped error: the pipeline classifies
// it as INTERNAL, answers 500 and keeps the message server-side.
func brokenDependency(ctx iris.Context) error {
	return fmt.Errorf("inventory service is unreachable")
}
