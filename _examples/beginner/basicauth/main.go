package main

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/middleware/basicauth"
)

func main() {
	app := iris.New()

	authConfig := basicauth.Config{
		Users:      map[string]string{"myusername": "mypassword", "mySecondusername": "mySecondpassword"},
		Realm:      "Authorization Required", // defaults to "Authorization Required"
		ContextKey: "user",                   // defaults to "user"
		Expires:    time.Duration(30) * time.Minute,
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
			ctx.Writef("Hello authenticated user: %s from: % ", username, ctx.Path())
		})

		// http://localhost:8080/admin/settings
		needAuth.Get("/settings", func(ctx context.Context) {
			username := authConfig.User(ctx) // shortcut for ctx.Values().GetString("user")
			ctx.Writef("Hello authenticated user: %s from: %s", username, ctx.Path())
		})
	}

	// open http://localhost:8080/admin
	app.Run(iris.Addr(":8080"))
}
