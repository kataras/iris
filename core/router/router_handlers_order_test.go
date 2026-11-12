// black-box testing
//
// see _examples/14-testing/iris-httptest/main_test.go for the most common router tests that you may want to see,
// this is a test which makes sure that the RouterBuilder's `UseRouter`, `Use` and `Done` functions are
// working as expected.

package router_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// test registering of below handlers
// with a different order but the route's final
// response should be the same at all cases.
var (
	writeHandler = func(s string) iris.Handler {
		return func(ctx iris.Context) {
			ctx.WriteString(s)
			ctx.Next()
		}
	}

	mainResponse = "main"
	mainHandler  = writeHandler(mainResponse)

	firstUseResponse = "use1"
	firstUseHandler  = writeHandler(firstUseResponse)

	secondUseResponse = "use2"
	secondUseHandler  = writeHandler(secondUseResponse)

	firstUseRouterResponse = "userouter1"
	// Use inline handler, no the `writeHandler`,
	// because it will be overridden by `secondUseRouterHandler` otherwise,
	// look `UseRouter:context.UpsertHandlers` for more.
	firstUseRouterHandler = func(ctx iris.Context) {
		ctx.WriteString(firstUseRouterResponse)
		ctx.Next()
	}

	secondUseRouterResponse = "userouter2"
	secondUseRouterHandler  = writeHandler(secondUseRouterResponse)

	firstUseGlobalResponse = "useglobal1"
	firstUseGlobalHandler  = writeHandler(firstUseGlobalResponse)

	secondUseGlobalResponse = "useglobal2"
	secondUseGlobalHandler  = writeHandler(secondUseGlobalResponse)

	firstDoneResponse = "done1"
	firstDoneHandler  = func(ctx iris.Context) {
		ctx.Next()

		ctx.WriteString(firstDoneResponse)
	}

	secondDoneResponse = "done2"
	secondDoneHandler  = func(ctx iris.Context) {
		ctx.Next()

		ctx.WriteString(secondDoneResponse)
	}

	finalResponse = firstUseRouterResponse + secondUseRouterResponse + firstUseGlobalResponse + secondUseGlobalResponse +
		firstUseResponse + secondUseResponse + mainResponse + secondDoneResponse + firstDoneResponse /*done 1st is executed first with ctx.Next, so the done 2nd will be executed first */

	testResponse = func(t *testing.T, app *iris.Application, path string) {
		t.Helper()

		e := httptest.New(t, app)
		e.GET(path).Expect().Status(httptest.StatusOK).Body().IsEqual(finalResponse)
	}
)

func TestMiddlewareByRouteDef(t *testing.T) {
	app := iris.New()
	app.UseRouter(firstUseRouterHandler)
	app.UseRouter(secondUseRouterHandler)

	app.Get("/mypath", firstDoneHandler, secondDoneHandler, firstUseGlobalHandler, secondUseGlobalHandler, firstUseHandler, secondUseHandler,
		mainHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByUseAndDoneDef(t *testing.T) {
	app := iris.New()
	app.UseRouter(firstUseRouterHandler, secondUseRouterHandler)

	doneHandler := func(ctx iris.Context) {
		ctx.Next()

		firstDoneHandler(ctx)
	}

	doneHandler2 := func(ctx iris.Context) {
		ctx.Next()

		secondDoneHandler(ctx)
	}

	app.Use(doneHandler, doneHandler2, firstUseGlobalHandler, secondUseGlobalHandler, firstUseHandler, secondUseHandler)

	app.Get("/mypath", mainHandler)

	testResponse(t, app, "/mypath")
}

func TestUseRouterStopExecution(t *testing.T) {
	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.WriteString("stop")
		// no ctx.Next, so the router has not even the chance to work.
	})
	app.Get("/", writeHandler("index"))

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual("stop")

	app = iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.StopWithError(iris.StatusForbidden, fmt.Errorf("custom error"))
	})
	app.Get("/", writeHandler("index"))

	e = httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusForbidden).Body().IsEqual("custom error")
}

func TestUseRouterSubdomains(t *testing.T) {
	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		if ctx.Subdomain() == "old" {
			ctx.Next() // call the router, do not write.
			return
		}

		// if we write here, it will always give 200 OK,
		// even on not registered routes, that's the point at the end,
		// full control here when we need it.
		ctx.WriteString("always_")
		ctx.Next()
	})

	adminAPI := app.Subdomain("admin")
	adminAPI.UseRouter(func(ctx iris.Context) {
		ctx.WriteString("admin always_")
		ctx.Next()
	})
	adminAPI.Get("/", func(ctx iris.Context) {
		ctx.WriteString("admin")
	})

	adminControlAPI := adminAPI.Subdomain("control")
	adminControlAPI.UseRouter(func(ctx iris.Context) {
		ctx.WriteString("control admin always_")
		ctx.Next()
	})
	adminControlAPI.Get("/", func(ctx iris.Context) {
		ctx.WriteString("control admin")
	})

	oldAPI := app.Subdomain("old")
	oldAPI.Get("/", func(ctx iris.Context) {
		ctx.WriteString("chat")
	})

	e := httptest.New(t, app, httptest.URL("http://example.com"))
	e.GET("/notfound").Expect().Status(iris.StatusOK).Body().IsEqual("always_")

	e.GET("/").WithURL("http://admin.example.com").Expect().Status(iris.StatusOK).Body().
		IsEqual("always_admin always_admin")

	e.GET("/").WithURL("http://control.admin.example.com").Expect().Status(iris.StatusOK).Body().
		IsEqual("always_admin always_control admin always_control admin")

	// It has a route, and use router just proceeds to the router.
	e.GET("/").WithURL("http://old.example.com").Expect().Status(iris.StatusOK).Body().
		IsEqual("chat")
	// this is not a registered path, should fire 404, the UseRouter does not write
	// anything to the response writer, so the router has control over it.
	e.GET("/notfound").WithURL("http://old.example.com").Expect().Status(iris.StatusNotFound).Body().
		IsEqual("Not Found")
}

