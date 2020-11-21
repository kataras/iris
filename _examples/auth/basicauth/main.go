package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

func newApp() *iris.Application {
	app := iris.New()

	/*
		opts := basicauth.Options{
			Realm:  "Authorization Required",
			MaxAge: 30 * time.Minute,
			GC: basicauth.GC{
				Every: 2 * time.Hour,
			},
			Allow: basicauth.AllowUsers(map[string]string{
				"myusername":       "mypassword",
				"mySecondusername": "mySecondpassword",
			}),
			MaxTries: 2,
		}
		auth := basicauth.New(opts)

		OR simply:
	*/

	auth := basicauth.Default(map[string]string{
		"myusername":       "mypassword",
		"mySecondusername": "mySecondpassword",
	})

	// to global app.Use(auth) (or app.UseGlobal before the .Run)
	// to routes
	/*
		app.Get("/mysecret", auth, h)
	*/

	app.Get("/", func(ctx iris.Context) { ctx.Redirect("/admin") })

	// to party

	needAuth := app.Party("/admin", auth)
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
	username, _ := user.GetUsername()
	password, _ := user.GetPassword()
	ctx.Writef("%s %s:%s", ctx.Path(), username, password)
}

func logout(ctx iris.Context) {
	err := ctx.Logout() // fires 401, invalidates the basic auth.
	if err != nil {
		ctx.Application().Logger().Errorf("Logout error: %v", err)
	}
	ctx.Redirect("/admin", iris.StatusTemporaryRedirect)
}
