package errors_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors" // to register the error handler.
)

func TestDefaultPathParameterTypeErrorHandler(t *testing.T) {
	app := iris.New()

	// Globally:
	// errors.ApplyPatches()
	// OR per-application:
	// app.SetContextErrorHandler(errors.DefaultContextErrorHandler)
	app.Macros().SetErrorHandler(errors.DefaultPathParameterTypeErrorHandler)

	app.Get("/users/{name:alphabetical}", func(ctx iris.Context) {
		ctx.Writef("Hello, %s!", ctx.Params().GetString("name"))
	})
	app.Get("/cats/{name:string}", func(ctx iris.Context) {
		ctx.Writef("Mew, %s!", ctx.Params().GetString("name"))
	})

	e := httptest.New(t, app)

	e.GET("/users/42").Expect().Status(httptest.StatusBadRequest).JSON().Object().HasValue("message", "invalid path parameter")
	e.GET("/users/Gerasimos").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello, Gerasimos!")
	e.GET("/cats/cute").Expect().Status(httptest.StatusOK).Body().IsEqual("Mew, cute!")
}
