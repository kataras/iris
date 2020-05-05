package main

import (
	"fmt"
	"strings"

	"github.com/kataras/iris/v12"

	// External package to optionally filter JSON responses before sent,
	// see `sendJSON` for more.
	"github.com/jmespath/go-jmespath"
)

/*
	$ go get github.com/jmespath/go-jmespath
*/

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")

	// http://localhost:8080/users?query=[?Name == 'John Doe'].Age
	// <- client will receive the age of a user which his name is "John Doe".
	// You can also test query=[0].Name to retrieve the first user's name.
	// Or even query=[0:3].Age to print the first three ages.
	// Learn more about jmespath and how to filter:
	// http://jmespath.readthedocs.io/en/latest/ and
	// https://github.com/jmespath/go-jmespath/tree/master/fuzz/testdata
	//
	// http://localhost:8080/users
	// http://localhost:8080/users/William%20Woe
	// http://localhost:8080/users/William%20Woe/age
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	// PartyFunc is the same as usersRouter := app.Party("/users")
	// but it gives us an easy way to call router's registration functions,
	// i.e functions from another package that can handle this group of routes.
	app.PartyFunc("/users", registerUsersRoutes)

	return app
}

/*
	START OF USERS ROUTER
*/

func registerUsersRoutes(usersRouter iris.Party) {
	// GET: /users
	usersRouter.Get("/", getAllUsersHandler)
	usersRouter.Party("/{name}").ConfigureContainer(registerUserRoutes)
}

type user struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

var usersSample = []*user{
	{"William Woe", 25},
	{"Mary Moe", 15},
	{"John Doe", 17},
}

func getAllUsersHandler(ctx iris.Context) {
	err := sendJSON(ctx, usersSample)
	if err != nil {
		fail(ctx, iris.StatusInternalServerError, "unable to send a list of all users: %v", err)
		return
	}
}

/*
	START OF USERS.USER SUB ROUTER
*/

func registerUserRoutes(userRouter *iris.APIContainer) {
	userRouter.RegisterDependency(userDependency)
	// GET: /users/{name:string}
	userRouter.Get("/", getUserHandler)
	// GET: /users/{name:string}/age
	userRouter.Get("/age", getUserAgeHandler)
}

var userDependency = func(ctx iris.Context) *user {
	name := strings.Title(ctx.Params().Get("name"))
	for _, u := range usersSample {
		if u.Name == name {
			return u
		}
	}

	// you may want or no to handle the error here, either way the main route handler
	// is going to be executed, always. A dynamic dependency(per-request) is not a middleware, so things like `ctx.Next()` or `ctx.StopExecution()`
	// do not apply here, look the `getUserHandler`'s first lines; we stop/exit the handler manually
	// if the received user is nil but depending on your app's needs, it is possible to do other things too.
	// A dynamic dependency like this can return more output values, i.e (*user, bool).
	fail(ctx, iris.StatusNotFound, "user with name '%s' not found", name)
	return nil
}

func getUserHandler(ctx iris.Context, u *user) {
	sendJSON(ctx, u)
}

func getUserAgeHandler(u *user) string {
	return fmt.Sprintf("%d", u.Age)
}

/* END OF USERS.USER SUB ROUTER */

/* END OF USERS ROUTER */

// common JSON response for manual HTTP errors, optionally.
type httpError struct {
	Code   int    `json:"code"`
	Reason string `json:"reason"`
}

func (h httpError) Error() string {
	return fmt.Sprintf("Status Code: %d\nReason: %s", h.Code, h.Reason)
}

func fail(ctx iris.Context, statusCode int, format string, a ...interface{}) {
	err := httpError{
		Code:   statusCode,
		Reason: fmt.Sprintf(format, a...),
	}

	// log all the >= 500 internal errors.
	if statusCode >= 500 {
		ctx.Application().Logger().Error(err)
	}

	// no next handlers will run.
	ctx.StopWithJSON(statusCode, err)
}

// JSON helper to give end-user the ability to put indention chars or filtering the response, you can do that, optionally.
// If you'd like to see that function inside the Iris' Context itself raise a [Feature Request] issue and link this example.
func sendJSON(ctx iris.Context, resp interface{}) (err error) {
	indent := ctx.URLParamDefault("indent", "  ")
	// i.e [?Name == 'John Doe'].Age # to output the [age] of a user which his name is "John Doe".
	if query := ctx.URLParam("query"); query != "" && query != "[]" {
		resp, err = jmespath.Search(query, resp)
		if err != nil {
			return
		}
	}

	_, err = ctx.JSON(resp, iris.JSON{Indent: indent, UnescapeHTML: true})
	return err
}
