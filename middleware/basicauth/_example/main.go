package main

import (
	"time"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/middleware/basicauth"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger()) // adapt a simple internal logger to print any errors
	app.Adapt(httprouter.New()) // adapt a router, you can use gorillamux too

	authConfig := basicauth.Config{
		Users:      map[string]string{"myusername": "mypassword", "mySecondusername": "mySecondpassword"},
		Realm:      "Authorization Required", // defaults to "Authorization Required"
		ContextKey: "mycustomkey",            // defaults to "user"
		Expires:    time.Duration(30) * time.Minute,
	}

	authentication := basicauth.New(authConfig)
	app.Get("/", func(ctx *iris.Context) { ctx.Redirect("/admin") })
	// to global app.Use(authentication) (or app.UseGlobal before the .Listen)
	// to routes
	/*
		app.Get("/mysecret", authentication, func(ctx *iris.Context) {
			username := ctx.GetString("mycustomkey") //  the Contextkey from the authConfig
			ctx.Writef("Hello authenticated user: %s ", username)
		})
	*/

	// to party

	needAuth := app.Party("/admin", authentication)
	{
		//http://localhost:8080/admin
		needAuth.Get("/", func(ctx *iris.Context) {
			username := ctx.GetString("mycustomkey") //  the Contextkey from the authConfig
			ctx.Writef("Hello authenticated user: %s from: %s ", username, ctx.Path())
		})
		// http://localhost:8080/admin/profile
		needAuth.Get("/profile", func(ctx *iris.Context) {
			username := ctx.GetString("mycustomkey") //  the Contextkey from the authConfig
			ctx.Writef("Hello authenticated user: %s from: %s ", username, ctx.Path())
		})
		// http://localhost:8080/admin/settings
		needAuth.Get("/settings", func(ctx *iris.Context) {
			username := authConfig.User(ctx) // shortcut for ctx.GetString("mycustomkey")
			ctx.Writef("Hello authenticated user: %s from: %s ", username, ctx.Path())
		})
	}

	// open http://localhost:8080/admin
	app.Listen(":8080")
}
