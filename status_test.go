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

func testRegisterRegex(routerPolicy iris.Policy, t *testing.T) {

	app := iris.New()
	app.Adapt(routerPolicy)

	h := func(ctx *iris.Context) {
		ctx.WriteString(ctx.Method())
	}

	app.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.WriteString("root 404")
	})

	staticP := app.Party("/static")
	{
		// or app.Errors... same thing
		staticP.Errors.RegisterRegex(iris.StatusNotFound, iris.HandlerFunc(func(ctx *iris.Context) {
			ctx.WriteString("/static 404")
		}), "/static/[0-9]+")

		// simulate a StaticHandler or StaticWeb simple behavior
		// in order to get a not found error from a wildcard path registered on the root path of the "/static".
		// Note:
		// RouteWildcardPath when you want to work on all router adaptors:httprouter=> *file, gorillamux=> {file:.*}
		staticP.Get(app.RouteWildcardPath("/", "file"), func(ctx *iris.Context) {

			i, err := ctx.ParamIntWildcard("file")
			if i > 0 || err == nil {
				// this handler supposed to accept only strings, for the sake of the test.
				ctx.EmitError(iris.StatusNotFound)
				return
			}

			ctx.SetStatusCode(iris.StatusOK)
		})

	}

	app.Get("/mypath", h)

	e := httptest.New(app, t)
	// print("-------------------TESTING ")
	// println(app.Config.Other[iris.RouterNameConfigKey].(string) + "-------------------")

	e.GET("/mypath").Expect().Status(iris.StatusOK).Body().Equal("GET")
	e.GET("/rootnotfound").Expect().Status(iris.StatusNotFound).Body().Equal("root 404")

	// test found on party
	e.GET("/static/itshouldbeok").Expect().Status(iris.StatusOK)
	// test no found on party ( by putting at least one integer after /static)
	e.GET("/static/42").Expect().Status(iris.StatusNotFound).Body().Equal("/static 404")

	// println("-------------------------------------------------------")
}

func TestRegisterRegex(t *testing.T) {
	testRegisterRegex(httprouter.New(), t)
	testRegisterRegex(gorillamux.New(), t)
}
