package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"

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

	// To the next routes of a party (group of routes):
	/*
		app.Use(auth)
	*/

	// For global effect, including not founds:
	/*
		app.UseRouter(auth)
	*/

	// For global effect, excluding http errors such as not founds:
	/*
		app.UseGlobal(auth) or app.Use(auth) before any route registered.
	*/

	// For single/per routes:
	/*
		app.Get("/mysecret", auth, h)
	*/

	app.Get("/", func(ctx iris.Context) { ctx.Redirect("/admin") })

	// to party

	needAuth := app.Party("/admin", auth)
	{
		//http://localhost:8080/admin
		needAuth.Get("/", handler)
		// http://localhost:8080/admin/profile
		needAuth.Get("/profile", handler)

		// http://localhost:8080/admin/settings
		needAuth.Get("/settings", handler)

		needAuth.Get("/logout", logout)
	}

	return app
}

func main() {
	app := newApp()
	// open http://localhost:8080/admin
	app.Listen(":8080")
}

func handler(ctx iris.Context) {
	// user := ctx.User().(*myUserType)
	// or ctx.User().GetRaw().(*myUserType)
	// ctx.Writef("%s %s:%s", ctx.Path(), user.Username, user.Password)
	// OR if you don't have registered custom User structs:
	username, password, _ := ctx.Request().BasicAuth()
	ctx.Writef("%s %s:%s", ctx.Path(), username, password)
}

func logout(ctx iris.Context) {
	// fires 401, invalidates the basic auth,
	// logout through javascript and ajax is a better solution though.
	err := ctx.Logout()
	if err != nil {
		errors.Internal.Err(ctx, err)
		return
	}
}
