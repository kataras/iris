package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/sessions"
)

func newApp() *iris.Framework {
	app := iris.New()
	app.Adapt(httprouter.New())
	app.Adapt(sessions.New(sessions.Config{Cookie: "mysessionid"}))

	app.Get("/hello", func(ctx *iris.Context) {
		sess := ctx.Session()
		if !sess.HasFlash() /* or sess.GetFlash("name") == "", same thing here */ {
			ctx.HTML(iris.StatusUnauthorized, "<h1> Unauthorized Page! </h1>")
			return
		}

		ctx.JSON(iris.StatusOK, iris.Map{
			"Message": "Hello",
			"From":    sess.GetFlash("name"),
		})
	})

	app.Post("/login", func(ctx *iris.Context) {
		sess := ctx.Session()
		if !sess.HasFlash() {
			sess.SetFlash("name", ctx.FormValue("name"))
		}
		// let's no redirect, just set the flash message, nothing more.
	})

	return app
}

func main() {
	app := newApp()
	app.Listen(":8080")
}
