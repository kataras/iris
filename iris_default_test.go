package iris_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// Default advertises the plain three-line usage: create, register, listen.
// It must therefore hand back an application that has not been built yet,
// otherwise the routes registered after it are never added to the mux.
func TestDefaultServesRoutesRegisteredAfterIt(t *testing.T) {
	app := iris.Default()
	app.Get("/ping", func(ctx iris.Context) {
		ctx.WriteString("pong")
	})

	e := httptest.New(t, app)
	e.GET("/ping").Expect().Status(httptest.StatusOK).Body().IsEqual("pong")
}

// The middleware set Default documents is registered on the returned
// application: wildcard CORS, the Server header and a request id.
func TestDefaultRegistersItsMiddlewares(t *testing.T) {
	app := iris.Default()
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString(ctx.ID().(string))
	})

	e := httptest.New(t, app)
	resp := e.GET("/").WithHeader("Origin", "https://example.com").Expect().
		Status(httptest.StatusOK)
	resp.Header("Access-Control-Allow-Origin").IsEqual("https://example.com")
	resp.Header("Server").IsEqual("Iris")
	resp.Header("X-Request-Id").NotEmpty()
	resp.Body().NotEmpty()
}
