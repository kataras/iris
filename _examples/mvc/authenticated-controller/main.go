// Package main shows how to use a dependency to check if a user is logged in
// using a special custom Go type `Authenticated`, which when,
// present on a controller's method or a field then
// it limits the visibility to "authenticated" users only.
//
// The same result could be done through a middleware as well, however using a static type
// any person reads your code can see that an "X" controller or method
// should only be used by "authenticated" users.
package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	sess := sessions.New(sessions.Config{
		Cookie:       "myapp_session_id",
		AllowReclaim: true,
	})
	app.Use(sess.Handler())

	userRouter := app.Party("/user")
	{
		// Initialize a new MVC application on top of the "userRouter".
		userApp := mvc.New(userRouter)
		// Register Dependencies.
		userApp.Register(authDependency)

		// Register Controllers.
		userApp.Handle(new(MeController))
		userApp.Handle(new(UserController))
	}

	// Open a client, e.g. Postman and visit the below endpoints.
	// GET: http://localhost:8080/user
	// POST: http://localhost:8080/user/login
	// GET: http://localhost:8080/user
	// GET: http://localhost:8080/user/me
	// POST: http://localhost:8080/user/logout
	app.Listen(":8080")
}

// Authenticated is a custom type used as "annotation" for resources that requires authentication,
// its value should be the logged user ID.
type Authenticated uint64

func authDependency(ctx iris.Context, session *sessions.Session) Authenticated {
	userID := session.GetUint64Default("user_id", 0)
	if userID == 0 {
		// If execution was stopped
		// any controller's method will not be executed at all.
		ctx.StopWithStatus(iris.StatusUnauthorized)
		return 0
	}

	return Authenticated(userID)
}

// UserController serves the "public" User API.
type UserController struct {
	Session *sessions.Session
}

// PostLogin serves
// POST: /user/login
func (c *UserController) PostLogin() mvc.Response {
	c.Session.Set("user_id", 1)

	// Redirect (you can still use the Context.Redirect if you want so).
	return mvc.Response{
		Path: "/user",
		Code: iris.StatusFound,
	}
}

// PostLogout serves
// POST: /user/logout
func (c *UserController) PostLogout(ctx iris.Context) {
	c.Session.Man.Destroy(ctx)
}

// GetMe showcases that the same type can be used inside controller's method too,
// a second controller like `MeController` is not required.
// GET: user/me
func (c *UserController) GetMe(_ Authenticated) string {
	return `UserController.GetMe: The Authenticated type
can be used to secure a controller's method too.`
}

// MeController provides the logged user's available actions.
type MeController struct {
	CurrentUserID Authenticated
}

// Get returns a message for the shake of the example.
// GET: /user
func (c *MeController) Get() string {
	return "This will be executed only when the user is logged in"
}
