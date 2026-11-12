// Path parameters: typed macros in the route pattern, typed accessors in the
// handler.
//
// A macro such as {id:int} validates and converts the segment BEFORE any
// handler runs, so a handler that runs can trust its input. A request that
// does not match never reaches your code: the router answers 404 by default,
// or the status code given by a trailing "else N".
//
// ctx.Params() has typed accessors with fixed failure defaults. ctx.BindParam
// is the helper form: it converts the parameter and, when that cannot be done,
// answers the client itself (404 when the parameter is missing, 400 when it
// has another type) and reports false, so the handler's only branch is a
// one-line return. With a typed macro such as {id:int} the router has already
// rejected a non-integer, so that branch fires only for a handler reused on a
// route that does not carry the parameter.
//
// Learn: register typed path parameters and let the router reject bad values
// with 404, or with the status of a trailing "else N", before your handler runs.
// Before: ../routing-basics
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/greet/iris                                   # 200, {"message":"hello, iris"}
//	curl http://localhost:8080/orders/42                                    # 200, {"order":42,"status":"pending"}
//	curl -i http://localhost:8080/orders/abc                                # 404, "abc" is not an int
//	curl http://localhost:8080/shelves/9f86d081-8292-4c4c-812f-0a1b2c3d4e5f  # 200, {"shelf":"9f86d081-..."}
//	curl -i http://localhost:8080/page/0                                    # 400, from "else 400"
//	curl http://localhost:8080/page/7                                       # 200, {"page":7}
//	curl http://localhost:8080/carts/3                                      # 200, {"cart":3}
//
// Next: ../groups
package main

import (
	"log"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	// {name:string} matches any single path segment. It is the default type,
	// so {name} means the same thing.
	app.Get("/greet/{name:string}", func(ctx iris.Context) {
		// GetString returns "" when the parameter is missing.
		name := ctx.Params().GetString("name")
		ctx.JSON(iris.Map{"message": "hello, " + name})
	})

	// {id:int} matches integers only.
	app.Get("/orders/{id:int}", func(ctx iris.Context) {
		// GetInt returns -1 on a missing key or a failed conversion. Here the
		// macro has already guaranteed the value, so -1 cannot happen.
		id := ctx.Params().GetInt("id")
		ctx.JSON(iris.Map{"order": id, "status": "pending"})
	})

	// {uid:uuid} matches the UUID format (v4 and v1) and stores the parsed
	// uuid.UUID, so GetUUID reads it back without parsing again.
	app.Get("/shelves/{uid:uuid}", func(ctx iris.Context) {
		uid := ctx.Params().GetUUID("uid")

		// Lookup reports presence, for the cases where the zero value of an
		// accessor is ambiguous.
		if raw, ok := ctx.Params().Lookup("uid"); ok {
			log.Printf("shelf parameter: %v", raw)
		}

		ctx.JSON(iris.Map{"shelf": uid})
	})

	// {addr:email} checks the address syntax AND looks up the domain's MX
	// records, so it needs working DNS. Use {addr:mail} for the offline,
	// syntax-only check.
	app.Get("/verify/{addr:email}", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"verify": ctx.Params().GetString("addr")})
	})

	// ctx.BindParam[T] is ctx.Params() plus the error response: it renders
	// the 404 or the 400 and reports false, so the handler just returns. Here
	// the {id:int} macro already guarantees an integer; the branch is what
	// keeps this handler correct if it is ever mounted without the parameter.
	app.Get("/carts/{id:int}", func(ctx iris.Context) {
		id, ok := ctx.BindParam[int]("id")
		if !ok {
			return // 404 or 400 already written.
		}

		ctx.OK(iris.Map{"cart": id}, nil)
	})

	// Constraints follow the type, separated by spaces, and a trailing
	// "else N" sets the status code the router answers with when one fails.
	// Without it the default is 404.
	app.Get("/page/{page:int min(1) max(100) else 400}", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"page": ctx.Params().GetInt("page")})
	})

	app.Listen(":8080")
}
