package router_test

import (
	"fmt"
	"net/http"
	stdhttptest "net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func newDynamicApp() *iris.Application {
	return iris.New(iris.Options{
		Router: iris.RouterOptions{
			EnableDynamicHandler: true,
		},
	})
}

// A route added after the first build is served once BuildRouter runs again,
// also when a pre-routing (UseRouter) handler stands in front of the mux.
func TestDynamicHandlerRebuildWithUseRouter(t *testing.T) {
	app := newDynamicApp()

	var pre atomic.Int64
	app.UseRouter(func(ctx iris.Context) {
		pre.Add(1)
		ctx.Next()
	})
	app.Get("/a", func(ctx iris.Context) { ctx.WriteString("a") })

	e := httptest.New(t, app)
	e.GET("/a").Expect().Status(httptest.StatusOK).Body().IsEqual("a")

	b := app.Get("/b", func(ctx iris.Context) { ctx.WriteString("b") })
	if err := app.BuildRouter(); err != nil {
		t.Fatal(err)
	}

	pre.Store(0)
	e.GET("/b").Expect().Status(httptest.StatusOK).Body().IsEqual("b")
	if n := pre.Load(); n != 1 {
		t.Fatalf("the pre-routing handler ran %d times for one request", n)
	}

	// And a removed one is gone.
	if !app.RemoveRoute(b.GetName()) {
		t.Fatalf("route %q was not removed", b.GetName())
	}
	if err := app.BuildRouter(); err != nil {
		t.Fatal(err)
	}
	e.GET("/b").Expect().Status(httptest.StatusNotFound)
	e.GET("/a").Expect().Status(httptest.StatusOK).Body().IsEqual("a")
}

// A router wrapper still runs once per request after a rebuild.
func TestDynamicHandlerRebuildWithRouterWrapper(t *testing.T) {
	app := newDynamicApp()

	var wrapped atomic.Int64
	app.AddRouterWrapper(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		wrapped.Add(1)
		next(w, r)
	})
	app.Get("/a", func(ctx iris.Context) { ctx.WriteString("a") })

	e := httptest.New(t, app)
	e.GET("/a").Expect().Status(httptest.StatusOK).Body().IsEqual("a")

	app.Get("/b", func(ctx iris.Context) { ctx.WriteString("b") })
	if err := app.BuildRouter(); err != nil {
		t.Fatal(err)
	}

	wrapped.Store(0)
	e.GET("/b").Expect().Status(httptest.StatusOK).Body().IsEqual("b")
	if n := wrapped.Load(); n != 1 {
		t.Fatalf("the router wrapper ran %d times for one request", n)
	}
}

// Requests are served while routes are added and the router is rebuilt.
// Run it with -race: that is what reports an unsafe rebuild.
func TestDynamicHandlerRebuildWhileServing(t *testing.T) {
	app := newDynamicApp()
	app.UseRouter(func(ctx iris.Context) { ctx.Next() })
	app.Use(func(ctx iris.Context) { ctx.Next() })
	app.Get("/a", func(ctx iris.Context) { ctx.WriteString("a") })
	// The fallback parameter type is registered first.
	app.Get("/users/{name:string}", func(ctx iris.Context) { ctx.Writef("string %s", ctx.Params().GetString("name")) })
	app.Get("/users/{id:int}", func(ctx iris.Context) { ctx.Writef("int %d", ctx.Params().GetInt("id")) })

	httptest.New(t, app) // builds the application and silences its logger.

	get := func(path string) (int, string) {
		w := stdhttptest.NewRecorder()
		app.ServeHTTP(w, stdhttptest.NewRequest(http.MethodGet, path, nil))
		return w.Code, w.Body.String()
	}

	want := map[string]string{"/a": "a", "/users/42": "int 42", "/users/bob": "string bob", "/nope": "Not Found"}

	var (
		wg   sync.WaitGroup
		stop atomic.Bool
		bad  atomic.Int64
	)
	for range 4 {
		wg.Go(func() {
			for !stop.Load() {
				for path, body := range want {
					if _, got := get(path); got != body {
						bad.Add(1)
					}
				}
			}
		})
	}

	const n = 20
	for i := range n {
		path := fmt.Sprintf("/r%d", i)
		app.Get(path, func(ctx iris.Context) { ctx.WriteString(path) })
		if err := app.BuildRouter(); err != nil {
			t.Fatal(err)
		}
	}
	stop.Store(true)
	wg.Wait()

	if n := bad.Load(); n > 0 {
		t.Fatalf("%d requests got a wrong answer during the rebuilds", n)
	}
	for i := range n {
		path := fmt.Sprintf("/r%d", i)
		if code, body := get(path); code != http.StatusOK || body != path {
			t.Fatalf("%s: got %d %q", path, code, body)
		}
	}
}

// A handler may add a route and rebuild the router during its own request:
// the rebuild must not wait for the request that asked for it.
func TestDynamicHandlerRebuildInsideHandler(t *testing.T) {
	app := newDynamicApp()
	app.Get("/add", func(ctx iris.Context) {
		app.Get("/b", func(ctx iris.Context) { ctx.WriteString("b") })
		if err := app.BuildRouter(); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		ctx.WriteString("added")
	})

	httptest.New(t, app)

	done := make(chan *stdhttptest.ResponseRecorder, 1)
	go func() {
		w := stdhttptest.NewRecorder()
		app.ServeHTTP(w, stdhttptest.NewRequest(http.MethodGet, "/add", nil))
		done <- w
	}()

	select {
	case w := <-done:
		if w.Code != http.StatusOK || w.Body.String() != "added" {
			t.Fatalf("/add: got %d %q", w.Code, w.Body.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the request that rebuilds the router did not return")
	}

	w := stdhttptest.NewRecorder()
	app.ServeHTTP(w, stdhttptest.NewRequest(http.MethodGet, "/b", nil))
	if w.Code != http.StatusOK || w.Body.String() != "b" {
		t.Fatalf("/b: got %d %q", w.Code, w.Body.String())
	}
}
