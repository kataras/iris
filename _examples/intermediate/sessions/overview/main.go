package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"

	"github.com/kataras/iris/sessions"
)

var (
	key = "my_sessionid"
)

func secret(ctx context.Context) {

	// Check if user is authenticated
	if auth, _ := ctx.Session().GetBoolean("authenticated"); !auth {
		ctx.StatusCode(iris.StatusForbidden)
		return
	}

	// Print secret message
	ctx.WriteString("The cake is a lie!")
}

func login(ctx context.Context) {
	session := ctx.Session()

	// Authentication goes here
	// ...

	// Set user as authenticated
	session.Set("authenticated", true)
}

func logout(ctx context.Context) {
	session := ctx.Session()

	// Revoke users authentication
	session.Set("authenticated", false)
}

func main() {
	app := iris.New()

	// Look https://github.com/kataras/iris/tree/master/sessions/_examples for more features,
	// i.e encode/decode and lifetime.
	sess := sessions.New(sessions.Config{Cookie: key})
	app.AttachSessionManager(sess)

	app.Get("/secret", secret)
	app.Get("/login", login)
	app.Get("/logout", logout)

	app.Run(iris.Addr(":8080"))
}
