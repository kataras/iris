package main

import (
	"time"

	"github.com/kataras/iris/v12/_examples/structuring/login-mvc-single-responsibility-package/user"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"
)

func main() {
	app := iris.New()
	// You got full debug messages, useful when using MVC and you want to make
	// sure that your code is aligned with the Iris' MVC Architecture.
	app.Logger().SetLevel("debug")

	app.RegisterView(iris.HTML("./views", ".html").Layout("shared/layout.html"))

	app.HandleDir("/public", "./public")

	mvc.Configure(app, configureMVC)

	// http://localhost:8080/user/register
	// http://localhost:8080/user/login
	// http://localhost:8080/user/me
	// http://localhost:8080/user/logout
	// http://localhost:8080/user/1
	app.Listen(":8080", configure)
}

func configureMVC(app *mvc.Application) {
	manager := sessions.New(sessions.Config{
		Cookie:  "sessioncookiename",
		Expires: 24 * time.Hour,
	})

	userApp := app.Party("/user")
	userApp.Register(
		user.NewDataSource(),
		manager.Start,
	)
	userApp.Handle(new(user.Controller))
}

func configure(app *iris.Application) {
	app.Configure(
		iris.WithoutServerError(iris.ErrServerClosed),
	)
}
