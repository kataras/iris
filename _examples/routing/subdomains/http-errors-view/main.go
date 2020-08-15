package main

import "github.com/kataras/iris/v12"

func main() {
	newApp().Listen("mydomain.com:80", iris.WithLogLevel("debug"))
}

func newApp() *iris.Application {
	app := iris.New()

	test := app.Subdomain("test")
	test.RegisterView(iris.HTML("./views", ".html").
		Layout("layouts/test.layout.html"))

	test.OnErrorCode(iris.StatusNotFound, handleNotFoundTestSubdomain)
	test.Get("/", testIndex)

	return app
}

func handleNotFoundTestSubdomain(ctx iris.Context) {
	ctx.View("error.html", iris.Map{
		"ErrorCode": ctx.GetStatusCode(),
	})
}

func testIndex(ctx iris.Context) {
	ctx.Writef("%s index page\n", ctx.Subdomain())
}
