package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

func TestRoutingFuncController(t *testing.T) {
	app := iris.New()
	opts := errors.NewOptions()
	opts.MapErrors(errors.Unauthenticated, ErrDomainLevelError)
	app.Use(opts.Handler())

	routingFunc := func(r *errors.Router) {
		registerRoutes(r, new(serviceImpl))
	}
	app.Router("/", routingFunc)

	e := httptest.New(t, app)
	testMethods(e)
}

func TestRouter(t *testing.T) {
	app := iris.New()

	r := errors.NewRouter(app)
	opts := errors.NewOptions()
	opts.MapErrors(errors.Unauthenticated, ErrDomainLevelError)
	r.Use(opts.HandlerWithError())

	registerRoutes(r, new(serviceImpl))

	e := httptest.New(t, app)
	testMethods(e)
}

var ErrDomainLevelError = fmt.Errorf("a domain error")

func registerRoutes(r *errors.Router, service Service) {
	r.Get("/", func(ctx iris.Context) error {
		return fmt.Errorf("a server error")
	})
	r.Get("/invalid", func(ctx iris.Context) error {
		return errors.InvalidArgument.Wrap(fmt.Errorf("an invalid argument error"))
	})
	r.Get("/domain", func(ctx iris.Context) error {
		return fmt.Errorf("%w: user not found", ErrDomainLevelError) // converted to 401 unauthorized, see app creation.
	})

	r.Get("/ok", func(ctx iris.Context) error {
		ctx.WriteString(service.Print("world"))
		return nil
	})
}

func testMethods(e *httptest.Expect) {
	e.GET("/").Expect().Status(httptest.StatusInternalServerError).JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.Internal,
			Status:        500,
		},
		Message: "server error",
		Detail:  "an unexpected error occurred while processing the request",
	})
	e.GET("/invalid").Expect().Status(httptest.StatusBadRequest).JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.InvalidArgument,
			Status:        400,
		},
		Message: "an invalid argument error",
	})
	e.GET("/domain").Expect().Status(httptest.StatusUnauthorized).JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.Unauthenticated,
			Status:        401,
		},
		Message: "a domain error: user not found",
	})
	e.GET("/ok").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello, world!")
}
