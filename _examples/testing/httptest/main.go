package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

func newApp() *iris.Application {
	app := iris.New()

	opts := basicauth.Options{
		Allow: basicauth.AllowUsers(map[string]string{"myusername": "mypassword"}),
	}

	authentication := basicauth.New(opts) // or just: basicauth.Default(map...)

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

func main() {
	app := newApp()
	app.Listen(":8080")
}
