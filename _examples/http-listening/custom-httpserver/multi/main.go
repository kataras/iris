package main

import (
	"net/http"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hello from the server")
	})

	app.Get("/mypath", func(ctx iris.Context) {
		ctx.Writef("Hello from %s", ctx.Path())
	})

	// Note: It's not needed if the first action is "go app.Run".
	if err := app.Build(); err != nil {
		panic(err)
	}

	// start a secondary server listening on localhost:9090.
	// use "go" keyword for Listen functions if you need to use more than one server at the same app.
	//
	// http://localhost:9090/
	// http://localhost:9090/mypath
	srv1 := &http.Server{Addr: ":9090", Handler: app}
	go srv1.ListenAndServe()
	println("Start a server listening on http://localhost:9090")

	// start a "second-secondary" server listening on localhost:5050.
	//
	// http://localhost:5050/
	// http://localhost:5050/mypath
	srv2 := &http.Server{Addr: ":5050", Handler: app}
	go srv2.ListenAndServe()
	println("Start a server listening on http://localhost:5050")

	// Note: app.Run is totally optional, we have already built the app with app.Build,
	// you can just make a new http.Server instead.
	// http://localhost:8080/
	// http://localhost:8080/mypath
	app.Run(iris.Addr(":8080")) // Block here.
}
