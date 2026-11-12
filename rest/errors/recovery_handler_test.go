package errors_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

func TestRecoveryHandler(t *testing.T) {
	app := iris.New()
	app.Use(errors.RecoveryHandler)

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"message": "Hello, world!"})
	})
	app.Get("/err", func(ctx iris.Context) {
		panic("a panic error")
	})

	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK).JSON().Object().HasValue("message", "Hello, world!")
	e.GET("/err").Expect().Status(httptest.StatusInternalServerError).JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.Internal,
			Status:        500,
		},
		Message: "server error",
		Detail:  "an unexpected error occurred while processing the request",
	})
}
