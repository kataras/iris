package versioning_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/kataras/iris/versioning"
)

func notFoundHandler(ctx iris.Context) {
	ctx.NotFound()
}

const (
	v10Response = "v1.0 handler"
	v2Response  = "v2.x handler"
)

func sendHandler(contents string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.WriteString(contents)
	}
}

func TestIf(t *testing.T) {
	if expected, got := true, versioning.If("1.0", ">=1"); expected != got {
		t.Fatalf("expected %s to be %s", "1.0", ">= 1")
	}
	if expected, got := true, versioning.If("1.2.3", "> 1.2"); expected != got {
		t.Fatalf("expected %s to be %s", "1.2.3", "> 1.2")
	}
}
func TestNewMatcher(t *testing.T) {
	app := iris.New()

	userAPI := app.Party("/api/user")
	userAPI.Get("/", versioning.NewMatcher(versioning.Map{
		"1.0":               sendHandler(v10Response),
		">= 2, < 3":         sendHandler(v2Response),
		versioning.NotFound: notFoundHandler,
	}))

	// middleware as usual.
	myMiddleware := func(ctx iris.Context) {
		ctx.Header("X-Custom", "something")
		ctx.Next()
	}
	myVersions := versioning.Map{
		"1.0": sendHandler(v10Response),
	}

	userAPI.Get("/with_middleware", myMiddleware, versioning.NewMatcher(myVersions))

	e := httptest.New(t, app)

	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "1").Expect().
		Status(iris.StatusOK).Body().Equal(v10Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.0").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.1").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.9.9").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)

	// middleware as usual.
	ex := e.GET("/api/user/with_middleware").WithHeader(versioning.AcceptVersionHeaderKey, "1.0").Expect()
	ex.Status(iris.StatusOK).Body().Equal(v10Response)
	ex.Header("X-Custom").Equal("something")

	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "3.0").Expect().
		Status(iris.StatusNotFound).Body().Equal("Not Found")
}

func TestNewGroup(t *testing.T) {
	app := iris.New()

	userAPI := app.Party("/api/user")
	// [... static serving, middlewares and etc goes here].

	userAPIV10 := versioning.NewGroup("1.0").Deprecated(versioning.DefaultDeprecationOptions)
	// V10middlewareResponse := "m1"
	// userAPIV10.Use(func(ctx iris.Context) {
	// 	println("exec userAPIV10.Use - midl1")
	// 	sendHandler(V10middlewareResponse)(ctx)
	// 	ctx.Next()
	// })
	// userAPIV10.Use(func(ctx iris.Context) {
	// 	println("exec userAPIV10.Use - midl2")
	// 	sendHandler(V10middlewareResponse + "midl2")(ctx)
	// 	ctx.Next()
	// })
	// userAPIV10.Use(func(ctx iris.Context) {
	// 	println("exec userAPIV10.Use - midl3")
	// 	ctx.Next()
	// })

	userAPIV10.Get("/", sendHandler(v10Response))
	userAPIV2 := versioning.NewGroup(">= 2, < 3")
	// V2middlewareResponse := "m2"
	// userAPIV2.Use(func(ctx iris.Context) {
	// 	println("exec userAPIV2.Use - midl1")
	// 	sendHandler(V2middlewareResponse)(ctx)
	// 	ctx.Next()
	// })
	// userAPIV2.Use(func(ctx iris.Context) {
	// 	println("exec userAPIV2.Use - midl2")
	// 	ctx.Next()
	// })

	userAPIV2.Get("/", sendHandler(v2Response))
	userAPIV2.Post("/", sendHandler(v2Response))
	userAPIV2.Put("/other", sendHandler(v2Response))

	versioning.RegisterGroups(userAPI, versioning.NotFoundHandler, userAPIV10, userAPIV2)

	e := httptest.New(t, app)

	ex := e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "1").Expect()
	ex.Status(iris.StatusOK).Body().Equal(v10Response)
	ex.Header("X-API-Warn").Equal(versioning.DefaultDeprecationOptions.WarnMessage)

	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.0").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.1").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.9.9").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)
	e.POST("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)
	e.PUT("/api/user/other").WithHeader(versioning.AcceptVersionHeaderKey, "2.9").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)

	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "3.0").Expect().
		Status(iris.StatusNotImplemented).Body().Equal("version not found")
}
