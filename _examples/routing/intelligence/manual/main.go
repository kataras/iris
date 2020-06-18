package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.OnErrorCode(iris.StatusNotFound, notFound)

	// [register some routes...]
	app.Get("/home", handler)
	app.Get("/news", handler)
	app.Get("/news/politics", handler)
	app.Get("/user/profile", handler)
	app.Get("/user", handler)
	app.Get("/newspaper", handler)
	app.Get("/user/{id}", handler)

	app.Listen(":8080")
}

func notFound(ctx iris.Context) {
	suggestPaths := ctx.FindClosest(3)
	if len(suggestPaths) == 0 {
		ctx.WriteString("404 not found")
		return
	}

	ctx.HTML("Did you mean?<ul>")
	for _, s := range suggestPaths {
		ctx.HTML(`<li><a href="%s">%s</a></li>`, s, s)
	}
	ctx.HTML("</ul>")
}

func handler(ctx iris.Context) {
	ctx.Writef("Path: %s", ctx.Path())
}
