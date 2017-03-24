package iris_test

import (
	"net/http"
	"testing"

	. "gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/httptest"
)

// This will be removed at the final release
// it's here just for my tests
// it will may transferred to the (advanced) examples repository
// in order to show the users how they can adapt any third-party router.
// They can also view the ./adaptors/httprouter and ./adaptors/gorillamux.
func newTestNativeRouter() Policies {
	fireMethodNotAllowed := false
	return Policies{
		EventPolicy: EventPolicy{
			Boot: func(s *Framework) {
				fireMethodNotAllowed = s.Config.FireMethodNotAllowed
			},
		},
		RouterReversionPolicy: RouterReversionPolicy{
			// path normalization done on iris' side
			StaticPath:   func(path string) string { return path },
			WildcardPath: func(requestPath string, paramName string) string { return requestPath },
			URLPath: func(r RouteInfo, args ...string) string {
				if r == nil {
					return ""
				}
				// note:
				// as we already know, net/http servemux doesn't provides parameterized paths so we will
				// use the passed args(if any) to build the url query
				path := r.Path()
				if len(args) > 0 {
					if len(args)%2 != 0 {
						// key=value
						// so the result of len args should be %2==0.
						// if not return just the path.
						return path
					}
					path += "?"
					for i := 0; i < len(args)-1; i++ {
						path += args[i] + "=" + args[i+1]
						i++
						if i != len(args)-1 {
							path += "&"
						}

					}
				}
				return path
			},
		},
		RouterBuilderPolicy: func(repo RouteRepository, context ContextPool) http.Handler {
			servemux := http.NewServeMux()
			noIndexRegistered := true

			repo.Visit(func(route RouteInfo) {
				path := route.Path()
				if path == "/" {
					noIndexRegistered = false // this goes before the handlefunc("/")
				}
				if path[len(path)-1] != '/' { // append a slash (net/http works this way)
					path += "/"
					repo.ChangePath(route, path)
				}
				servemux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					ctx := context.Acquire(w, r)

					if ctx.Method() == route.Method() || ctx.Method() == MethodOptions && route.HasCors() {
						ctx.Middleware = route.Middleware()
						recorder := ctx.Recorder()
						ctx.Do()

						statusCode := recorder.StatusCode()
						if statusCode >= 400 { // if we have an error status code try to find a custom error handler
							errorHandler := ctx.Framework().Errors.Get(statusCode)
							if errorHandler != nil {
								// it will reset the response and write its own response by the user's handler
								errorHandler.Serve(ctx)
							}
						}
					} else if fireMethodNotAllowed {
						ctx.EmitError(StatusMethodNotAllowed) // fire method not allowed if enabled
					} else { // else fire not found
						ctx.EmitError(StatusNotFound)
					}

					context.Release(ctx)
				})
			})

			// ok, we can't bypass the net/http server.go's err handlers
			// we have two options:
			// - create the mux by ourselve, not an ideal because we already done two of them.
			// - create a new response writer which will check once if user has registered error handler,if yes write that response instead.
			// - on "/" path(which net/http fallbacks if no any registered route handler found) make if requested_path != "/" or ""
			// and emit the 404 error, but for the rest of the custom errors...?
			// - use our custom context's recorder to record the status code, this will be a bit slower solution(maybe not)
			//   but it covers all our scenarios.
			if noIndexRegistered {
				servemux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					context.Run(w, r, func(ctx *Context) {
						ctx.EmitError(StatusNotFound)
					})
				})
			}
			return servemux
		},
	}
}

func h(ctx *Context) {
	ctx.WriteString("hello from " + ctx.Path())
}

type testNativeRoute struct {
	method, path, body string
	status             int
}

func TestNativeRouter(t *testing.T) {
	expectedWrongMethodStatus := StatusNotFound
	app := New(Configuration{FireMethodNotAllowed: true})
	app.Adapt(newTestNativeRouter())
	// 404 or 405
	if app.Config.FireMethodNotAllowed {
		expectedWrongMethodStatus = StatusMethodNotAllowed
	}

	var testRoutes = []testNativeRoute{
		{"GET", "/hi", "Should be /hi/ \nhello from /hi/", StatusOK},
		{"GET", "/other", "Should be /other/ \nhello from /other/", StatusOK},
		{"GET", "/hi/you", "Should be /hi/you/ \nhello from /hi/you/", StatusOK},
		{"POST", "/hey", "hello from /hey/", StatusOK},
		{"GET", "/hey", "Method Not Allowed", expectedWrongMethodStatus},
		{"GET", "/doesntexists", "<b>Custom 404 page</b>", StatusNotFound},
	}
	app.OnError(404, func(ctx *Context) {
		ctx.HTML(404, "<b>Custom 404 page</b>")
	})

	app.Get("/hi", func(ctx *Context) {
		ctx.Writef("Should be /hi/ \n")
		ctx.Next()
	}, h)

	app.Get("/other", func(ctx *Context) {
		ctx.Writef("Should be /other/ \n")
		ctx.Next()
	}, h)

	app.Get("/hi/you", func(ctx *Context) {
		ctx.Writef("Should be /hi/you/ \n")
		ctx.Next()
	}, h)

	app.Post("/hey", h)

	app.None("/profile", func(ctx *Context) {
		userid, _ := ctx.URLParamInt("user_id")
		ref := ctx.URLParam("ref")
		ctx.Writef("%s\n%d", userid, ref)
	}).ChangeName("profile")

	e := httptest.New(app, t)
	expected := "/profile/?user_id=42&ref=iris-go&anything=something"

	if got := app.Path("profile", "user_id", 42, "ref", "iris-go", "anything", "something"); got != expected {
		t.Fatalf("URLPath expected %s but got %s", expected, got)
	}

	for _, r := range testRoutes {
		// post should be passed with ending / here, it's not iris-specific.
		if r.method == "POST" {
			r.path += "/"
		}
		e.Request(r.method, r.path).Expect().Status(r.status).Body().Equal(r.body).Raw()
	}

}
