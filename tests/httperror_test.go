package tests

import (
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/gavv/httpexpect/fasthttpexpect"
	"github.com/kataras/iris"
)

var notFoundMessage = "Iris custom message for 404 not found"
var internalServerMessage = "Iris custom message for 500 internal server error"

var routesCustomErrors = []route{
	// NOT FOUND CUSTOM ERRORS - not registed
	{"GET", "/test_get_nofound_custom", "/test_get_nofound_custom", notFoundMessage, 404, false, nil, nil},
	{"POST", "/test_post_nofound_custom", "/test_post_nofound_custom", notFoundMessage, 404, false, nil, nil},
	{"PUT", "/test_put_nofound_custom", "/test_put_nofound_custom", notFoundMessage, 404, false, nil, nil},
	{"DELETE", "/test_delete_nofound_custom", "/test_delete_nofound_custom", notFoundMessage, 404, false, nil, nil},
	{"HEAD", "/test_head_nofound_custom", "/test_head_nofound_custom", notFoundMessage, 404, false, nil, nil},
	{"OPTIONS", "/test_options_nofound_custom", "/test_options_nofound_custom", notFoundMessage, 404, false, nil, nil},
	{"CONNECT", "/test_connect_nofound_custom", "/test_connect_nofound_custom", notFoundMessage, 404, false, nil, nil},
	{"PATCH", "/test_patch_nofound_custom", "/test_patch_nofound_custom", notFoundMessage, 404, false, nil, nil},
	{"TRACE", "/test_trace_nofound_custom", "/test_trace_nofound_custom", notFoundMessage, 404, false, nil, nil},
	// SERVER INTERNAL ERROR 500 PANIC CUSTOM ERRORS - registed
	{"GET", "/test_get_panic_custom", "/test_get_panic_custom", internalServerMessage, 500, true, nil, nil},
	{"POST", "/test_post_panic_custom", "/test_post_panic_custom", internalServerMessage, 500, true, nil, nil},
	{"PUT", "/test_put_panic_custom", "/test_put_panic_custom", internalServerMessage, 500, true, nil, nil},
	{"DELETE", "/test_delete_panic_custom", "/test_delete_panic_custom", internalServerMessage, 500, true, nil, nil},
	{"HEAD", "/test_head_panic_custom", "/test_head_panic_custom", internalServerMessage, 500, true, nil, nil},
	{"OPTIONS", "/test_options_panic_custom", "/test_options_panic_custom", internalServerMessage, 500, true, nil, nil},
	{"CONNECT", "/test_connect_panic_custom", "/test_connect_panic_custom", internalServerMessage, 500, true, nil, nil},
	{"PATCH", "/test_patch_panic_custom", "/test_patch_panic_custom", internalServerMessage, 500, true, nil, nil},
	{"TRACE", "/test_trace_panic_custom", "/test_trace_panic_custom", internalServerMessage, 500, true, nil, nil},
}

func TestCustomErrors(t *testing.T) {
	api := iris.New()
	// first register the routes needed
	for _, r := range routesCustomErrors {
		if r.Register {
			api.HandleFunc(r.Method, r.Path, func(ctx *iris.Context) {
				ctx.EmitError(r.Status)
			})
		}
	}

	// register the custom errors
	api.OnError(404, func(ctx *iris.Context) {
		ctx.Write("%s", notFoundMessage)
	})

	api.OnError(500, func(ctx *iris.Context) {
		ctx.Write("%s", internalServerMessage)
	})

	// create httpexpect instance that will call fasthtpp.RequestHandler directly
	e := httpexpect.WithConfig(httpexpect.Config{
		Reporter: httpexpect.NewAssertReporter(t),
		Client:   fasthttpexpect.NewBinder(api.NoListen().Handler),
	})

	// run the tests
	for _, r := range routesCustomErrors {
		e.Request(r.Method, r.RequestPath).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}
}
