// Package main simply shows how you can getting started with Iris and Vue Router.
// Read more at: https://router.vuejs.org/guide/#html
package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.HandleDir("/", "./frontend")

	app.Listen(":8080")
}

/* For those who want to use HTML template as the index page
   and serve static files in the root request path
   and use vue router as the main router of the entire application,
   please follow the below code example:

func fullVueRouter() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))
	app.OnAnyErrorCode(index)
	app.HandleDir("/", "./frontend")
	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	if err := ctx.View("index.html"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
*/
