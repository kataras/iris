package iris_test

import (
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/httptest"
)

func testStatusErrors(routerPolicy iris.Policy, t *testing.T) {
	var (
		notFoundMessage        = "Iris custom message for 404 not found"
		internalServerMessage  = "Iris custom message for 500 internal server error"
		testRoutesCustomErrors = []testRoute{
			// NOT FOUND CUSTOM ERRORS - not registered
			{"GET", "/test_get_nofound_custom", "/test_get_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"POST", "/test_post_nofound_custom", "/test_post_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"PUT", "/test_put_nofound_custom", "/test_put_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"DELETE", "/test_delete_nofound_custom", "/test_delete_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"HEAD", "/test_head_nofound_custom", "/test_head_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"OPTIONS", "/test_options_nofound_custom", "/test_options_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"CONNECT", "/test_connect_nofound_custom", "/test_connect_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"PATCH", "/test_patch_nofound_custom", "/test_patch_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"TRACE", "/test_trace_nofound_custom", "/test_trace_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			// SERVER INTERNAL ERROR 500 PANIC CUSTOM ERRORS - registered
			{"GET", "/test_get_panic_custom", "/test_get_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"POST", "/test_post_panic_custom", "/test_post_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"PUT", "/test_put_panic_custom", "/test_put_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"DELETE", "/test_delete_panic_custom", "/test_delete_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"HEAD", "/test_head_panic_custom", "/test_head_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"OPTIONS", "/test_options_panic_custom", "/test_options_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"CONNECT", "/test_connect_panic_custom", "/test_connect_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"PATCH", "/test_patch_panic_custom", "/test_patch_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"TRACE", "/test_trace_panic_custom", "/test_trace_panic_custom", "", internalServerMessage, 500, true, nil, nil},
		}
	)
	app := iris.New()
	app.Adapt(routerPolicy)
	// first register the testRoutes needed
	for _, r := range testRoutesCustomErrors {
		if r.Register {
			app.HandleFunc(r.Method, r.Path, func(ctx *iris.Context) {
				ctx.EmitError(r.Status)
			})
		}
	}

	// register the custom errors
	app.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.Writef("%s", notFoundMessage)
	})

	app.OnError(iris.StatusInternalServerError, func(ctx *iris.Context) {
		ctx.Writef("%s", internalServerMessage)
	})

	// create httpexpect instance that will call fasthtpp.RequestHandler directly
	e := httptest.New(app, t)

	// run the tests
	for _, r := range testRoutesCustomErrors {
		e.Request(r.Method, r.RequestPath).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}
}

// Author's note:
// these are easy tests and can be done without special path parameters,
// so we have them here for all possible router policies.
func TestStatusErrors(t *testing.T) {
	testStatusErrors(httprouter.New(), t)
	testStatusErrors(gorillamux.New(), t)
}

func testStatusMethodNotAllowed(routerPolicy iris.Policy, t *testing.T) {
	app := iris.New()
	app.Adapt(routerPolicy)
	app.Config.FireMethodNotAllowed = true
	h := func(ctx *iris.Context) {
		ctx.WriteString(ctx.Method())
	}

	app.OnError(iris.StatusMethodNotAllowed, func(ctx *iris.Context) {
		ctx.WriteString("Hello from my custom 405 page")
	})

	app.Get("/mypath", h)
	app.Put("/mypath", h)

	e := httptest.New(app, t)

	e.GET("/mypath").Expect().Status(iris.StatusOK).Body().Equal("GET")
	e.PUT("/mypath").Expect().Status(iris.StatusOK).Body().Equal("PUT")
	// this should fail with 405 and catch by the custom http error

	e.POST("/mypath").Expect().Status(iris.StatusMethodNotAllowed).Body().Equal("Hello from my custom 405 page")
}

func TestStatusMethodNotAllowed(t *testing.T) {
	testStatusMethodNotAllowed(httprouter.New(), t)
	testStatusMethodNotAllowed(gorillamux.New(), t)
}
