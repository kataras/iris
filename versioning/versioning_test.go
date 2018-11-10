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

func TestHandler(t *testing.T) {
	app := iris.New()

	userAPI := app.Party("/api/user")
	userAPI.Get("/", versioning.Handler(versioning.Map{
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
