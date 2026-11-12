package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

var errPlanNotFound = fmt.Errorf("plan not found")

func newHandleErrorfApp() *iris.Application {
	app := iris.New()

	opts := errors.NewOptions().MapErrors(errors.NotFound, errPlanNotFound)
	app.Use(opts.Handler())

	app.Get("/plans/{id}", func(ctx iris.Context) {
		errors.HandleErrorf(ctx, errPlanNotFound,
			"plan with id: %s", ctx.Params().GetString("id"))
	})

	app.Get("/nil", func(ctx iris.Context) {
		if !errors.HandleErrorf(ctx, nil, "never rendered %d", 42) {
			ctx.WriteString("no-error")
		}
	})

	app.Get("/unknown", func(ctx iris.Context) {
		errors.HandleErrorf(ctx, fmt.Errorf("boom"), "while doing %s", "work")
	})

	return app
}

// HandleErrorf must keep errors.Is/As classification (the sentinel still maps
// to 404) while enriching the rendered message with call-site detail.
func TestHandleErrorfKeepsMappingAndEnriches(t *testing.T) {
	e := httptest.New(t, newHandleErrorfApp())

	for _, path := range []string{"/plans/42"} {
		body := e.GET(path).Expect().Status(httptest.StatusNotFound).
			JSON().Object()

		// The call-site detail is the message; the wrapped cause is kept
		// in the data list.
		body.ContainsSubset(errors.Error{
			ErrorCode: errors.ErrorCode{
				CanonicalName: errors.NotFound,
				Status:        404,
			},
			Message: "plan with id: 42",
		})
		body.Value("data").Array().Value(0).Object().
			HasValue("message", "plan not found")
	}
}

func TestHandleErrorfNilError(t *testing.T) {
	e := httptest.New(t, newHandleErrorfApp())

	e.GET("/nil").Expect().Status(httptest.StatusOK).Body().IsEqual("no-error")
}

func TestHandleErrorfUnknownErrorIsInternal(t *testing.T) {
	e := httptest.New(t, newHandleErrorfApp())

	e.GET("/unknown").Expect().Status(httptest.StatusInternalServerError)
}
