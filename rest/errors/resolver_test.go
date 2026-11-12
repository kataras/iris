package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

var (
	errResolvedByResolver = fmt.Errorf("resolved by resolver")
	errMappedAndResolved  = fmt.Errorf("mapped and resolved")
)

func init() {
	// Resolvers are process-global; match only this test's sentinels.
	errors.RegisterErrorResolver(func(ctx iris.Context, err error) bool {
		if errors.Is(err, errResolvedByResolver) || errors.Is(err, errMappedAndResolved) {
			errors.ResourceExhausted.Message(ctx, "handled by resolver")
			return true
		}
		return false
	})
}

// A registered resolver handles errors that no per-route code map claimed.
func TestErrorResolverHandlesUnmappedError(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		errors.HandleError(ctx, errResolvedByResolver)
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusTooManyRequests).
		JSON().Object().HasValue("message", "handled by resolver")
}

// The per-route Options code map runs before resolvers and wins.
func TestErrorResolverRunsAfterCodeMap(t *testing.T) {
	app := iris.New()
	opts := errors.NewOptions().MapErrors(errors.NotFound, errMappedAndResolved)
	app.Use(opts.Handler())

	app.Get("/", func(ctx iris.Context) {
		errors.HandleError(ctx, errMappedAndResolved)
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusNotFound)
}
