// Request-scoped dependencies: the function itself is injected, not its result.
//
// Ordinary registrations are singletons built at startup. A registered function
// whose input list contains a request type - iris.Context, context.Context,
// *http.Request or time.Time - is never called at registration time. The
// container registers the function value under its exact function type, and the
// consumer calls it with the live request. The container resolves none of its
// inputs: the caller supplies them all.
//
// Custom marker types can be added with
// dep.RegisterRequestDependencyType from github.com/kataras/iris/v14/core/dep,
// before any registration. A struct dependency may not hold a request-scoped
// field; registration fails and asks you to use a function instead.
//
// Learn: register a function that takes a request type and call it yourself
// with the live request inside the handler.
// Before: ../handler-func
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -H 'X-Request-ID: abc-123' http://localhost:8080/whoami   # 200, request_id abc-123
//	curl http://localhost:8080/whoami                              # 200, request_id anonymous
//
// Next: ../initializer
package main

import (
	"log"
	"time"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	// RegisterDependency returns the error instead of panicking. Both functions
	// below are stored as values; NewAuditor is called right here.
	if err := app.RegisterDependency(requestID, stamp, NewAuditor); err != nil {
		log.Fatal(err)
	}

	// The outer function runs once, at startup, with the two function values
	// and the singleton injected. The handler it returns is what serves
	// requests, calling the request-scoped functions with the live context.
	app.Get("/whoami", app.HandlerFunc(func(
		rid func(iris.Context) string,
		at func(time.Time) string,
		auditor *Auditor,
	) iris.Handler {
		return func(ctx iris.Context) {
			id, when := rid(ctx), at(time.Now())
			auditor.Record(id, when, ctx.Path())

			ctx.OK(iris.Map{"request_id": id, "at": when}, nil)
		}
	}))

	app.Listen(":8080")
}

// Auditor is a plain singleton, created once at registration. It mixes freely
// with request-scoped functions in the same injected parameter list.
type Auditor struct {
	logger *log.Logger
}

func NewAuditor() *Auditor {
	return &Auditor{logger: log.New(log.Writer(), "audit: ", 0)}
}

func (a *Auditor) Record(id, at, path string) {
	a.logger.Printf("%s %s %s", at, id, path)
}

// requestID is request-scoped because of its iris.Context input. It is
// injectable as func(iris.Context) string.
func requestID(ctx iris.Context) string {
	if id := ctx.Header("X-Request-ID"); id != "" {
		return id
	}

	return "anonymous"
}

// stamp is request-scoped because of its time.Time input. Nothing supplies the
// time for you - the handler passes the moment it cares about.
func stamp(now time.Time) string {
	return now.UTC().Format(time.RFC3339Nano)
}
