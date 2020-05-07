package main

import (
	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/sessions"
)

const cookieNameForSessionID = "session_id_cookie"

func secret(ctx iris.Context) {
	// Check if user is authenticated
	if auth, _ := sessions.Get(ctx).GetBoolean("authenticated"); !auth {
		ctx.StatusCode(iris.StatusForbidden)
		return
	}

	// Print secret message
	ctx.WriteString("The cake is a lie!")
}

func login(ctx iris.Context) {
	session := sessions.Get(ctx)

	// Authentication goes here
	// ...

	// Set user as authenticated
	session.Set("authenticated", true)
}

func logout(ctx iris.Context) {
	session := sessions.Get(ctx)

	// Revoke users authentication
	session.Set("authenticated", false)
}

func main() {
	app := iris.New()
	sess := sessions.New(sessions.Config{Cookie: cookieNameForSessionID, AllowReclaim: true})
	app.Use(sess.Handler())
	// ^ or comment this line and use sess.Start(ctx) inside your handlers
	// instead of sessions.Get(ctx).

	app.Get("/secret", secret)
	app.Get("/login", login)
	app.Get("/logout", logout)

	app.Listen(":8080")
}
