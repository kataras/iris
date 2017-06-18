package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/core/router"
)

func main() {
	app := iris.New()
	// need for manually reverse routing when needed outside of view engine.
	// you normally don't need it because of the {{ urlpath "routename" "path" "values" "here"}}
	rv := router.NewRoutePathReverser(app)

	myroute, _ := app.Get("/anything/{anythingparameter:path}", func(ctx context.Context) {
		paramValue := ctx.Params().Get("anythingparameter")
		ctx.Writef("The path after /anything is: %s", paramValue)
	})

	// useful for links, altough iris' view engine has the {{ urlpath "routename" "path values"}} already.
	app.Get("/reverse_myroute", func(ctx context.Context) {
		myrouteRequestPath := rv.Path(myroute.Name, "any/path")
		ctx.HTML("Should be <b>/anything/any/path</b>: " + myrouteRequestPath)
	})

	// execute a route, similar to redirect but without redirect :)
	app.Get("/execute_myroute", func(ctx context.Context) {
		ctx.Exec("GET", "/anything/any/path") // like it was called by the client.
	})

	// http://localhost:8080/reverse_myroute
	// http://localhost:8080/execute_myroute
	// http://localhost:8080/anything/any/path/here
	app.Run(iris.Addr(":8080"))
}
