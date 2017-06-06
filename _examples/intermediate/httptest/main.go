package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions"
)

func main() {
	app := iris.New()

	app.AttachSessionManager(sessions.New(sessions.Config{Cookie: "mysessionid"}))

	app.Get("/hello", func(ctx context.Context) {
		sess := ctx.Session()
		if !sess.HasFlash() {
			ctx.HTML("<h1> Unauthorized Page! </h1>")
			return
		}

		ctx.JSON(context.Map{
			"Message": "Hello",
			"From":    sess.GetFlash("name"),
		})
	})

	app.Post("/login", func(ctx context.Context) {
		sess := ctx.Session()
		if !sess.HasFlash() {
			sess.SetFlash("name", ctx.FormValue("name"))
		}

	})
	app.Run(iris.Addr(":8080"))
}
