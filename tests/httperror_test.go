package tests

import (
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/gavv/httpexpect/fasthttpexpect"
	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
)

var notFoundMessage = "Iris custom message for 404 not found"
var internalServerMessage = "Iris custom message for 500 internal server error"

var routesCustomErrors = []route{
	// NOT FOUND CUSTOM ERRORS - not registed
	route{"GET", "/test_get_nofound_custom", "/test_get_nofound_custom", notFoundMessage, 404, false, nil, nil},
	route{"POST", "/test_post_nofound_custom", "/test_post_nofound_custom", notFoundMessage, 404, false, nil, nil},
	route{"PUT", "/test_put_nofound_custom", "/test_put_nofound_custom", notFoundMessage, 404, false, nil, nil},
	route{"DELETE", "/test_delete_nofound_custom", "/test_delete_nofound_custom", notFoundMessage, 404, false, nil, nil},
	route{"HEAD", "/test_head_nofound_custom", "/test_head_nofound_custom", notFoundMessage, 404, false, nil, nil},
	route{"OPTIONS", "/test_options_nofound_custom", "/test_options_nofound_custom", notFoundMessage, 404, false, nil, nil},
	route{"CONNECT", "/test_connect_nofound_custom", "/test_connect_nofound_custom", notFoundMessage, 404, false, nil, nil},
	route{"PATCH", "/test_patch_nofound_custom", "/test_patch_nofound_custom", notFoundMessage, 404, false, nil, nil},
	route{"TRACE", "/test_trace_nofound_custom", "/test_trace_nofound_custom", notFoundMessage, 404, false, nil, nil},
	// SERVER INTERNAL ERROR 500 PANIC CUSTOM ERRORS - registed
	route{"GET", "/test_get_panic_custom", "/test_get_panic_custom", internalServerMessage, 500, true, nil, nil},
	route{"POST", "/test_post_panic_custom", "/test_post_panic_custom", internalServerMessage, 500, true, nil, nil},
	route{"PUT", "/test_put_panic_custom", "/test_put_panic_custom", internalServerMessage, 500, true, nil, nil},
	route{"DELETE", "/test_delete_panic_custom", "/test_delete_panic_custom", internalServerMessage, 500, true, nil, nil},
	route{"HEAD", "/test_head_panic_custom", "/test_head_panic_custom", internalServerMessage, 500, true, nil, nil},
	route{"OPTIONS", "/test_options_panic_custom", "/test_options_panic_custom", internalServerMessage, 500, true, nil, nil},
	route{"CONNECT", "/test_connect_panic_custom", "/test_connect_panic_custom", internalServerMessage, 500, true, nil, nil},
	route{"PATCH", "/test_patch_panic_custom", "/test_patch_panic_custom", internalServerMessage, 500, true, nil, nil},
	route{"TRACE", "/test_trace_panic_custom", "/test_trace_panic_custom", internalServerMessage, 500, true, nil, nil},
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

	api.PreListen(config.Server{ListeningAddr: ""})

	// create httpexpect instance that will call fasthtpp.RequestHandler directly
	e := httpexpect.WithConfig(httpexpect.Config{
		Reporter: httpexpect.NewAssertReporter(t),
		Client:   fasthttpexpect.NewBinder(api.ServeRequest),
	})
	// first register the custom errors

	api.OnError(404, func(ctx *iris.Context) {
		ctx.Write("%s", notFoundMessage)
	})

	api.OnError(500, func(ctx *iris.Context) {
		ctx.Write("%s", internalServerMessage)
	})

	// run the tests
	for _, r := range routesCustomErrors {
		e.Request(r.Method, r.RequestPath).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}
}
