package main

import (
	"github.com/kataras/iris"
)

// User is just a bindable object structure.
type User struct {
	Username  string `json:"username"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	City      string `json:"city"`
	Age       int    `json:"age"`
}

func main() {
	app := iris.New()
	// app.Logger().SetLevel("disable") to disable the logger

	// Define templates using the std html/template engine.
	// Parse and load all files inside "./views" folder with ".html" file extension.
	// Reload the templates on each request (development mode).
	app.RegisterView(iris.HTML("./views", ".html").Reload(true))

	// Register custom handler for specific http errors.
	app.OnErrorCode(iris.StatusInternalServerError, func(ctx iris.Context) {
		// .Values are used to communicate between handlers, middleware.
		errMessage := ctx.Values().GetString("error")
		if errMessage != "" {
			ctx.Writef("Internal server error: %s", errMessage)
			return
		}

		ctx.Writef("(Unexpected) internal server error")
	})

	app.Use(func(ctx iris.Context) {
		ctx.Application().Logger().Infof("Begin request for path: %s", ctx.Path())
		ctx.Next()
	})
	// app.Done(func(ctx iris.Context) {]})

	// POST: scheme://mysubdomain.$domain.com/decode
	app.Subdomain("mysubdomain.").Post("/decode", func(ctx iris.Context) {})
	// Method POST: http://localhost:8080/decode
	app.Post("/decode", func(ctx iris.Context) {
		var user User
		ctx.ReadJSON(&user)
		ctx.Writef("%s %s is %d years old and comes from %s", user.Firstname, user.Lastname, user.Age, user.City)
	})

	// Method GET: http://localhost:8080/encode
	app.Get("/encode", func(ctx iris.Context) {
		doe := User{
			Username:  "Johndoe",
			Firstname: "John",
			Lastname:  "Doe",
			City:      "Neither FBI knows!!!",
			Age:       25,
		}

		ctx.JSON(doe)
	})

	// Method GET: http://localhost:8080/profile/anytypeofstring
	app.Get("/profile/{username:string}", profileByUsername)

	usersRoutes := app.Party("/users", logThisMiddleware)
	{
		// Method GET: http://localhost:8080/users/42
		usersRoutes.Get("/{id:int min(1)}", getUserByID)
		// Method POST: http://localhost:8080/users/create
		usersRoutes.Post("/create", createUser)
	}

	// Listen for incoming HTTP/1.x & HTTP/2 clients on localhost port 8080.
	app.Run(iris.Addr(":8080"), iris.WithCharset("UTF-8"))
}

func logThisMiddleware(ctx iris.Context) {
	ctx.Application().Logger().Infof("Path: %s | IP: %s", ctx.Path(), ctx.RemoteAddr())

	// .Next is required to move forward to the chain of handlers,
	// if missing then it stops the execution at this handler.
	ctx.Next()
}

func profileByUsername(ctx iris.Context) {
	// .Params are used to get dynamic path parameters.
	username := ctx.Params().Get("username")
	ctx.ViewData("Username", username)
	// renders "./views/user/profile.html"
	// with {{ .Username }} equals to the username dynamic path parameter.
	ctx.View("user/profile.html")
}

func getUserByID(ctx iris.Context) {
	userID := ctx.Params().Get("id") // Or convert directly using: .Values().GetInt/GetInt64 etc...
	// your own db fetch here instead of user :=...
	user := User{Username: "username" + userID}

	ctx.XML(user)
}

func createUser(ctx iris.Context) {
	var user User
	err := ctx.ReadForm(&user)
	if err != nil {
		ctx.Values().Set("error", "creating user, read and parse form failed. "+err.Error())
		ctx.StatusCode(iris.StatusInternalServerError)
		return
	}
	// renders "./views/user/create_verification.html"
	// with {{ . }} equals to the User object, i.e {{ .Username }} , {{ .Firstname}} etc...
	ctx.ViewData("", user)
	ctx.View("user/create_verification.html")
}
