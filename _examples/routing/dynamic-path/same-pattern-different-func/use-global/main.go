package main // #1552

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	app.UseGlobal(middleware("first"))
	app.UseGlobal(middleware("second"))
	app.DoneGlobal(onDone)

	app.Get("/{name prefix(one)}", handler("first route"))
	app.Get("/{name prefix(two)}", handler("second route"))
	app.Get("/{name prefix(three)}", handler("third route"))

	return app
}

func middleware(str string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.Writef("Called %s middleware\n", str)
		ctx.Next()
	}
}

func handler(str string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.Writef("%s\n", str)
		ctx.Next() // or ignroe that and use app.SetRegisterRules.
	}
}

func onDone(ctx iris.Context) {
	ctx.Writef("Called done: %s", ctx.Params().Get("name"))
}
