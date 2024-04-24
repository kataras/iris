package main

import "github.com/kataras/iris/v12"

func newApp() *iris.Application {
	app := iris.New()

	app.HandleDir("/", iris.Dir("./public"), iris.DirOptions{
		IndexName: "index.html",
		SPA:       true,
	})

	return app
}

func main() {
	app := newApp()

	// http://localhost:8080
	// http://localhost:8080/about
	// http://localhost:8080/a_notfound
	app.Listen(":8080")
}
