// Custom path macros: a new constraint on a built-in parameter type, and a
// parameter type of your own.
//
// app.Macros() exposes the registry behind {id:int}, {id:uuid} and the rest of
// the built-in path parameter types. Two things can be added to it:
//
//   - Get(type).RegisterFunc(name, fn) adds a constraint function to an
//     existing type. The outer function receives the compile-time arguments
//     written in the route pattern, the returned function validates the typed
//     value of every request.
//   - Register(...) adds a whole new parameter type with its own evaluator,
//     which converts the raw path segment to a Go value or reports failure.
//
// Both must happen before any route is registered. The registry belongs to
// this application: it starts as a copy of the built-in types, and what is
// registered on it does not reach another application in the same binary.
//
// When the evaluator fails with an error, the 400 body is the rest/errors
// envelope:
//
//	{
//	  "id": "...",
//	  "http_error_code": {"canonical_name":"INVALID_ARGUMENT","status":400},
//	  "message": "invalid path parameter",
//	  "detail": "time: invalid duration \"nope\"",
//	  "data": "nope"
//	}
//
// Learn: add a constraint function to a built-in path parameter type, or
// register a new type with its own evaluator and 400 error handler.
// Before: ../subdomains
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/pages/30          # 200, 30 is divisible by 10
//	curl -i http://localhost:8080/pages/33       # 404, no route matched
//	curl http://localhost:8080/cache/1h30m       # 200, typed time.Duration
//	curl -i http://localhost:8080/cache/nope     # 400, from the error handler
//
// Next: ../wrap-router
package main

import (
	"fmt"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// A new constraint on the built-in int type. The returned function
	// receives the already evaluated value, so it is a func(int) bool.
	app.Macros().Get("int").RegisterFunc("divisible",
		func(divisor int) func(int) bool {
			return func(paramValue int) bool {
				return divisor != 0 && paramValue%divisor == 0
			}
		})

	// A brand new parameter type. The arguments after the alias ("") are the
	// sample value - its type becomes the parameter's Go type - whether this
	// type is the master default and whether it is trailing (consumes the rest
	// of the path, like the built-in "path" type).
	//
	// Returning a non-nil error together with false is what makes the value
	// reach SetErrorHandler; returning a plain nil falls back to the route's
	// error code (404 by default, or the "else" literal).
	//
	// errors.DefaultPathParameterTypeErrorHandler is the canonical handler:
	// it renders the 400 rest/errors envelope with the offending value under
	// "data". app.Macros().SetErrorHandler installs it for every macro at
	// once, and errors.ApplyPatches() installs it process-wide.
	app.Macros().Register("duration", "", time.Duration(0), false, false,
		func(paramValue string) (any, bool) {
			d, err := time.ParseDuration(paramValue)
			if err != nil {
				return err, false
			}

			if d <= 0 {
				return fmt.Errorf("duration must be positive, got %s", d), false
			}

			return d, true
		}).SetErrorHandler(errors.DefaultPathParameterTypeErrorHandler)

	// A constraint failure is a routing failure: the request simply does not
	// match this route.
	app.Get("/pages/{n:int divisible(10)}", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"page": ctx.Params().GetInt("n")})
	})

	// GetDuration returns the macro's typed value as is. Had the value been a
	// string, it would parse it with time.ParseDuration.
	app.Get("/cache/{ttl:duration}", func(ctx iris.Context) {
		ttl := ctx.Params().GetDuration("ttl")
		ctx.JSON(iris.Map{
			"ttl_seconds": ttl.Seconds(),
			"expires_at":  time.Now().Add(ttl).Format(time.RFC3339),
		})
	})

	app.Listen(":8080")
}
