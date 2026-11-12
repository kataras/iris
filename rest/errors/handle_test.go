package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// Handle is the guard form of Err: it states the code at the call site and
// reports whether it fired, so a nil error falls through to the next step.
func TestErrorCodeHandle(t *testing.T) {
	app := iris.New()

	app.Get("/fails", func(ctx iris.Context) {
		if errors.Unavailable.Handle(ctx, fmt.Errorf("upstream is down")) {
			return
		}
		ctx.WriteString("unreachable")
	})

	// A nil error writes nothing and reports false, so the handler continues.
	app.Get("/passes", func(ctx iris.Context) {
		if errors.Unavailable.Handle(ctx, nil) {
			return
		}
		ctx.WriteString("continued")
	})

	// Two steps, each naming the code its own failure means.
	app.Get("/second", func(ctx iris.Context) {
		if ctx.HandleError(nil) {
			return
		}
		if errors.PermissionDenied.Handle(ctx, fmt.Errorf("not your row")) {
			return
		}
		ctx.WriteString("unreachable")
	})

	e := httptest.New(t, app)

	e.GET("/fails").Expect().Status(httptest.StatusServiceUnavailable).
		Body().Contains("upstream is down")
	e.GET("/passes").Expect().Status(httptest.StatusOK).Body().IsEqual("continued")
	e.GET("/second").Expect().Status(httptest.StatusForbidden).
		Body().Contains("not your row")
}

// ctx.HandleError is the error half of ctx.OK and friends, on its own: it maps
// the error through the registry rather than forcing a code.
func TestContextHandleError(t *testing.T) {
	app := iris.New()

	app.Get("/mapped", func(ctx iris.Context) {
		if ctx.HandleError(errors.NotFound.Wrap(fmt.Errorf("plan 7 is gone"))) {
			return
		}
		ctx.WriteString("unreachable")
	})

	app.Get("/unmapped", func(ctx iris.Context) {
		if ctx.HandleError(fmt.Errorf("something the registry has never seen")) {
			return
		}
		ctx.WriteString("unreachable")
	})

	app.Get("/nil", func(ctx iris.Context) {
		if ctx.HandleError(nil) {
			return
		}
		ctx.WriteString("continued")
	})

	e := httptest.New(t, app)

	// A registered code keeps its status.
	e.GET("/mapped").Expect().Status(httptest.StatusNotFound)
	// An unrecognized error is a 500, as it is for OK, Created and NoContent.
	e.GET("/unmapped").Expect().Status(httptest.StatusInternalServerError)
	e.GET("/nil").Expect().Status(httptest.StatusOK).Body().IsEqual("continued")
}
