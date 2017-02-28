package iris_test

import (
	"strconv"
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
	"gopkg.in/kataras/iris.v6/httptest"
)

func newGorillaMuxAPP() *iris.Framework {
	app := iris.New()
	app.Adapt(gorillamux.New())

	return app
}

func TestGorillaMuxSimple(t *testing.T) {
	app := newGorillaMuxAPP()

	testRoutes := []testRoute{
		// FOUND - registered
		{"GET", "/test_get", "/test_get", "", "hello, get!", 200, true, nil, nil},
		{"POST", "/test_post", "/test_post", "", "hello, post!", 200, true, nil, nil},
		{"PUT", "/test_put", "/test_put", "", "hello, put!", 200, true, nil, nil},
		{"DELETE", "/test_delete", "/test_delete", "", "hello, delete!", 200, true, nil, nil},
		{"HEAD", "/test_head", "/test_head", "", "hello, head!", 200, true, nil, nil},
		{"OPTIONS", "/test_options", "/test_options", "", "hello, options!", 200, true, nil, nil},
		{"CONNECT", "/test_connect", "/test_connect", "", "hello, connect!", 200, true, nil, nil},
		{"PATCH", "/test_patch", "/test_patch", "", "hello, patch!", 200, true, nil, nil},
		{"TRACE", "/test_trace", "/test_trace", "", "hello, trace!", 200, true, nil, nil},
		// NOT FOUND - not registered
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
		{"GET", "/test_get_parameter1/{name}", "/test_get_parameter1/iris", "", "name=iris", 200, true, []param{{"name", "iris"}}, nil},
		{"GET", "/test_get_parameter2/{name}/details/{something}", "/test_get_parameter2/iris/details/anything", "", "name=iris,something=anything", 200, true, []param{{"name", "iris"}, {"something", "anything"}}, nil},
		{"GET", "/test_get_parameter2/{name}/details/{something}/{else:.*}", "/test_get_parameter2/iris/details/anything/elsehere", "", "name=iris,something=anything,else=elsehere", 200, true, []param{{"name", "iris"}, {"something", "anything"}, {"else", "elsehere"}}, nil},
		// URL Parameters
		{"GET", "/test_get_urlparameter1/first", "/test_get_urlparameter1/first", "name=irisurl", "name=irisurl", 200, true, nil, []param{{"name", "irisurl"}}},
		{"GET", "/test_get_urlparameter2/second", "/test_get_urlparameter2/second", "name=irisurl&something=anything", "name=irisurl,something=anything", 200, true, nil, []param{{"name", "irisurl"}, {"something", "anything"}}},
		{"GET", "/test_get_urlparameter2/first/second/third", "/test_get_urlparameter2/first/second/third", "name=irisurl&something=anything&else=elsehere", "name=irisurl,something=anything,else=elsehere", 200, true, nil, []param{{"name", "irisurl"}, {"something", "anything"}, {"else", "elsehere"}}},
	}

	for idx := range testRoutes {
		r := testRoutes[idx]
		if r.Register {
			app.HandleFunc(r.Method, r.Path, func(ctx *iris.Context) {
				ctx.SetStatusCode(r.Status)
				if r.Params != nil && len(r.Params) > 0 {
					ctx.Writef(ctx.ParamsSentence())
				} else if r.URLParams != nil && len(r.URLParams) > 0 {
					if len(r.URLParams) != len(ctx.URLParams()) {
						t.Fatalf("Error when comparing length of url parameters %d != %d", len(r.URLParams), len(ctx.URLParams()))
					}
					paramsKeyVal := ""

					for idxp, p := range r.URLParams {
						val := ctx.URLParam(p.Key)
						paramsKeyVal += p.Key + "=" + val + ","
						if idxp == len(r.URLParams)-1 {
							paramsKeyVal = paramsKeyVal[0 : len(paramsKeyVal)-1]
						}
					}
					ctx.Writef(paramsKeyVal)
				} else {
					ctx.Writef(r.Body)
				}

			})
		}
	}

	e := httptest.New(app, t)

	// run the tests (1)
	for idx := range testRoutes {
		r := testRoutes[idx]
		e.Request(r.Method, r.RequestPath).WithQueryString(r.RequestQuery).
			Expect().
			// compare just the Len because gorillamux gets and sets the vars as map, so the values are unorderded.
			Status(r.Status).Body().Length().Equal(len(r.Body))
	}

}

