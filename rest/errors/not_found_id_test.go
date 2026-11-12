package errors_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// NotFoundID renders the "<resource> <id> not found" envelope and reports
// false, so a handler can return it directly.
func TestNotFoundID(t *testing.T) {
	app := iris.New()

	// Recorded here and asserted on the test goroutine: t.Fatal from inside a
	// handler leaves the response half written.
	var reported bool

	app.Get("/users/{id}", func(ctx iris.Context) {
		reported = errors.NotFoundID(ctx, "user", ctx.Params().GetString("id"))
	})

	app.Get("/plans/{id:int}", func(ctx iris.Context) {
		errors.NotFoundID(ctx, "plan", ctx.Params().GetInt("id"))
	})

	e := httptest.New(t, app)

	e.GET("/users/abc-123").Expect().Status(httptest.StatusNotFound).
		JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.NotFound,
			Status:        httptest.StatusNotFound,
		},
		Detail: "user abc-123 not found",
	})

	// False: the ctx.OK convention, an error was written so the handler did
	// not succeed. HandleError in the same package reports the opposite.
	if reported {
		t.Fatal("expected NotFoundID to report false: an error was written")
	}

	// A non-string id is formatted with %v.
	e.GET("/plans/42").Expect().Status(httptest.StatusNotFound).
		JSON().Object().HasValue("detail", "plan 42 not found")
}
