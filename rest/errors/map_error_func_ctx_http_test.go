package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

var errCtxSentinel = fmt.Errorf("ctx sentinel")

// A zero-value Options must be usable, matching MapErrorFunc and MapErrors.
func TestOptionsMapErrorFuncCtxZeroValue(t *testing.T) {
	var opts errors.Options
	opts.MapErrorFuncCtx(errors.NotFound, func(ctx iris.Context, err error) error {
		return nil
	}) // must not panic on the nil internal map.
}

// MapErrorFuncCtx mappers see the request context, so the transformed error
// can carry request detail (here: the path).
func TestOptionsMapErrorFuncCtx(t *testing.T) {
	app := iris.New()

	opts := errors.NewOptions().MapErrorFuncCtx(errors.NotFound,
		func(ctx iris.Context, err error) error {
			if errors.Is(err, errCtxSentinel) {
				return fmt.Errorf("%s: %w", ctx.Path(), err)
			}
			return nil
		})
	app.Use(opts.Handler())

	app.Get("/things/{id}", func(ctx iris.Context) {
		errors.HandleError(ctx, errCtxSentinel)
	})

	e := httptest.New(t, app)

	body := e.GET("/things/42").Expect().Status(httptest.StatusNotFound).
		JSON().Object()
	body.ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.NotFound,
			Status:        404,
		},
		Message: "/things/42",
	})
}
