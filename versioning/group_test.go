package versioning_test

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/versioning"
)

const (
	v10Response = "v1.0 handler"
	v2Response  = "v2.x handler"
)

func sendHandler(contents string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.WriteString(contents)
	}
}

func TestNewGroup(t *testing.T) {
	app := iris.New()

	userAPI := app.Party("/api/user")
	// [... static serving, middlewares and etc goes here].

	userAPIV10 := versioning.NewGroup(userAPI, "1.0.0").Deprecated(versioning.DefaultDeprecationOptions)

	userAPIV10.Get("/", sendHandler(v10Response))
	userAPIV2 := versioning.NewGroup(userAPI, ">= 2.0.0 < 3.0.0")

	userAPIV2.Get("/", sendHandler(v2Response))
	userAPIV2.Post("/", sendHandler(v2Response))
	userAPIV2.Put("/other", sendHandler(v2Response))

	e := httptest.New(t, app)

	ex := e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "1.0.0").Expect()
	ex.Status(iris.StatusOK).Body().IsEqual(v10Response)
	ex.Header("X-API-Warn").Equal(versioning.DefaultDeprecationOptions.WarnMessage)

	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.0.0").Expect().
		Status(iris.StatusOK).Body().IsEqual(v2Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.1.0").Expect().
		Status(iris.StatusOK).Body().IsEqual(v2Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.9.9").Expect().
		Status(iris.StatusOK).Body().IsEqual(v2Response)
	e.POST("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.0.0").Expect().
		Status(iris.StatusOK).Body().IsEqual(v2Response)
	e.PUT("/api/user/other").WithHeader(versioning.AcceptVersionHeaderKey, "2.9.0").Expect().
		Status(iris.StatusOK).Body().IsEqual(v2Response)

	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "3.0").Expect().
		Status(iris.StatusNotImplemented).Body().IsEqual("version not found")
}
