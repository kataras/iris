package methodoverride_test

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/middleware/methodoverride"
)

func TestMethodOverrideWrapper(t *testing.T) {
	app := iris.New()

	mo := methodoverride.New(
		// Defaults to nil.
		//
		methodoverride.SaveOriginalMethod("_originalMethod"),
		// Default values.
		//
		// methodoverride.Methods(http.MethodPost),
		// methodoverride.Headers("X-HTTP-Method", "X-HTTP-Method-Override", "X-Method-Override"),
		// methodoverride.FormField("_method"),
		// methodoverride.Query("_method"),
	)
	// Register it with `WrapRouter`.
	app.WrapRouter(mo)

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
		_, err := ctx.Writef("%s%s", expectedDelResponse, ctx.Request().Context().Value("_originalMethod"))
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
