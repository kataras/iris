package versioning_test

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/versioning"
)

func TestIf(t *testing.T) {
	if expected, got := true, versioning.If("1.0.0", ">=1.0.0"); expected != got {
		t.Fatalf("expected %s to be %s", "1.0.0", ">= 1.0.0")
	}
	if expected, got := true, versioning.If("1.2.3", "> 1.2.0"); expected != got {
		t.Fatalf("expected %s to be %s", "1.2.3", "> 1.2.0")
	}
}

func TestGetVersion(t *testing.T) {
	app := iris.New()

	writeVesion := func(ctx iris.Context) {
		ctx.WriteString(versioning.GetVersion(ctx))
	}

	app.Get("/", writeVesion)
	app.Get("/manual", func(ctx iris.Context) {
		versioning.SetVersion(ctx, "11.0.5")
		ctx.Next()
	}, writeVesion)

	e := httptest.New(t, app)

	e.GET("/").WithHeader(versioning.AcceptVersionHeaderKey, "1.0.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("1.0.0")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=2.1.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("2.1.0")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=2.1.0 ;other=dsa").Expect().
		Status(iris.StatusOK).Body().IsEqual("2.1.0")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "version=2.1.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("2.1.0")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "version=1.0.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("1.0.0")

	// unknown versions.
	e.GET("/").WithHeader(versioning.AcceptVersionHeaderKey, "").Expect().
		Status(iris.StatusOK).Body().IsEqual("")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version=").Expect().
		Status(iris.StatusOK).Body().IsEqual("")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "application/vnd.api+json; version= ;other=dsa").Expect().
		Status(iris.StatusOK).Body().IsEqual("")
	e.GET("/").WithHeader(versioning.AcceptHeaderKey, "version=").Expect().
		Status(iris.StatusOK).Body().IsEqual("")

	e.GET("/manual").Expect().Status(iris.StatusOK).Body().IsEqual("11.0.5")
}

func TestVersionAliases(t *testing.T) {
	app := iris.New()

	api := app.Party("/api")
	api.Use(versioning.Aliases(map[string]string{
		versioning.Empty: "1.0.0",
		"stage":          "2.0.0",
	}))

	writeVesion := func(ctx iris.Context) {
		ctx.WriteString(versioning.GetVersion(ctx))
	}

	// A group without registration order.
	v3 := versioning.NewGroup(api, ">= 3.0.0 < 4.0.0")
	v3.Get("/", writeVesion)

	v1 := versioning.NewGroup(api, ">= 1.0.0 < 2.0.0")
	v1.Get("/", writeVesion)

	v2 := versioning.NewGroup(api, ">= 2.0.0 < 3.0.0")
	v2.Get("/", writeVesion)

	api.Get("/manual", func(ctx iris.Context) {
		versioning.SetVersion(ctx, "12.0.0")
		ctx.Next()
	}, writeVesion)

	e := httptest.New(t, app)

	// Make sure the SetVersion still works.
	e.GET("/api/manual").Expect().Status(iris.StatusOK).Body().IsEqual("12.0.0")

	// Test Empty default.
	e.GET("/api").WithHeader(versioning.AcceptVersionHeaderKey, "").Expect().
		Status(iris.StatusOK).Body().IsEqual("1.0.0")
	// Test NotFound error, aliases are not responsible for that.
	e.GET("/api").WithHeader(versioning.AcceptVersionHeaderKey, "4.0.0").Expect().
		Status(iris.StatusNotImplemented).Body().IsEqual("version not found")
	// Test "stage" alias.
	e.GET("/api").WithHeader(versioning.AcceptVersionHeaderKey, "stage").Expect().
		Status(iris.StatusOK).Body().IsEqual("2.0.0")
	// Test version 2.
	e.GET("/api").WithHeader(versioning.AcceptVersionHeaderKey, "2.0.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("2.0.0")
	// Test version 3 (registered first).
	e.GET("/api").WithHeader(versioning.AcceptVersionHeaderKey, "3.1.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("3.1.0")
}
