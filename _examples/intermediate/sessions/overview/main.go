package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/sessions"
)

var (
	key = "my_sessionid"
)

func secret(ctx *iris.Context) {

	// Check if user is authenticated
	if auth, _ := ctx.Session().GetBoolean("authenticated"); !auth {
		ctx.EmitError(iris.StatusForbidden)
		return
	}

	// Print secret message
	ctx.WriteString("The cake is a lie!")
}

func login(ctx *iris.Context) {
	session := ctx.Session()

	// Authentication goes here
	// ...

	// Set user as authenticated
	session.Set("authenticated", true)
}

func logout(ctx *iris.Context) {
	session := ctx.Session()

	// Revoke users authentication
	session.Set("authenticated", false)
}

func main() {
	app := iris.New()
	app.Adapt(httprouter.New())
	// Look https://github.com/kataras/iris/tree/v6/adaptors/sessions/_examples for more features,
	// i.e encode/decode and lifetime.
	sess := sessions.New(sessions.Config{Cookie: key})
	app.Adapt(sess)

	app.Get("/secret", secret)
	app.Get("/login", login)
	app.Get("/logout", logout)

	app.Listen(":8080")
}
