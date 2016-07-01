package test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris"
)

type param struct {
	Key   string
	Value string
}

type route struct {
	Method       string
	Path         string
	RequestPath  string
	RequestQuery string
	Body         string
	Status       int
	Register     bool
	Params       []param
	UrlParams    []param
}

func TestMuxSimple(t *testing.T) {
	api := iris.New()
	routes := []route{
		// FOUND - registed
		{"GET", "/test_get", "/test_get", "", "hello, get!", 200, true, nil, nil},
		{"POST", "/test_post", "/test_post", "", "hello, post!", 200, true, nil, nil},
		{"PUT", "/test_put", "/test_put", "", "hello, put!", 200, true, nil, nil},
		{"DELETE", "/test_delete", "/test_delete", "", "hello, delete!", 200, true, nil, nil},
		{"HEAD", "/test_head", "/test_head", "", "hello, head!", 200, true, nil, nil},
		{"OPTIONS", "/test_options", "/test_options", "", "hello, options!", 200, true, nil, nil},
		{"CONNECT", "/test_connect", "/test_connect", "", "hello, connect!", 200, true, nil, nil},
		{"PATCH", "/test_patch", "/test_patch", "", "hello, patch!", 200, true, nil, nil},
		{"TRACE", "/test_trace", "/test_trace", "", "hello, trace!", 200, true, nil, nil},
		// NOT FOUND - not registed
		{"GET", "/test_get_nofound", "/test_get_nofound", "", "Not Found", 404, false, nil, nil},
		{"POST", "/test_post_nofound", "/test_post_nofound", "", "Not Found", 404, false, nil, nil},
		{"PUT", "/test_put_nofound", "/test_put_nofound", "", "Not Found", 404, false, nil, nil},
		{"DELETE", "/test_delete_nofound", "/test_delete_nofound", "", "Not Found", 404, false, nil, nil},
		{"HEAD", "/test_head_nofound", "/test_head_nofound", "", "Not Found", 404, false, nil, nil},
		{"OPTIONS", "/test_options_nofound", "/test_options_nofound", "", "Not Found", 404, false, nil, nil},
		{"CONNECT", "/test_connect_nofound", "/test_connect_nofound", "", "Not Found", 404, false, nil, nil},
		{"PATCH", "/test_patch_nofound", "/test_patch_nofound", "", "Not Found", 404, false, nil, nil},
		{"TRACE", "/test_trace_nofound", "/test_trace_nofound", "", "Not Found", 404, false, nil, nil},
		// Parameters
		{"GET", "/test_get_parameter1/:name", "/test_get_parameter1/iris", "", "name=iris", 200, true, []param{{"name", "iris"}}, nil},
		{"GET", "/test_get_parameter2/:name/details/:something", "/test_get_parameter2/iris/details/anything", "", "name=iris,something=anything", 200, true, []param{{"name", "iris"}, {"something", "anything"}}, nil},
		{"GET", "/test_get_parameter2/:name/details/:something/*else", "/test_get_parameter2/iris/details/anything/elsehere", "", "name=iris,something=anything,else=/elsehere", 200, true, []param{{"name", "iris"}, {"something", "anything"}, {"else", "elsehere"}}, nil},
		// URL Parameters
		{"GET", "/test_get_urlparameter1/first", "/test_get_urlparameter1/first", "name=irisurl", "name=irisurl", 200, true, nil, []param{{"name", "irisurl"}}},
		{"GET", "/test_get_urlparameter2/second", "/test_get_urlparameter2/second", "name=irisurl&something=anything", "name=irisurl,something=anything", 200, true, nil, []param{{"name", "irisurl"}, {"something", "anything"}}},
		{"GET", "/test_get_urlparameter2/first/second/third", "/test_get_urlparameter2/first/second/third", "name=irisurl&something=anything&else=elsehere", "name=irisurl,something=anything,else=elsehere", 200, true, nil, []param{{"name", "irisurl"}, {"something", "anything"}, {"else", "elsehere"}}},
	}

	for idx := range routes {
		r := routes[idx]
		if r.Register {
			api.HandleFunc(r.Method, r.Path, func(ctx *iris.Context) {
				ctx.SetStatusCode(r.Status)
				if r.Params != nil && len(r.Params) > 0 {
					ctx.SetBodyString(ctx.Params.String())
				} else if r.UrlParams != nil && len(r.UrlParams) > 0 {
					if len(r.UrlParams) != len(ctx.URLParams()) {
						t.Fatalf("Error when comparing length of url parameters %d != %d", len(r.UrlParams), len(ctx.URLParams()))
					}
					paramsKeyVal := ""
					for idxp, p := range r.UrlParams {
						val := ctx.URLParam(p.Key)
						paramsKeyVal += p.Key + "=" + val + ","
						if idxp == len(r.UrlParams)-1 {
							paramsKeyVal = paramsKeyVal[0 : len(paramsKeyVal)-1]
						}
					}
					ctx.SetBodyString(paramsKeyVal)
				} else {
					ctx.SetBodyString(r.Body)
				}

			})
		}
	}

	e := tester(api, t)

	// run the tests (1)
	for idx := range routes {
		r := routes[idx]
		e.Request(r.Method, r.RequestPath).WithQueryString(r.RequestQuery).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}

}

