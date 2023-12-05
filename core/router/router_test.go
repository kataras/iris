package router_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/httptest"
)

func TestRouteExists(t *testing.T) {
	// build the api
	app := iris.New()
	emptyHandler := func(*context.Context) {}

	// setup the tested routes
	app.Handle("GET", "/route-exists", emptyHandler)
	app.Handle("POST", "/route-with-param/{param}", emptyHandler)

	// check RouteExists
	app.Handle("GET", "/route-test", func(ctx *context.Context) {
		if ctx.RouteExists("GET", "/route-not-exists") {
			t.Error("Route with path should not exists")
		}

		if ctx.RouteExists("POST", "/route-exists") {
			t.Error("Route with method should not exists")
		}

		if !ctx.RouteExists("GET", "/route-exists") {
			t.Error("Route 1 should exists")
		}

		if !ctx.RouteExists("POST", "/route-with-param/a-param") {
			t.Error("Route 2 should exists")
		}
	})

	// run the tests
	httptest.New(t, app, httptest.Debug(false)).Request("GET", "/route-test").Expect().Status(iris.StatusOK)
}

func TestLowercaseRouting(t *testing.T) {
	app := iris.New()
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		// test bottom to begin wrapper, the last ones should execute first.
		// The ones that are registered at `Build` state, after this `WrapRouter` call.
		// So path should be already lowecased.
		if expected, got := strings.ToLower(r.URL.Path), r.URL.Path; expected != got {
			t.Fatalf("expected path: %s but got: %s", expected, got)
		}
		next(w, r)
	})

	h := func(ctx iris.Context) { ctx.WriteString(ctx.Path()) }

	// Register routes.
	tests := []string{"/", "/lowercase", "/UPPERCASE", "/Title", "/m1xEd2"}
	for _, tt := range tests {
		app.Get(tt, h)
	}

	app.Configure(iris.WithLowercaseRouting)
	// Test routes.
	e := httptest.New(t, app)
	for _, tt := range tests {
		s := strings.ToLower(tt)
		e.GET(tt).Expect().Status(httptest.StatusOK).Body().IsEqual(s)
		e.GET(s).Expect().Status(httptest.StatusOK).Body().IsEqual(s)
		e.GET(strings.ToUpper(tt)).Expect().Status(httptest.StatusOK).Body().IsEqual(s)
	}
}

func TestRouterWrapperOrder(t *testing.T) {
	// last is wrapping the previous.

	// first is executed last.
	userWrappers := []router.WrapperFunc{
		func(w http.ResponseWriter, r *http.Request, main http.HandlerFunc) {
			io.WriteString(w, "6")
			main(w, r)
		},
		func(w http.ResponseWriter, r *http.Request, main http.HandlerFunc) {
			io.WriteString(w, "5")
			main(w, r)
		},
	}
	// should be executed before userWrappers.
	redirectionWrappers := []router.WrapperFunc{
		func(w http.ResponseWriter, r *http.Request, main http.HandlerFunc) {
			io.WriteString(w, "3")
			main(w, r)
		},
		func(w http.ResponseWriter, r *http.Request, main http.HandlerFunc) {
			io.WriteString(w, "4")
			main(w, r)
		},
	}
	// should be executed before redirectionWrappers.
	afterRedirectionWrappers := []router.WrapperFunc{
		func(w http.ResponseWriter, r *http.Request, main http.HandlerFunc) {
			io.WriteString(w, "2")
			main(w, r)
		},
		func(w http.ResponseWriter, r *http.Request, main http.HandlerFunc) {
			io.WriteString(w, "1")
			main(w, r)
		},
	}

	testOrder1 := iris.New()
	for _, w := range userWrappers {
		testOrder1.WrapRouter(w)
		// this always wraps the previous one, but it's not accessible after Build state,
		// the below are simulating the SubdomainRedirect and ForceLowercaseRouting.
	}
	for _, w := range redirectionWrappers {
		testOrder1.AddRouterWrapper(w)
	}
	for _, w := range afterRedirectionWrappers {
		testOrder1.PrependRouterWrapper(w)
	}

	testOrder2 := iris.New()
	for _, w := range redirectionWrappers {
		testOrder2.AddRouterWrapper(w)
	}
	for _, w := range userWrappers {
		testOrder2.WrapRouter(w)
	}
	for _, w := range afterRedirectionWrappers {
		testOrder2.PrependRouterWrapper(w)
	}

	testOrder3 := iris.New()
	for _, w := range redirectionWrappers {
		testOrder3.AddRouterWrapper(w)
	}
	for _, w := range afterRedirectionWrappers {
		testOrder3.PrependRouterWrapper(w)
	}
	for _, w := range userWrappers {
		testOrder3.WrapRouter(w)
	}

	appTests := []*iris.Application{
		testOrder1, testOrder2, testOrder3,
	}

	expectedOrderStr := "123456"
	for _, app := range appTests {
		app.Get("/", func(ctx iris.Context) {}) // to not append the not found one.

		e := httptest.New(t, app)
		e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual(expectedOrderStr)
	}
}

func TestNewSubdomainPartyRedirectHandler(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("root index")
	})

	test := app.Subdomain("test")
	test.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.WriteString("test 404")
	})
	test.Get("/", func(ctx iris.Context) {
		ctx.WriteString("test index")
	})

	testold := app.Subdomain("testold")
	// redirects testold.mydomain.com to test.mydomain.com .
	testold.UseRouter(router.NewSubdomainPartyRedirectHandler(test))
	testold.Get("/", func(ctx iris.Context) {
		ctx.WriteString("test old index (should never be fired)")
	})
	testoldLeveled := testold.Subdomain("leveled")
	testoldLeveled.Get("/", func(ctx iris.Context) {
		ctx.WriteString("leveled.testold this can be fired")
	})

	if redirectHandler := router.NewSubdomainPartyRedirectHandler(app.WildcardSubdomain()); redirectHandler != nil {
		t.Fatal("redirect handler should be nil, we cannot redirect to a wildcard")
	}

	e := httptest.New(t, app)
	e.GET("/").WithURL("http://mydomain.com").Expect().Status(iris.StatusOK).Body().IsEqual("root index")
	e.GET("/").WithURL("http://test.mydomain.com").Expect().Status(iris.StatusOK).Body().IsEqual("test index")
	e.GET("/").WithURL("http://testold.mydomain.com").Expect().Status(iris.StatusOK).Body().IsEqual("test index")
	e.GET("/").WithURL("http://testold.mydomain.com/notfound").Expect().Status(iris.StatusNotFound).Body().IsEqual("test 404")
	e.GET("/").WithURL("http://leveled.testold.mydomain.com").Expect().Status(iris.StatusOK).Body().IsEqual("leveled.testold this can be fired")
}

func TestHandleServer(t *testing.T) {
	otherApp := iris.New()
	otherApp.Get("/test/me/{first:string}", func(ctx iris.Context) {
		ctx.HTML("<h1>Other App: %s</h1>", ctx.Params().Get("first"))
	})
	otherApp.Build()

	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>Main App</h1>")
	})

	app.HandleServer("/api/identity/{first:string}/orgs/{second:string}/{p:path}", otherApp)

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual("<h1>Main App</h1>")
	e.GET("/api/identity/first/orgs/second/test/me/kataras").Expect().Status(iris.StatusOK).Body().IsEqual("<h1>Other App: kataras</h1>")
	e.GET("/api/identity/first/orgs/second/test/me").Expect().Status(iris.StatusNotFound)
}
