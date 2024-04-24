package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/rewrite"
)

func main() {
	app := newApp()

	// http://mydomain.com             -> http://www.mydomain.com
	// http://mydomain.com/user        -> http://www.mydomain.com/user
	// http://mydomain.com/user/login  -> http://www.mydomain.com/user/login
	app.Listen(":80")
}

func newApp() *iris.Application {
	app := iris.New()
	app.Logger().SetLevel("debug")

	static := app.Subdomain("static")
	static.Get("/", staticIndex)

	app.Get("/", index)
	userRouter := app.Party("/user")
	userRouter.Get("/", userGet)
	userRouter.Get("/login", userGetLogin)

	// redirects := rewrite.Load("redirects.yml")
	// ^ see _examples/routing/rewrite example for that.
	//
	// Now let's do that by code.
	rewriteEngine, _ := rewrite.New(rewrite.Options{
		PrimarySubdomain: "www",
	})
	// Enable this line for debugging:
	// rewriteEngine.SetLogger(app.Logger())
	app.WrapRouter(rewriteEngine.Rewrite)

	return app
}

func staticIndex(ctx iris.Context) {
	ctx.Writef("This is the static.mydomain.com index.")
}

func index(ctx iris.Context) {
	ctx.Writef("This is the www.mydomain.com index.")
}

func userGet(ctx iris.Context) {
	// Also, ctx.Subdomain(), ctx.SubdomainFull(), ctx.Host() and ctx.Path()
	// can be helpful when working with subdomains.
	ctx.Writef("This is the www.mydomain.com/user endpoint.")
}

func userGetLogin(ctx iris.Context) {
	ctx.Writef("This is the www.mydomain.com/user/login endpoint.")
}
