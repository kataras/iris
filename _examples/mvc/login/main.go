// file: main.go

package main

import (
	"time"

	"github.com/kataras/iris/_examples/mvc/login/datasource"
	"github.com/kataras/iris/_examples/mvc/login/repositories"
	"github.com/kataras/iris/_examples/mvc/login/services"
	"github.com/kataras/iris/_examples/mvc/login/web/controllers"
	"github.com/kataras/iris/_examples/mvc/login/web/middleware"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/kataras/iris/sessions"
)

func main() {
	app := iris.New()
	// You got full debug messages, useful when using MVC and you want to make
	// sure that your code is aligned with the Iris' MVC Architecture.
	app.Logger().SetLevel("debug")

	// Load the template files.
	tmpl := iris.HTML("./web/views", ".html").
		Layout("shared/layout.html").
		Reload(true)
	app.RegisterView(tmpl)

	app.StaticWeb("/public", "./web/public")

	app.OnAnyErrorCode(func(ctx iris.Context) {
		ctx.ViewData("Message", ctx.Values().
			GetStringDefault("message", "The page you're looking for doesn't exist"))
		ctx.View("shared/error.html")
	})

	// ---- Serve our controllers. ----

	// Prepare our repositories and services.
	db, err := datasource.LoadUsers(datasource.Memory)
	if err != nil {
		app.Logger().Fatalf("error while loading the users: %v", err)
		return
	}
	repo := repositories.NewUserRepository(db)
	userService := services.NewUserService(repo)

	// "/users" based mvc application.
	users := mvc.New(app.Party("/users"))
	// Add the basic authentication(admin:password) middleware
	// for the /users based requests.
	users.Router.Use(middleware.BasicAuth)
	// Bind the "userService" to the UserController's Service (interface) field.
	users.Register(userService)
	users.Handle(new(controllers.UsersController))

	// "/user" based mvc application.
	sessManager := sessions.New(sessions.Config{
		Cookie:  "sessioncookiename",
		Expires: 24 * time.Hour,
	})
	user := mvc.New(app.Party("/user"))
	user.Register(
		userService,
		sessManager.Start,
	)
	user.Handle(new(controllers.UserController))

	// http://localhost:8080/noexist
	// and all controller's methods like
	// http://localhost:8080/users/1
	// http://localhost:8080/user/register
	// http://localhost:8080/user/login
	// http://localhost:8080/user/me
	// http://localhost:8080/user/logout
	// basic auth: "admin", "password", see "./middleware/basicauth.go" source file.
	app.Run(
		// Starts the web server at localhost:8080
		iris.Addr("localhost:8080"),
		// Ignores err server closed log when CTRL/CMD+C pressed.
		iris.WithoutServerError(iris.ErrServerClosed),
		// Enables faster json serialization and more.
		iris.WithOptimizations,
	)
}
