package main

import (
	"net/http"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/handlerconv"
)

func main() {
	app := iris.New()
	irisMiddleware := handlerconv.FromStdWithNext(negronilikeTestMiddleware)
	app.Use(irisMiddleware)

	// Method GET: http://localhost:8080/
	app.Get("/", func(ctx context.Context) {
		ctx.HTML("<h1> Home </h1>")
		// this will print an error,
		// this route's handler will never be executed because the middleware's criteria not passed.
	})

	// Method GET: http://localhost:8080/ok
	app.Get("/ok", func(ctx context.Context) {
		ctx.Writef("Hello world!")
		// this will print "OK. Hello world!".
	})

	// http://localhost:8080
	// http://localhost:8080/ok
	app.Run(iris.Addr(":8080"))
}

func negronilikeTestMiddleware(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.URL.Path == "/ok" && r.Method == "GET" {
		w.Write([]byte("OK. "))
		next(w, r) // go to the next route's handler
		return
	}
	// else print an error and do not forward to the route's handler.
	w.WriteHeader(iris.StatusBadRequest)
	w.Write([]byte("Bad request"))
}
