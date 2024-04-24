package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	sess := sessions.New(sessions.Config{Cookie: "session_cookie", AllowReclaim: true})
	app.Use(sess.Handler())
	// ^ use app.UseRouter instead to access sessions on HTTP errors too.

	// Register our custom middleware, after the sessions middleware.
	app.Use(setSessionViewData)

	app.Get("/", index)
	app.Listen(":8080")
}

func setSessionViewData(ctx iris.Context) {
	session := sessions.Get(ctx)
	ctx.ViewData("session", session)
	ctx.Next()
}

func index(ctx iris.Context) {
	session := sessions.Get(ctx)
	session.Set("username", "kataras")
	if err := ctx.View("index"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
	/* OR without middleware:
	if err := ctx.View("index", iris.Map{
			"session": session,
			//   {{.session.Get "username"}}
			// OR to pass only the 'username':
			// "username": session.Get("username"),
			//   {{.username}}
		})
	*/
}
