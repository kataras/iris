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

// Default installs the rest/errors path parameter handler on its own macros,
// so a value that is not an int is answered with the 400 envelope instead of
// being offered to the next route of the same path. That choice belongs to the
// application Default returns: one created with New in the same process,
// before it or after it, keeps falling through to the string route. The macros
// used to be one collection shared by every application, and a single Default
// changed every route registered after it.
func TestDefaultKeepsItsPathParameterErrorHandlerToItself(t *testing.T) {
	register := func(app *iris.Application) {
		app.Get("/users/{name:string}", func(ctx iris.Context) {
			ctx.Writef("string %s", ctx.Params().GetString("name"))
		})
		app.Get("/users/{id:int}", func(ctx iris.Context) {
			ctx.Writef("int %d", ctx.Params().GetInt("id"))
		})
	}

	// The test creates its own Default, so it does not matter what the tests
	// that ran earlier in this binary created.
	before := iris.New()
	def := iris.Default()
	after := iris.New()

	for _, tt := range []struct {
		when string
		app  *iris.Application
	}{{"before", before}, {"after", after}} {
		register(tt.app)

		e := httptest.New(t, tt.app)
		e.GET("/users/bob").Expect().Status(httptest.StatusOK).Body().IsEqual("string bob")
		e.GET("/users/42").Expect().Status(httptest.StatusOK).Body().IsEqual("int 42")
		if t.Failed() {
			t.Fatalf("the application created with iris.New() %s iris.Default() answers through the error handler of Default", tt.when)
		}
	}

	register(def)
	e := httptest.New(t, def)
	e.GET("/users/bob").Expect().Status(httptest.StatusBadRequest).
		JSON().Object().HasValue("message", "invalid path parameter")
	e.GET("/users/42").Expect().Status(httptest.StatusOK).Body().IsEqual("int 42")
}
