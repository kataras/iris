package main_test

import (
	"github.com/kataras/iris/v12"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGinkgotest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ginkgotest Suite")
}

func newApp(authentication iris.Handler) *iris.Application {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) { ctx.Redirect("/admin") })

	// to party

	needAuth := app.Party("/admin", authentication)
	{
		//http://localhost:8080/admin
		needAuth.Get("/", h)
		// http://localhost:8080/admin/profile
		needAuth.Get("/profile", h)

		// http://localhost:8080/admin/settings
		needAuth.Get("/settings", h)
	}

	return app
}
func h(ctx iris.Context) {
	username, password, _ := ctx.Request().BasicAuth()
	// third parameter it will be always true because the middleware
	// makes sure for that, otherwise this handler will not be executed.
	// OR:
	//
	// user := ctx.User().(*myUserType)
	// ctx.Writef("%s %s:%s", ctx.Path(), user.Username, user.Password)
	// OR if you don't have registered custom User structs:
	//
	// ctx.User().GetUsername()
	// ctx.User().GetPassword()
	ctx.Writef("%s %s:%s", ctx.Path(), username, password)
}
