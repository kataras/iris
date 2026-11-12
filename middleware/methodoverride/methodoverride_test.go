package methodoverride_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/methodoverride"
)

func TestMethodOverrideWrapper(t *testing.T) {
	app := iris.New()

	mo := methodoverride.New(methodoverride.Options{
		// Defaults to empty, don't save it.
		SaveOriginalMethodContextKey: "_originalMethod",
		// The rest keep their defaults:
		//
		// Methods:   []string{http.MethodPost},
		// Headers:   []string{"X-HTTP-Method", "X-HTTP-Method-Override", "X-Method-Override"},
		// FormField: "_method",
		// Query:     "_method",
	})
	// Register it with `UseRouter`.
	app.UseRouter(mo)

	var (
		expectedDelResponse  = "delete resp"
		expectedPostResponse = "post resp"
	)

	app.Post("/path", func(ctx iris.Context) {
		ctx.WriteString(expectedPostResponse)
	})

	app.Delete("/path", func(ctx iris.Context) {
		ctx.WriteString(expectedDelResponse)
	})

	app.Delete("/path2", func(ctx iris.Context) {
		_, err := ctx.Writef("%s%s", expectedDelResponse, ctx.Values().Get("_originalMethod"))
		if err != nil {
			t.Fatal(err)
		}
	})

	e := httptest.New(t, app)

	// Test headers.
	e.POST("/path").WithHeader("X-HTTP-Method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual(expectedDelResponse)
	e.POST("/path").WithHeader("X-HTTP-Method-Override", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual(expectedDelResponse)
	e.POST("/path").WithHeader("X-Method-Override", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual(expectedDelResponse)

	// Test form field value.
	e.POST("/path").WithFormField("_method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual(expectedDelResponse)

	// Test URL Query (although it's the same as form field in this case).
	e.POST("/path").WithQuery("_method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual(expectedDelResponse)

	// Test saved original method and
	// Test without registered "POST" route.
	e.POST("/path2").WithQuery("_method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual(expectedDelResponse + iris.MethodPost)

	// Test simple POST request without method override fields.
	e.POST("/path").Expect().Status(iris.StatusOK).Body().IsEqual(expectedPostResponse)

	// Test simple DELETE request.
	e.DELETE("/path").Expect().Status(iris.StatusOK).Body().IsEqual(expectedDelResponse)
}

// Options.HeadersOnly is the recommended configuration: a header may still
// override the method, but the "_method" query string and form field - which a
// cross-site form POST can set without a preflight - no longer can.
func TestOptionsHeadersOnly(t *testing.T) {
	app := iris.New()
	app.UseRouter(methodoverride.New(methodoverride.Options{HeadersOnly: true}))

	app.Post("/path", func(ctx iris.Context) {
		ctx.WriteString("post resp")
	})
	app.Delete("/path", func(ctx iris.Context) {
		ctx.WriteString("delete resp")
	})

	e := httptest.New(t, app)

	e.POST("/path").WithHeader("X-HTTP-Method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual("delete resp")
	e.POST("/path").WithQuery("_method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual("post resp")
	e.POST("/path").WithFormField("_method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual("post resp")
}

// Options.Methods, Options.Headers and Options.Getters replace the defaults
// rather than adding to them.
func TestOptionsCustomSources(t *testing.T) {
	app := iris.New()
	app.UseRouter(methodoverride.New(methodoverride.Options{
		Methods: []string{"put"}, // lower-case on purpose: it is upper-cased.
		Headers: []string{"X-Custom-Method"},
		Getters: []methodoverride.GetterFunc{
			func(ctx iris.Context) string {
				return ctx.Query().Get("custom")
			},
		},
	}))

	app.Put("/path", func(ctx iris.Context) {
		ctx.WriteString("put resp")
	})
	app.Delete("/path", func(ctx iris.Context) {
		ctx.WriteString("delete resp")
	})
	app.Post("/path", func(ctx iris.Context) {
		ctx.WriteString("post resp")
	})

	e := httptest.New(t, app)

	// PUT is overridable now, through the custom header and the custom getter.
	e.PUT("/path").WithHeader("X-Custom-Method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual("delete resp")
	e.PUT("/path").WithQuery("custom", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual("delete resp")

	// The default header and POST are no longer honored.
	e.PUT("/path").WithHeader("X-HTTP-Method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual("put resp")
	e.POST("/path").WithHeader("X-Custom-Method", iris.MethodDelete).Expect().
		Status(iris.StatusOK).Body().IsEqual("post resp")
}

// A client that can only send POST reaches a QUERY list route through the
// override header.
func TestMethodOverrideToQuery(t *testing.T) {
	app := iris.New()
	app.UseRouter(methodoverride.New(methodoverride.Options{}))
	app.Query("/books", func(ctx iris.Context) {
		ctx.WriteString(ctx.Method())
	})

	e := httptest.New(t, app)
	e.POST("/books").WithHeader("X-HTTP-Method-Override", "QUERY").
		Expect().Status(httptest.StatusOK).Body().IsEqual("QUERY")
}
