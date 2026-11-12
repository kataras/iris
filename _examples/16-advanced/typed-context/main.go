// Custom typed contexts: handlers written against your own context type through iris.NewContextWrapper.
//
// Handlers that keep reading the same values out of ctx.Values() and casting
// them accumulate noise. iris.NewContextWrapper lets you define your own
// context type, embedding iris.Context, and write handlers against it.
//
// The type parameter is the struct; its pointer must implement
// SetContext(iris.Context), which the wrapper calls per request before running
// the handler. w.Handler converts a func(*appContext) into a plain
// iris.Handler. Handlers (a slice at once), HandlerReturnError,
// HandlerReturnDuration and Filter cover the other shapes Iris expects.
//
// The default pool allocates a zero value per request. To recycle instances
// instead, build the wrapper from the context subpackage:
//
//	import "github.com/kataras/iris/v14/context"
//	w := context.NewContextWrapperWithPool(context.NewContextPool[appContext]())
//
// With a recycling pool SetContext is the reset hook: it runs on every acquire,
// so clear any custom fields there, because instances outlive a request.
//
// Learn: give your handlers a context type with your own helper methods, for middleware and routes alike.
// Before: ../std-interop
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -H "X-Tenant: acme" http://localhost:8080/dashboard    # 200 {"tenant":"acme"}
//	curl -i http://localhost:8080/dashboard                     # 400, no tenant
//	curl -H "X-Tenant: acme" http://localhost:8080/tenants/42   # 200 {"tenant":"acme","id":42}
//
// Next: ../file-server
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	w := iris.NewContextWrapper[appContext]()

	// Middleware written against the custom type as well.
	app.Use(w.Handler(func(ctx *appContext) {
		tenant := ctx.Header("X-Tenant")
		if tenant == "" {
			ctx.BadRequest("missing X-Tenant header")
			return
		}

		ctx.Values().Set("tenant_id", tenant)
		ctx.Next()
	}))

	app.Get("/dashboard", w.Handler(func(ctx *appContext) {
		ctx.JSON(iris.Map{"tenant": ctx.TenantID()})
	}))

	// Everything iris.Context offers is still there, through the embedded field.
	app.Get("/tenants/{id:int}", w.Handler(func(ctx *appContext) {
		ctx.JSON(iris.Map{
			"tenant": ctx.TenantID(),
			"id":     ctx.Params().GetInt("id"),
		})
	}))

	app.Listen(":8080")
}

// appContext extends iris.Context with application-specific helpers.
type appContext struct {
	iris.Context
}

// SetContext makes *appContext usable by the wrapper's default pool.
// It is also the reset hook when a recycling pool is used.
func (c *appContext) SetContext(ctx iris.Context) {
	c.Context = ctx
}

// TenantID reads the value that the middleware above stored.
func (c *appContext) TenantID() string {
	return c.Values().GetString("tenant_id")
}

// BadRequest is a small helper of our own, available on every handler. It
// delegates to rest/errors, so a custom context does not mean a custom error
// shape: the error code carries the 400 and stops the chain.
func (c *appContext) BadRequest(msg string) {
	errors.InvalidArgument.Message(c.Context, msg)
}
