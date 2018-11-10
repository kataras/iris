package versioning_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/kataras/iris/versioning"
)

func TestGetVersion(t *testing.T) {
	app := iris.New()

	writeVesion := func(ctx iris.Context) {
		ctx.WriteString(versioning.GetVersion(ctx))
	}

	app.Get("/", writeVesion)
	app.Get("/manual", func(ctx iris.Context) {
		ctx.Values().Set(versioning.Key, "11.0.5")
		ctx.Next()
	}, writeVesion)

	e := httptest.New(t, app)

	e.GET("/").WithHeader(versioning.AcceptVersionHeaderKey, "1.0").Expect().
		Status(iris.StatusOK).Body().Equal("1.0")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=2.1").Expect().
		Status(iris.StatusOK).Body().Equal("2.1")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=2.1 ;other=dsa").Expect().
		Status(iris.StatusOK).Body().Equal("2.1")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "version=2.1").Expect().
		Status(iris.StatusOK).Body().Equal("2.1")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "version=1").Expect().
		Status(iris.StatusOK).Body().Equal("1")

	// unknown versions.
	e.GET("/").WithHeader(versioning.AcceptVersionHeaderKey, "").Expect().
		Status(iris.StatusOK).Body().Equal(versioning.NotFound)
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=").Expect().
		Status(iris.StatusOK).Body().Equal(versioning.NotFound)
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version= ;other=dsa").Expect().
		Status(iris.StatusOK).Body().Equal(versioning.NotFound)
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "version=").Expect().
		Status(iris.StatusOK).Body().Equal(versioning.NotFound)

	e.GET("/manual").Expect().Status(iris.StatusOK).Body().Equal("11.0.5")
}
