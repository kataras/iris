package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

func newApp() *iris.Application {
	app := iris.New()

	authConfig := basicauth.Config{
		Users:   map[string]string{"myusername": "mypassword", "mySecondusername": "mySecondpassword"},
		Realm:   "Authorization Required", // defaults to "Authorization Required"
		Expires: time.Duration(30) * time.Minute,
	}

	authentication := basicauth.New(authConfig)

	// to global app.Use(authentication) (or app.UseGlobal before the .Run)
	// to routes
	/*
		app.Get("/mysecret", authentication, h)
	*/

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

		needAuth.Get("/logout", logout)
	}

	return app
}

func main() {
	app := newApp()
	// open http://localhost:8080/admin
	app.Listen(":8080")
}

func h(ctx iris.Context) {
	// username, password, _ := ctx.Request().BasicAuth()
	// third parameter it will be always true because the middleware
	// makes sure for that, otherwise this handler will not be executed.
	// OR:
	user := ctx.User()
	ctx.Writef("%s %s:%s", ctx.Path(), user.GetUsername(), user.GetPassword())
}

func logout(ctx iris.Context) {
	err := ctx.Logout() // fires 401, invalidates the basic auth.
	if err != nil {
		ctx.Application().Logger().Errorf("Logout error: %v", err)
	}
	ctx.Redirect("/admin", iris.StatusTemporaryRedirect)
}
