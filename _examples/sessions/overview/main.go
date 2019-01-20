package main

import (
	"github.com/kataras/iris"

	"github.com/kataras/iris/sessions"
)

var (
	cookieNameForSessionID = "mycookiesessionnameid"
	sess                   = sessions.New(sessions.Config{Cookie: cookieNameForSessionID, AllowReclaim: true})
)

func secret(ctx iris.Context) {

	// Check if user is authenticated
	if auth, _ := sess.Start(ctx).GetBoolean("authenticated"); !auth {
		ctx.StatusCode(iris.StatusForbidden)
		return
	}

	// Print secret message
	ctx.WriteString("The cake is a lie!")
}

func login(ctx iris.Context) {
	session := sess.Start(ctx)

	// Authentication goes here
	// ...

	// Set user as authenticated
	session.Set("authenticated", true)
}

func logout(ctx iris.Context) {
	session := sess.Start(ctx)

	// Revoke users authentication
	session.Set("authenticated", false)
}

func main() {
	app := iris.New()

	app.Get("/secret", secret)
	app.Get("/login", login)
	app.Get("/logout", logout)

	app.Run(iris.Addr(":8080"))
}
