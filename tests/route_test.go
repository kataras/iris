package tests

import (
	"fmt"
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/gavv/httpexpect/fasthttpexpect"
	"github.com/kataras/iris"
)

type param struct {
	Key   string
	Value string
}

type route struct {
	Method      string
	Path        string
	RequestPath string
	Body        string
	Status      int
	Register    bool
	Params      []param
	UrlParams   []param
}

var routes = []route{
	// FOUND - registed
	{"GET", "/test_get", "/test_get", "hello, get!", 200, true, nil, nil},
	{"POST", "/test_post", "/test_post", "hello, post!", 200, true, nil, nil},
	{"PUT", "/test_put", "/test_put", "hello, put!", 200, true, nil, nil},
	{"DELETE", "/test_delete", "/test_delete", "hello, delete!", 200, true, nil, nil},
	{"HEAD", "/test_head", "/test_head", "hello, head!", 200, true, nil, nil},
	{"OPTIONS", "/test_options", "/test_options", "hello, options!", 200, true, nil, nil},
	{"CONNECT", "/test_connect", "/test_connect", "hello, connect!", 200, true, nil, nil},
	{"PATCH", "/test_patch", "/test_patch", "hello, patch!", 200, true, nil, nil},
	{"TRACE", "/test_trace", "/test_trace", "hello, trace!", 200, true, nil, nil},
	// NOT FOUND - not registed
	{"GET", "/test_get_nofound", "/test_get_nofound", "Not Found", 404, false, nil, nil},
	{"POST", "/test_post_nofound", "/test_post_nofound", "Not Found", 404, false, nil, nil},
	{"PUT", "/test_put_nofound", "/test_put_nofound", "Not Found", 404, false, nil, nil},
	{"DELETE", "/test_delete_nofound", "/test_delete_nofound", "Not Found", 404, false, nil, nil},
	{"HEAD", "/test_head_nofound", "/test_head_nofound", "Not Found", 404, false, nil, nil},
	{"OPTIONS", "/test_options_nofound", "/test_options_nofound", "Not Found", 404, false, nil, nil},
	{"CONNECT", "/test_connect_nofound", "/test_connect_nofound", "Not Found", 404, false, nil, nil},
	{"PATCH", "/test_patch_nofound", "/test_patch_nofound", "Not Found", 404, false, nil, nil},
	{"TRACE", "/test_trace_nofound", "/test_trace_nofound", "Not Found", 404, false, nil, nil},
	// Parameters
	{"GET", "/test_get_parameter1/:name", "/test_get_parameter1/iris", "name=iris", 200, true, []param{{"name", "iris"}}, nil},
	{"GET", "/test_get_parameter2/:name/details/:something", "/test_get_parameter2/iris/details/anything", "name=iris,something=anything", 200, true, []param{{"name", "iris"}, {"something", "anything"}}, nil},
	{"GET", "/test_get_parameter2/:name/details/:something/*else", "/test_get_parameter2/iris/details/anything/elsehere", "name=iris,something=anything,else=/elsehere", 200, true, []param{{"name", "iris"}, {"something", "anything"}, {"else", "elsehere"}}, nil},
	// URL Parameters
	{"GET", "/test_get_urlparameter1/first", "/test_get_urlparameter1/first?name=irisurl", "name=irisurl", 200, true, nil, []param{{"name", "irisurl"}}},
	{"GET", "/test_get_urlparameter2/second", "/test_get_urlparameter2/second?name=irisurl&something=anything", "name=irisurl,something=anything", 200, true, nil, []param{{"name", "irisurl"}, {"something", "anything"}}},
	{"GET", "/test_get_urlparameter2/first/second/third", "/test_get_urlparameter2/first/second/third?name=irisurl&something=anything&else=elsehere", "name=irisurl,something=anything,else=elsehere", 200, true, nil, []param{{"name", "irisurl"}, {"something", "anything"}, {"else", "elsehere"}}},
}

func TestRouter(t *testing.T) {
	api := iris.New()
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

	// create httpexpect instance that will call fasthtpp.RequestHandler directly
	e := httpexpect.WithConfig(httpexpect.Config{
		Reporter: httpexpect.NewAssertReporter(t),
		Client:   fasthttpexpect.NewBinder(api.NoListen().Handler),
	})

	// run the tests (1)
	for idx := range routes {
		r := routes[idx]
		e.Request(r.Method, r.RequestPath).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}

}

func TestPathEscape(t *testing.T) {
	api := iris.New()

	api.Get("/details/:name", func(ctx *iris.Context) {
		name := ctx.Param("name")
		highlight := ctx.URLParam("highlight")
		ctx.Text(iris.StatusOK, fmt.Sprintf("name=%s,highlight=%s", name, highlight))
	})

	e := httpexpect.WithConfig(httpexpect.Config{Reporter: httpexpect.NewAssertReporter(t), Client: fasthttpexpect.NewBinder(api.NoListen().Handler)})

	e.Request("GET", "/details/Sakamoto desu ga?highlight=text").Expect().Status(iris.StatusOK).Body().Equal("name=Sakamoto desu ga,highlight=text")
}
