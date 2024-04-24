package requestid_test

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/middleware/requestid"
)

func TestRequestID(t *testing.T) {
	app := iris.New()
	h := func(ctx iris.Context) {
		ctx.WriteString(requestid.Get(ctx))
	}

	def := app.Party("/default")
	{
		def.Use(requestid.New())
		def.Get("/", h)
	}

	const expectedCustomID = "my_id"
	custom := app.Party("/custom")
	{
		customGen := func(ctx *context.Context) string {
			return expectedCustomID
		}

		custom.Use(requestid.New(customGen))
		custom.Get("/", h)
	}

	const expectedErrMsg = "no id"
	customWithErr := app.Party("/custom_err")
	{
		customGen := func(ctx *context.Context) string {
			ctx.StopWithText(iris.StatusUnauthorized, expectedErrMsg)
			return ""
		}

		customWithErr.Use(requestid.New(customGen))
		customWithErr.Get("/", h)
	}

	const expectedCustomIDFromOtherMiddleware = "my custom id"
	changeID := app.Party("/custom_change_id")
	{
		changeID.Use(func(ctx iris.Context) {
			ctx.SetID(expectedCustomIDFromOtherMiddleware)
			ctx.Next()
		})
		changeID.Use(requestid.New())
		changeID.Get("/", h)
	}

	const expectedClientSentID = "client sent id"
	clientSentID := app.Party("/client_id")
	{
		clientSentID.Use(requestid.New())
		clientSentID.Get("/", h)
	}

	e := httptest.New(t, app)
	e.GET("/default").Expect().Status(httptest.StatusOK).Body().NotEmpty()
	e.GET("/custom").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedCustomID)
	e.GET("/custom_err").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual(expectedErrMsg)
	e.GET("/custom_change_id").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedCustomIDFromOtherMiddleware)
	e.GET("/client_id").WithHeader("X-Request-Id", expectedClientSentID).Expect().Header("X-Request-Id").IsEqual(expectedClientSentID)
}
