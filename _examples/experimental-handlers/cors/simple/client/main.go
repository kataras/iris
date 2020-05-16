package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("index.html")
	})

	// Read index.html comments,
	// adn then start and navigate to http://localhost:8080.
	app.Listen(":8080")
}
