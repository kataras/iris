package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()
	v1 := app.Party("/api/v1")

	myFilter := func(ctx iris.Context) bool {
		// don't do that on production, use session or/and database calls and etc.
		ok, _ := ctx.URLParamBool("admin")
		return ok
	}

	onlyWhenFilter1 := func(ctx iris.Context) {
		ctx.Application().Logger().Infof("admin: %s", ctx.Params())
		ctx.Next()
	}

	onlyWhenFilter2 := func(ctx iris.Context) {
		// You can always use the per-request storage
		// to perform actions like this ofc.
		//
		// this handler: ctx.Values().Set("is_admin", true)
		// next handler: isAdmin := ctx.Values().GetBoolDefault("is_admin", false)
		//
		// but, let's simplify it:
		ctx.HTML("<h1>Hello Admin</h1><br>")
		ctx.Next()
	}

	// HERE:
	// It can be registered anywhere, as a middleware.
	// It will fire the `onlyWhenFilter1` and `onlyWhenFilter2` as middlewares (with ctx.Next())
	// if myFilter pass otherwise it will just continue the handler chain with ctx.Next() by ignoring
	// the `onlyWhenFilter1` and `onlyWhenFilter2`.
	myMiddleware := iris.NewConditionalHandler(myFilter, onlyWhenFilter1, onlyWhenFilter2)

	v1UsersRouter := v1.Party("/users", myMiddleware)
	v1UsersRouter.Get("/", func(ctx iris.Context) {
		ctx.HTML("requested: <b>/api/v1/users</b>")
	})

	// http://localhost:8080/api/v1/users
	// http://localhost:8080/api/v1/users?admin=true
	app.Run(iris.Addr(":8080"))
}