func TestUseRouterPathPrefix(t *testing.T) {
	app := iris.New()
	app.UseRouter(func(ctx iris.Context) { // this registers everywhere.
		ctx.WriteString("root_")
		ctx.Next()
	})
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("index")
	})
	app.Get("/about", func(ctx iris.Context) {
		ctx.WriteString("about")
	})

	usersAPI := app.Router("/users")
	usersAPI.UseRouter(func(ctx iris.Context) { // this only registers under the /users prefix.
		ctx.WriteString("users_")
		ctx.Next()
	})
	usersAPI.Get("/", func(ctx iris.Context) {
		ctx.WriteString("index")
	})
	usersAPI.Get("/profile", func(ctx iris.Context) {
		ctx.WriteString("profile")
	})

	postsAPI := app.Router("/posts")
	postsAPI.UseRouter(func(ctx iris.Context) { // this only registers under the /posts prefix.
		ctx.WriteString("posts_")
		ctx.Next()
	})
	postsAPI.Get("/", func(ctx iris.Context) {
		ctx.WriteString("index")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual("root_index")
	e.GET("/about").Expect().Status(iris.StatusOK).Body().IsEqual("root_about")
	e.GET("/users").Expect().Status(iris.StatusOK).Body().IsEqual("root_users_index")
	e.GET("/users/profile").Expect().Status(iris.StatusOK).Body().IsEqual("root_users_profile")
	e.GET("/posts").Expect().Status(iris.StatusOK).Body().IsEqual("root_posts_index")
}

func TestUseWrapOrder(t *testing.T) {
	var (
		expectedBody         = "#1 .WrapRouter\n#2 .UseRouter\n#3 .Use\n#4 Main Handler\n"
		expectedNotFoundBody = "#1 .UseError\n#2 Main Error Handler\n"
		makeMiddleware       = func(body string) iris.Handler {
			return func(ctx iris.Context) {
				if route := ctx.CurrentRoute(); route.IsNoRoute() || route.IsMethodNotAllowed() {
					ctx.WriteString("#1 .UseError\n")
				} else {
					ctx.WriteString(body)
				}
				ctx.Next()
			}
		}

		handler = func(ctx iris.Context) {
			ctx.WriteString("#4 Main Handler\n")
		}

		errorHandler = func(ctx iris.Context) {
			ctx.WriteString("#2 Main Error Handler\n")
		}

		useHandler = makeMiddleware("#3 .Use\n")
		useRouter  = func(ctx iris.Context) {
			if ctx.Path() == "/" {
				ctx.WriteString("#2 .UseRouter\n")
			}

			ctx.Next()
		}
		wrapRouter = func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
			if r.URL.Path == "/" {
				w.Write([]byte("#1 .WrapRouter\n"))
				/* Note for new Gophers:

				If we write something here, on a not found resource,
				in the raw `net/http` wrapper like this one, then the
				response writer will send `200` status OK (on first write).
				Any error handler will not be fired as expected.
				Also, when `w.WriteHeader` is called you can NOT
				change the status code later on.

				In Iris Handlers, if you write before status code set,
				then it sends 200 status OK and it cannot change as well.
				However if we just called `ctx.StatusCode` inside an
				Iris Handler without any content written then we
				would able to change the status code later on.
				When you need to change that behavior you should
				start the handler with a ctx.Record() call.
				*/
			}

			// Continue by executing the Iris Router and leave it do its job.
			router(w, r)
		}
	)

	app := iris.New()
	app.Use(useHandler)
	app.UseRouter(useRouter)
	app.WrapRouter(wrapRouter)

	app.HandleNoRoute(errorHandler)
	app.HandleNoMethod(errorHandler)
	app.Get("/", handler)

	e := httptest.New(t, app)
	e.GET("/NotFound").Expect().Status(iris.StatusNotFound).Body().IsEqual(expectedNotFoundBody)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual(expectedBody)
}

func TestResumeExecution(t *testing.T) {
	before := func(ctx iris.Context) {
		ctx.WriteString("1")

		curIdx := ctx.HandlerIndex(-1)

		ctx.StopExecution()
		ctx.Next()
		ctx.StopExecution()
		ctx.Next()
		ctx.ResumeExecution()

		if ctx.HandlerIndex(-1) != curIdx {
			ctx.WriteString("| 1. NOT OK")
		}

		ctx.StopExecution()
		ctx.ResumeExecution()

		if ctx.HandlerIndex(-1) != curIdx {
			ctx.WriteString("| 2. NOT OK")
		}

		ctx.Next()

		if ctx.HandlerIndex(-1) != curIdx+2 /* 2 and 3 */ {
			ctx.WriteString("| 3. NOT OK")
		}
	}

	handler := func(ctx iris.Context) {
		ctx.WriteString("2")
		ctx.Next()
	}

	after := func(ctx iris.Context) {
		ctx.WriteString("3")

		if !ctx.Proceed(func(ctx iris.Context) {
			ctx.Next()
		}) {
			ctx.WriteString(" | 4. NOT OK")
		}
	}

	expectedBody := "123"

	app := iris.New()
	app.Get("/", before, handler, after)

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual(expectedBody)
}
