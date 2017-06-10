package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/middleware/basicauth"
)

func buildApp() *iris.Application {
	app := iris.New()

	authConfig := basicauth.Config{
		Users:      map[string]string{"myusername": "mypassword", "mySecondusername": "mySecondpassword"},
		Realm:      "Authorization Required", // defaults to "Authorization Required"
		ContextKey: "user",                   // defaults to "user"
	}

	authentication := basicauth.New(authConfig)

	// to global app.Use(authentication) (or app.UseGlobal before the .Run)
	// to routes
	/*
		app.Get("/mysecret", authentication, func(ctx context.Context) {
			username := ctx.Values().GetString("user") //  the Contextkey from the authConfig
			ctx.Writef("Hello authenticated user: %s ", username)
		})
	*/

	app.Get("/", func(ctx context.Context) { ctx.Redirect("/admin") })

	// to party

	needAuth := app.Party("/admin", authentication)
	{
		//http://localhost:8080/admin
		needAuth.Get("/", func(ctx context.Context) {
			username := ctx.Values().GetString("user") //  the Contextkey from the authConfig
			ctx.Writef("Hello authenticated user: %s from: %s", username, ctx.Path())
		})
		// http://localhost:8080/admin/profile
		needAuth.Get("/profile", func(ctx context.Context) {
			username := ctx.Values().GetString("user") //  the Contextkey from the authConfig
			ctx.Writef("Hello authenticated user: %s from: %s", username, ctx.Path())
		})

		// http://localhost:8080/admin/settings
		needAuth.Get("/settings", func(ctx context.Context) {
			username := authConfig.User(ctx) // shortcut for ctx.Values().GetString("user")
			ctx.Writef("Hello authenticated user: %s from: %s", username, ctx.Path())
		})
	}

	return app
}

func main() {
	app := buildApp()
	app.Run(iris.Addr(":8080"))
}
