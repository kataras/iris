// black-box testing
//
// see _examples/routing/main_test.go for the most common router tests that you may want to see,
// this is a test which makes sure that the APIBuilder's `UseGlobal`, `Use` and `Done` functions are
// working as expected.

package router_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
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
	firstDoneHandler  = writeHandler(firstDoneResponse)

	secondDoneResponse = "done2"
	secondDoneHandler  = func(ctx iris.Context) {
		ctx.WriteString(secondDoneResponse)
	}

	finalResponse = firstUseRouterResponse + secondUseRouterResponse + firstUseGlobalResponse + secondUseGlobalResponse +
		firstUseResponse + secondUseResponse + mainResponse + firstDoneResponse + secondDoneResponse

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

	app.Get("/mypath", firstUseGlobalHandler, secondUseGlobalHandler, firstUseHandler, secondUseHandler,
		mainHandler, firstDoneHandler, secondDoneHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByUseAndDoneDef(t *testing.T) {
	app := iris.New()
	app.UseRouter(firstUseRouterHandler, secondUseRouterHandler)
	app.Use(firstUseGlobalHandler, secondUseGlobalHandler, firstUseHandler, secondUseHandler)
	app.Done(firstDoneHandler, secondDoneHandler)

	app.Get("/mypath", mainHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByUseUseGlobalAndDoneDef(t *testing.T) {
	app := iris.New()

	app.Use(firstUseHandler, secondUseHandler)
	// if failed then UseGlobal didnt' registered these handlers even before the
	// existing middleware.
	app.UseGlobal(firstUseGlobalHandler, secondUseGlobalHandler)
	app.Done(firstDoneHandler, secondDoneHandler)

	app.UseRouter(firstUseRouterHandler, secondUseRouterHandler)
	app.Get("/mypath", mainHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByUseDoneAndUseGlobalDef(t *testing.T) {
	app := iris.New()
	app.UseRouter(firstUseRouterHandler, secondUseRouterHandler)

	app.Use(firstUseHandler, secondUseHandler)
	app.Done(firstDoneHandler, secondDoneHandler)

	app.Get("/mypath", mainHandler)

	// if failed then UseGlobal was unable to
	// prepend these handlers to the route was registered before
	// OR
	// when order failed because these should be executed in order, first the firstUseGlobalHandler,
	// because they are the same type (global begin handlers)
	app.UseGlobal(firstUseGlobalHandler)
	app.UseGlobal(secondUseGlobalHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByUseGlobalUseAndDoneGlobalDef(t *testing.T) {
	app := iris.New()
	app.UseRouter(firstUseRouterHandler)
	app.UseRouter(secondUseRouterHandler)

	app.UseGlobal(firstUseGlobalHandler)
	app.UseGlobal(secondUseGlobalHandler)
	app.Use(firstUseHandler, secondUseHandler)

	app.Get("/mypath", mainHandler)

	app.DoneGlobal(firstDoneHandler, secondDoneHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByDoneUseAndUseGlobalDef(t *testing.T) {
	app := iris.New()
	app.UseRouter(firstUseRouterHandler, secondUseRouterHandler)
	app.Done(firstDoneHandler, secondDoneHandler)

	app.Use(firstUseHandler, secondUseHandler)

	app.Get("/mypath", mainHandler)

	app.UseGlobal(firstUseGlobalHandler)
	app.UseGlobal(secondUseGlobalHandler)

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
	app.OnErrorCode(iris.StatusForbidden, func(ctx iris.Context) {
		ctx.Writef("err: %v", ctx.GetErr())
	})
	app.UseRouter(func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusForbidden, fmt.Errorf("custom error"))
		// stopped but not data written yet, the error code handler
		// should be responsible of it (use StopWithError to write and close).
	})
	app.Get("/", writeHandler("index"))

	e = httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusForbidden).Body().IsEqual("err: custom error")
}

func TestUseRouterParentDisallow(t *testing.T) {
	const expectedResponse = "no_userouter_allowed"

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.WriteString("always")
		ctx.Next()
	})
	app.Get("/index", func(ctx iris.Context) {
		ctx.WriteString(expectedResponse)
	})

	app.SetPartyMatcher(func(ctx iris.Context, p iris.Party) bool {
		// modifies the PartyMatcher to not match any UseRouter,
		// tests should receive the handlers response alone.
		return false
	})

	app.PartyFunc("/", func(p iris.Party) { // it's the same instance of app.
		p.UseRouter(func(ctx iris.Context) {
			ctx.WriteString("_2")
			ctx.Next()
		})
		p.Get("/", func(ctx iris.Context) {
			ctx.WriteString(expectedResponse)
		})
	})

	app.PartyFunc("/user", func(p iris.Party) {
		p.UseRouter(func(ctx iris.Context) {
			ctx.WriteString("_3")
			ctx.Next()
		})

		p.Get("/", func(ctx iris.Context) {
			ctx.WriteString(expectedResponse)
		})
	})

	e := httptest.New(t, app)
	e.GET("/index").Expect().Status(iris.StatusOK).Body().IsEqual(expectedResponse)
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual(expectedResponse)
	e.GET("/user").Expect().Status(iris.StatusOK).Body().IsEqual(expectedResponse)
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

func TestUseWrapOrder(t *testing.T) {
	var (
		expectedBody         = "#1 .WrapRouter\n#2 .UseRouter\n#3 .UseGlobal\n#4 .Use\n#5 Main Handler\n"
		expectedNotFoundBody = "#3 .UseGlobal\n#1 .UseError\n#2 Main Error Handler\n"
		makeMiddleware       = func(body string) iris.Handler {
			return func(ctx iris.Context) {
				ctx.WriteString(body)
				ctx.Next()
			}
		}

		handler = func(ctx iris.Context) {
			ctx.WriteString("#5 Main Handler\n")
		}

		errorHandler = func(ctx iris.Context) {
			ctx.WriteString("#2 Main Error Handler\n")
		}

		useHandler = makeMiddleware("#4 .Use\n")
		useGlobal  = makeMiddleware("#3 .UseGlobal\n")
		useError   = func(ctx iris.Context) {
			// UseError has captured the status code, because it runs
			// after the router itself but only one error handlers.
			ctx.WriteString("#1 .UseError\n")
			ctx.Next()
		}
		useRouter = func(ctx iris.Context) {
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
	app.UseGlobal(useGlobal)
	app.UseError(useError)
	app.UseRouter(useRouter)
	app.WrapRouter(wrapRouter)

	app.OnErrorCode(iris.StatusNotFound, errorHandler)
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