func TestGorillaMuxSimpleParty(t *testing.T) {
	app := newGorillaMuxAPP()

	h := func(ctx *iris.Context) { ctx.WriteString(ctx.Host() + ctx.Path()) }

	if testEnableSubdomain {
		subdomainParty := app.Party(testSubdomain + ".")
		{
			subdomainParty.Get("/", h)
			subdomainParty.Get("/path1", h)
			subdomainParty.Get("/path2", h)
			subdomainParty.Get("/namedpath/{param1}/something/{param2}", h)
			subdomainParty.Get("/namedpath/{param1}/something/{param2}/else", h)
		}
	}

	// simple
	p := app.Party("/party1")
	{
		p.Get("/", h)
		p.Get("/path1", h)
		p.Get("/path2", h)
		p.Get("/namedpath/{param1}/something/{param2}", h)
		p.Get("/namedpath/{param1}/something/{param2}/else", h)
	}

	app.Config.VHost = "0.0.0.0:" + strconv.Itoa(getRandomNumber(2222, 2399))
	e := httptest.New(app, t)

	request := func(reqPath string) {

		e.Request("GET", reqPath).
			Expect().
			Status(iris.StatusOK).Body().Equal(app.Config.VHost + reqPath)
	}

	// run the tests
	request("/party1/")
	request("/party1/path1")
	request("/party1/path2")
	request("/party1/namedpath/theparam1/something/theparam2")
	request("/party1/namedpath/theparam1/something/theparam2/else")

	if testEnableSubdomain {
		es := subdomainTester(e, app)
		subdomainRequest := func(reqPath string) {
			es.Request("GET", reqPath).
				Expect().
				Status(iris.StatusOK).Body().Equal(testSubdomainHost(app.Config.VHost) + reqPath)
		}

		subdomainRequest("/")
		subdomainRequest("/path1")
		subdomainRequest("/path2")
		subdomainRequest("/namedpath/theparam1/something/theparam2")
		subdomainRequest("/namedpath/theparam1/something/theparam2/else")
	}
}

func TestGorillaMuxPathEscape(t *testing.T) {
	app := newGorillaMuxAPP()

	app.Get("/details/{name}", func(ctx *iris.Context) {
		name := ctx.Param("name")
		highlight := ctx.URLParam("highlight")
		ctx.Writef("name=%s,highlight=%s", name, highlight)
	})

	e := httptest.New(app, t)

	e.GET("/details/Sakamoto desu ga").
		WithQuery("highlight", "text").
		Expect().Status(iris.StatusOK).Body().Equal("name=Sakamoto desu ga,highlight=text")
}

func TestGorillaMuxParamDecodedDecodeURL(t *testing.T) {
	app := newGorillaMuxAPP()

	app.Get("/encoding/{url}", func(ctx *iris.Context) {
		url := iris.DecodeURL(ctx.ParamDecoded("url"))
		ctx.SetStatusCode(iris.StatusOK)
		ctx.WriteString(url)
	})

	e := httptest.New(app, t)

	e.GET("/encoding/http%3A%2F%2Fsome-url.com").Expect().Status(iris.StatusOK).Body().Equal("http://some-url.com")
}

func TestGorillaMuxRouteURLPath(t *testing.T) {
	app := iris.New()
	app.Adapt(gorillamux.New())

	app.None("/profile/{user_id}/{ref}/{anything:.*}", nil).ChangeName("profile")
	app.Boot()

	expected := "/profile/42/iris-go/something"

	if got := app.Path("profile", "user_id", 42, "ref", "iris-go", "anything", "something"); got != expected {
		t.Fatalf("gorillamux' reverse routing 'URLPath' error:  expected %s but got %s", expected, got)
	}
}

func TestGorillaMuxRouteParamAndWildcardPath(t *testing.T) {
	app := iris.New()
	app.Adapt(gorillamux.New())
	routePath := app.RouteWildcardPath("/profile/"+app.RouteParam("user_id")+"/"+app.RouteParam("ref")+"/", "anything")
	expectedRoutePath := "/profile/{user_id}/{ref}/{anything:.*}"
	if routePath != expectedRoutePath {
		t.Fatalf("Gorilla Mux Error on RouteParam and RouteWildcardPath, expecting '%s' but got '%s'", expectedRoutePath, routePath)
	}

	app.Get(routePath, func(ctx *iris.Context) {
		ctx.Writef(ctx.Path())
	})
	e := httptest.New(app, t)

	e.GET("/profile/42/areference/anythinghere").Expect().Status(iris.StatusOK).Body().Equal("/profile/42/areference/anythinghere")
}
