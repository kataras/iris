package main

import "github.com/kataras/iris/v12"

func main() {
	newApp().Listen("mydomain.com:80", iris.WithLogLevel("debug"))
}

func newApp() *iris.Application {
	app := iris.New()

	// Create the "test.mydomain.com" subdomain.
	test := app.Subdomain("test")
	// Register views for the test subdomain.
	test.RegisterView(iris.HTML("./views", ".html").
		Layout("layouts/test.layout.html"))

	// Optionally, to minify the HTML5 error response.
	// Note that minification might be slower, caching is advised.
	// test.UseError(iris.Minify)
	// or pass it to OnErrorCode:
	// Register error code 404 handler.
	test.OnErrorCode(iris.StatusNotFound, iris.Minify, handleNotFoundTestSubdomain)

	test.Get("/", testIndex)

	return app
}

func handleNotFoundTestSubdomain(ctx iris.Context) {
	if err := ctx.View("error.html", iris.Map{
		"ErrorCode": ctx.GetStatusCode(),
	}); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func testIndex(ctx iris.Context) {
	ctx.Writef("%s index page\n", ctx.Subdomain())
}
