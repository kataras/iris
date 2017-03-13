package iris_test

import (
	"net/http"
	"testing"

	. "gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/httptest"
)

func TestRouterWrapperPolicySimple(t *testing.T) {
	w1 := RouterWrapperPolicy(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		w.Write([]byte("DATA "))
		next(w, r) // continue to the main router
	})

	w2 := RouterWrapperPolicy(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.RequestURI == "/" {
			next(w, r) // continue to w1
			return
		}
		// else don't execute the router and the handler and fire not found
		w.WriteHeader(StatusNotFound)
	})

	app := New()
	app.Adapt(
		httprouter.New(),
		w1, // order matters, second wraps the first and so on, so the last(w2) is responsible to execute the next wrapper (if more than one) and the router
		w2,
		// w2 -> w1 -> httprouter -> handler
	)

	app.Get("/", func(ctx *Context) {
		ctx.Write([]byte("OK"))
	})

	app.Get("/routerDoesntContinue", func(ctx *Context) {

	})

	e := httptest.New(app, t)
	e.GET("/").Expect().Status(StatusOK).Body().Equal("DATA OK")
	e.GET("/routerDoesntContinue").Expect().Status(StatusNotFound)
}
