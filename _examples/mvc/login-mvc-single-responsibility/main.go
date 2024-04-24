package main

import (
	"time"

	"github.com/kataras/iris/v12/_examples/mvc/login-mvc-single-responsibility/user"

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

	app.HandleDir("/public", iris.Dir("./public"))

	userRouter := app.Party("/user")
	{
		manager := sessions.New(sessions.Config{
			Cookie:  "sessioncookiename",
			Expires: 24 * time.Hour,
		})
		userRouter.Use(manager.Handler())
		mvc.Configure(userRouter, configureUserMVC)
	}

	// http://localhost:8080/user/register
	// http://localhost:8080/user/login
	// http://localhost:8080/user/me
	// http://localhost:8080/user/logout
	// http://localhost:8080/user/1
	app.Listen(":8080", configure)
}

func configureUserMVC(userApp *mvc.Application) {
	userApp.Register(
		user.NewDataSource(),
	)
	userApp.Handle(new(user.Controller))
}

func configure(app *iris.Application) {
	app.Configure(
		iris.WithOptimizations,
		iris.WithFireMethodNotAllowed,
		iris.WithLowercaseRouting,
		iris.WithPathIntelligence,
		iris.WithTunneling,
	)
}
