// black-box testing
package context_test

import (
	stdContext "context"
	"net/http"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
)

func TestFromStd(t *testing.T) {
	expected := "ok"
	std := func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(expected))
		if err != nil {
			t.Fatal(err)
		}
	}

	h := context.FromStd(http.HandlerFunc(std))

	hFunc := context.FromStd(std)

	app := iris.New()
	app.Get("/handler", h)
	app.Get("/func", hFunc)

	e := httptest.New(t, app)

	e.GET("/handler").
		Expect().Status(iris.StatusOK).Body().IsEqual(expected)

	e.GET("/func").
		Expect().Status(iris.StatusOK).Body().IsEqual(expected)
}

// A standard handler mounted on an Iris route reads the route parameters with
// r.PathValue and the matched template with r.Pattern, in all three forms.
func TestFromStdSetsPathValues(t *testing.T) {
	write := func(w http.ResponseWriter, r *http.Request, key string) {
		_, _ = w.Write([]byte(r.PathValue(key) + "|" + r.Pattern))
	}

	forms := map[string]func(key string) context.Handler{
		"handler": func(key string) context.Handler {
			return context.FromStd(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { write(w, r, key) }))
		},
		"func": func(key string) context.Handler {
			return context.FromStd(func(w http.ResponseWriter, r *http.Request) { write(w, r, key) })
		},
		"next": func(key string) context.Handler {
			return context.FromStd(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) { write(w, r, key) })
		},
	}

	app := iris.New()
	for name, form := range forms {
		app.Get("/"+name+"/users/{id}", form("id"))
		app.Get("/"+name+"/items/{n:int}", form("n"))
	}

	e := httptest.New(t, app)

	for name := range forms {
		e.GET("/" + name + "/users/42").Expect().Status(iris.StatusOK).
			Body().IsEqual("42|/" + name + "/users/{id}")
		e.GET("/" + name + "/items/7").Expect().Status(iris.StatusOK).
			Body().IsEqual("7|/" + name + "/items/{n:int}")
	}
}

func TestFromStdWithNext(t *testing.T) {
	basicauth := "secret"
	passed := "ok"

	type contextKey string
	stdWNext := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if username, password, ok := r.BasicAuth(); ok &&
			username == basicauth && password == basicauth {
			ctx := stdContext.WithValue(r.Context(), contextKey("key"), "ok")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		w.WriteHeader(iris.StatusForbidden)
	}

	h := context.FromStdWithNext(stdWNext)
	next := func(ctx *context.Context) {
		ctx.WriteString(ctx.Request().Context().Value(contextKey("key")).(string))
	}

	app := iris.New()
	app.Get("/handlerwithnext", h, next)

	e := httptest.New(t, app)

	e.GET("/handlerwithnext").
		Expect().Status(iris.StatusForbidden)

	e.GET("/handlerwithnext").WithBasicAuth(basicauth, basicauth).
		Expect().Status(iris.StatusOK).Body().IsEqual(passed)
}
