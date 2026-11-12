// Request IDs: one identifier per request, in the response header and the logs.
//
// requestid.New() installs the middleware. The default generator reuses a
// client-supplied X-Request-Id header and only mints a UUIDv4 when it is
// absent - convenient behind a trusted proxy, unsafe when arbitrary clients
// can choose the value that lands in your logs. requestid.ServerOnlyGenerator
// is the hardened variant: it ignores the inbound header and always generates
// the ID server-side.
//
// Read the ID with requestid.Get(ctx) (a shortcut for ctx.ID().(string)).
//
// Learn: install requestid, read the ID with requestid.Get, and pick
// ServerOnlyGenerator when clients must not choose the value.
// Before: ../rate-limit
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/trusted                                        # 200, a generated X-Request-Id
//	curl -i -H "X-Request-Id: chosen-by-client" http://localhost:8080/trusted    # 200, echoes chosen-by-client
//	curl -i -H "X-Request-Id: chosen-by-client" http://localhost:8080/hardened   # 200, ignores it and generates its own
//
// Next: ../accesslog
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/requestid"
)

func main() {
	app := iris.New()

	// A context.Handler passed to app.Router is registered with Use on the
	// group, so it applies to every route declared in the closure that
	// follows it.
	app.Router("/trusted", requestid.New(), func(r iris.Router) {
		r.Get("/", report)
	})

	// Same middleware, unforgeable IDs. Use this variant whenever request IDs
	// reach your logs or are trusted downstream.
	app.Router("/hardened", requestid.New(requestid.ServerOnlyGenerator), func(r iris.Router) {
		r.Get("/", report)
	})

	// requestid.HashGenerator(includeBody) is a third option: it derives a
	// fixed-length ID from a hash of the request, so identical requests share
	// an ID. Useful for grouping, not for uniqueness.

	app.Listen(":8080")
}

func report(ctx iris.Context) {
	// requestid.Get is the accessor to use. ctx.ID() returns the same value
	// as an "any", because the ID slot is not reserved for request IDs.
	ctx.JSON(iris.Map{
		"request_id": requestid.Get(ctx),
		// The middleware also reflects the ID in the response header, which is
		// what a client or a proxy correlates on.
		"response_header": ctx.ResponseHeader("X-Request-Id"),
	})
}
