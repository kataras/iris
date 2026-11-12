// Route groups: child routers that inherit a path prefix and middleware.
//
// app.Router(relativePath, ...) returns a child router that inherits the
// parent's path prefix and middleware. Groups nest to any depth, iris.Handler
// values passed next to the path become the group's middleware, and the
// closure form configures the child router in place.
//
// Learn: group routes under one prefix with app.Router, nest groups and attach
// middleware to a single group.
// Before: ../path-parameters
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/api/status                         # 200, X-Group: api
//	curl -i http://localhost:8080/api/v1/users/42                    # 200, X-Group: api, then X-Group: v1
//	curl http://localhost:8080/reports                               # 200, ["weekly","monthly"]
//	curl -i http://localhost:8080/admin/stats                        # 401
//	curl -i -H "X-Token: s3cret" http://localhost:8080/admin/stats   # 200
//
// Next: ../subdomains
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// A child router: every route below is prefixed with "/api".
	api := app.Router("/api")
	// Use on a child router runs only for that group's matched routes.
	api.Use(tag("api"))

	api.Get("/status", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"status": "ok"})
	})

	// Nested group: "/api" + "/v1". It inherits the "api" tag middleware too.
	v1 := api.Router("/v1")
	v1.Use(tag("v1"))

	v1.Get("/users", func(ctx iris.Context) {
		// ctx.List renders a slice as a JSON array, and an empty one as []
		// rather than null.
		ctx.List([]iris.Map{{"id": 1, "name": "Makis"}}, nil)
	})
	v1.Get("/users/{id:int}", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"id": ctx.Params().GetInt("id")})
	})

	// Closure form: the child router is passed to the function.
	app.Router("/reports", func(r iris.Router) {
		r.Use(tag("reports"))

		r.Get("/", func(ctx iris.Context) {
			ctx.List([]string{"weekly", "monthly"}, nil)
		})
		r.Get("/{name:string}", func(ctx iris.Context) {
			ctx.JSON(iris.Map{"report": ctx.Params().GetString("name")})
		})
	})

	// Handlers listed after the path become the group's middleware.
	admin := app.Router("/admin", requireToken)
	admin.Get("/stats", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"requests": 1024})
	})

	app.Listen(":8080")
}

// tag records the group name on the response. AddHeader appends (SetHeader
// would replace), so a nested route lists one X-Group value per group it
// passed through.
func tag(name string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.AddHeader("X-Group", name)
		ctx.Next()
	}
}

// requireToken stops the chain when the token header is missing.
// errors.Unauthenticated carries the 401 and stops the chain itself, so the
// middleware only has to return.
func requireToken(ctx iris.Context) {
	if ctx.Header("X-Token") != "s3cret" {
		errors.Unauthenticated.Message(ctx, "missing or invalid X-Token header")
		return
	}

	ctx.Next()
}
