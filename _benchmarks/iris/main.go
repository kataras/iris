package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()
	// These handlers are serving the same routes as
	// `ValuesController`s of netcore-mvc and iris-mvc applications do.
	app.Get("/api/values/{id}", getHandler)
	app.Put("/api/values/{id}", putHandler)
	app.Delete("/api/values/{id}", delHandler)
	app.Run(iris.Addr(":5000"))
}

// getHandler handles "GET" requests to "api/values/{id}".
func getHandler(ctx context.Context) {
	// id,_ := vc.Params.GetInt("id")
	ctx.WriteString("value")
}

// putHandler handles "PUT" requests to "api/values/{id}".
func putHandler(ctx context.Context) {}

// delHandler handles "DELETE" requests to "api/values/{id}".
func delHandler(ctx context.Context) {}
