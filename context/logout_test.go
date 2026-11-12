package context_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
)

// Logout must call the registered logout function. Reading the stored
// value through unsafe.Pointer instead of a type assertion jumps to an
// invalid code address and kills the process.
func TestContextLogoutCallsRegisteredFunc(t *testing.T) {
	app := iris.New()
	app.Get("/logout", func(ctx iris.Context) {
		ctx.SetLogoutFunc(func(c *context.Context) error {
			c.WriteString("logged out")
			return nil
		})

		if err := ctx.Logout(); err != nil {
			ctx.StopWithStatus(httptest.StatusInternalServerError)
		}
	})

	// No logout function registered: Logout is a no-op that returns nil.
	app.Get("/noop", func(ctx iris.Context) {
		if err := ctx.Logout(); err != nil {
			ctx.StopWithStatus(httptest.StatusInternalServerError)
			return
		}
		ctx.WriteString("noop")
	})

	e := httptest.New(t, app)
	e.GET("/logout").Expect().Status(httptest.StatusOK).Body().IsEqual("logged out")
	e.GET("/noop").Expect().Status(httptest.StatusOK).Body().IsEqual("noop")
}
