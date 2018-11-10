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

func TestNewMatcher(t *testing.T) {
	app := iris.New()

	userAPI := app.Party("/api/user")
	userAPI.Get("/", versioning.NewMatcher(versioning.Map{
		"1.0":               sendHandler(v10Response),
		">= 2, < 3":         sendHandler(v2Response),
		versioning.NotFound: notFoundHandler,
	}))

	e := httptest.New(t, app)

	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "1").Expect().
		Status(iris.StatusOK).Body().Equal(v10Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.0").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.1").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)
	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "2.9.9").Expect().
		Status(iris.StatusOK).Body().Equal(v2Response)

	e.GET("/api/user").WithHeader(versioning.AcceptVersionHeaderKey, "3.0").Expect().
		Status(iris.StatusNotFound).Body().Equal("Not Found")
}

func TestNewGroup(t *testing.T) {
	app := iris.New()
	// userAPI := app.Party("/api/user")

	// userAPIV10 := versioning.NewGroup("1.0", userAPI)
	// userAPIV10.Get("/", sendHandler(v10Response))
	// userAPIV2 := versioning.NewGroup(">= 2, < 3", userAPI)
	// userAPIV2.Get("/", sendHandler(v2Response))

	// ---

	// userAPI := app.Party("/api/user")
	// userVAPI := versioning.NewGroup(userAPI)
	// userAPIV10 := userVAPI.Version("1.0")
	// userAPIV10.Get("/", sendHandler(v10Response))

	// userAPIV10 := userVAPI.Version("2.0")
	// userAPIV10.Get("/", sendHandler(v10Response))
	// userVAPI.NotFound(...)
	// userVAPI.Build()

	// --

	userAPI := app.Party("/api/user")
	// [... static serving, middlewares and etc goes here].

	userAPIV10 := versioning.NewGroup("1.0").Deprecated(versioning.DefaultDeprecationOptions)
	userAPIV10.Get("/", sendHandler(v10Response))

	userAPIV2 := versioning.NewGroup(">= 2, < 3")
	userAPIV2.Get("/", sendHandler(v2Response))
	userAPIV2.Post("/", sendHandler(v2Response))
	userAPIV2.Put("/other", sendHandler(v2Response))

	// versioning.Concat(userAPIV10, userAPIV2)
	// 	NotFound(func(ctx iris.Context) {
	// 		ctx.StatusCode(iris.StatusNotFound)
	// 		ctx.Writef("unknown version %s", versioning.GetVersion(ctx))
	// 	}).
	// For(userAPI)
	// This is legal too:
	// For(app.PartyFunc("/api/user", func(r iris.Party) {
	// 	// [... static serving, middlewares and etc goes here].
	// }))

	versioning.RegisterGroups(userAPI, userAPIV10, userAPIV2)

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