func TestMuxSimpleParty(t *testing.T) {

	api := iris.New()

	h := func(c *iris.Context) { c.WriteString(c.HostString() + c.PathString()) }

	if enable_subdomain_tests {
		subdomainParty := api.Party(subdomain + ".")
		{
			subdomainParty.Get("/", h)
			subdomainParty.Get("/path1", h)
			subdomainParty.Get("/path2", h)
			subdomainParty.Get("/namedpath/:param1/something/:param2", h)
			subdomainParty.Get("/namedpath/:param1/something/:param2/else", h)
		}
	}

	// simple
	p := api.Party("/party1")
	{
		p.Get("/", h)
		p.Get("/path1", h)
		p.Get("/path2", h)
		p.Get("/namedpath/:param1/something/:param2", h)
		p.Get("/namedpath/:param1/something/:param2/else", h)
	}

	e := tester(api, t)

	request := func(reqPath string) {
		e.Request("GET", reqPath).
			Expect().
			Status(iris.StatusOK).Body().Equal(host + reqPath)
	}

	// run the tests
	request("/party1/")
	request("/party1/path1")
	request("/party1/path2")
	request("/party1/namedpath/theparam1/something/theparam2")
	request("/party1/namedpath/theparam1/something/theparam2/else")

	if enable_subdomain_tests {
		subdomain_request := func(reqPath string) {
			e.Request("GET", subdomainURL+reqPath).
				Expect().
				Status(iris.StatusOK).Body().Equal(subdomainHost + reqPath)
		}

		subdomain_request("/")
		subdomain_request("/path1")
		subdomain_request("/path2")
		subdomain_request("/namedpath/theparam1/something/theparam2")
		subdomain_request("/namedpath/theparam1/something/theparam2/else")
	}
}

func TestMuxPathEscape(t *testing.T) {
	api := iris.New()

	api.Get("/details/:name", func(ctx *iris.Context) {
		name := ctx.Param("name")
		highlight := ctx.URLParam("highlight")
		ctx.Text(iris.StatusOK, fmt.Sprintf("name=%s,highlight=%s", name, highlight))
	})

	e := tester(api, t)

	e.GET("/details/Sakamoto desu ga").
		WithQuery("highlight", "text").
		Expect().Status(iris.StatusOK).Body().Equal("name=Sakamoto desu ga,highlight=text")
}

func TestMuxCustomErrors(t *testing.T) {
	var (
		notFoundMessage       = "Iris custom message for 404 not found"
		internalServerMessage = "Iris custom message for 500 internal server error"
		routesCustomErrors    = []route{
			// NOT FOUND CUSTOM ERRORS - not registed
			{"GET", "/test_get_nofound_custom", "/test_get_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"POST", "/test_post_nofound_custom", "/test_post_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"PUT", "/test_put_nofound_custom", "/test_put_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"DELETE", "/test_delete_nofound_custom", "/test_delete_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"HEAD", "/test_head_nofound_custom", "/test_head_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"OPTIONS", "/test_options_nofound_custom", "/test_options_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"CONNECT", "/test_connect_nofound_custom", "/test_connect_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"PATCH", "/test_patch_nofound_custom", "/test_patch_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"TRACE", "/test_trace_nofound_custom", "/test_trace_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			// SERVER INTERNAL ERROR 500 PANIC CUSTOM ERRORS - registed
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
		api = iris.New()
	)

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
	e := tester(api, t)

	// run the tests
	for _, r := range routesCustomErrors {
		e.Request(r.Method, r.RequestPath).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}
}
