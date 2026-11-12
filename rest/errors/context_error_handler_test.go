package errors_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors" // to register the error handler too.
)

func TestDefaultContextErrorHandler(t *testing.T) {
	app := iris.New()
	// Globally:
	// errors.ApplyPatches()
	// OR per-application:
	app.Use(func(ctx iris.Context) {
		ctx.Options().ErrorHandler = errors.DefaultContextErrorHandler
		ctx.Next()
	})
	// app.Macros().SetErrorHandler(errors.DefaultPathParameterTypeErrorHandler)

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"message": "Hello, world!"})
	})
	app.Get("/err", func(ctx iris.Context) {
		type InvalidJSON struct {
			Func func() // Functions are not supported in JSON
		}
		invalid := InvalidJSON{
			Func: func() {
				// do something
			},
		}
		ctx.JSON(invalid)
	})

	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK).JSON().Object().HasValue("message", "Hello, world!")
	e.GET("/err").Expect().Status(httptest.StatusInternalServerError).JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.Internal,
			Status:        500,
		},
		Message: "unable to parse response body",
		// The detail names the offending member by its JSON pointer — a
		// wire-level fact — instead of the old encoding/json text, which
		// leaked the server's Go type ("json: unsupported type: func()").
		Detail: `invalid value at "/Func"`,
	})
}
