package identity

import (
	"time"

	"github.com/kataras/iris"

	"github.com/kataras/iris/_examples/structuring/bootstrap/bootstrap"
)

// New returns a new handler which adds some headers and view data
// describing the application, i.e the owner, the startup time.
func New(b *bootstrap.Bootstrapper) iris.Handler {
	return func(ctx iris.Context) {
		// response headers
		ctx.Header("App-Name", b.AppName)
		ctx.Header("App-Owner", b.AppOwner)
		ctx.Header("App-Since", time.Since(b.AppSpawnDate).String())

		ctx.Header("Server", "Iris: https://iris-go.com")

		// view data if ctx.View or c.Tmpl = "$page.html" will be called next.
		ctx.ViewData("AppName", b.AppName)
		ctx.ViewData("AppOwner", b.AppOwner)
		ctx.Next()
	}
}

// Configure creates a new identity middleware and registers that to the app.
func Configure(b *bootstrap.Bootstrapper) {
	h := New(b)
	b.UseGlobal(h)
}
