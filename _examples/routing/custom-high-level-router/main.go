package main

import (
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

/* A Router should contain all three of the following methods:
   - HandleRequest should handle the request based on the Context.
	  HandleRequest(ctx context.Context)
   - Build should builds the handler, it's being called on router's BuildRouter.
	  Build(provider router.RoutesProvider) error
   - RouteExists reports whether a particular route exists.
      RouteExists(ctx context.Context, method, path string) bool

For a more detailed, complete and useful example
you can take a look at the iris' router itself which is located at:
https://github.com/kataras/iris/tree/master/core/router/handler.go
which completes this exact interface, the `router#RequestHandler`.
*/
type customRouter struct {
	// a copy of routes (safer because you will not be able to alter a route on serve-time without a `app.RefreshRouter` call):
	// []router.Route
	// or just expect the whole routes provider:
	provider router.RoutesProvider
}

// HandleRequest a silly example which finds routes based only on the first part of the requested path
// which must be a static one as well, the rest goes to fill the parameters.
func (r *customRouter) HandleRequest(ctx context.Context) {
	path := ctx.Path()
	ctx.Application().Logger().Infof("Requested resource path: %s", path)

	parts := strings.Split(path, "/")[1:]
	staticPath := "/" + parts[0]
	for _, route := range r.provider.GetRoutes() {
		if strings.HasPrefix(route.Path, staticPath) && route.Method == ctx.Method() {
			paramParts := parts[1:]
			for _, paramValue := range paramParts {
				for _, p := range route.Tmpl().Params {
					ctx.Params().Set(p.Name, paramValue)
				}
			}

			ctx.SetCurrentRouteName(route.Name)
			ctx.Do(route.Handlers)
			return
		}
	}

	// if nothing found...
	ctx.StatusCode(iris.StatusNotFound)
}

func (r *customRouter) Build(provider router.RoutesProvider) error {
	for _, route := range provider.GetRoutes() {
		// do any necessary validation or conversations based on your custom logic here
		// but always run the "BuildHandlers" for each registered route.
		route.BuildHandlers()
		// [...] r.routes = append(r.routes, *route)
	}

	r.provider = provider
	return nil
}

func (r *customRouter) RouteExists(ctx context.Context, method, path string) bool {
	// [...]
	return false
}

func main() {
	app := iris.New()

	// In case you are wondering, the parameter types and macros like "{param:string $func()}" still work inside
	// your custom router if you fetch by the Route's Handler
	// because they are middlewares under the hood, so you don't have to implement the logic of handling them manually,
	// though you have to match what requested path is what route and fill the ctx.Params(), this is the work of your custom router.
	app.Get("/hello/{name}", func(ctx context.Context) {
		name := ctx.Params().Get("name")
		ctx.Writef("Hello %s\n", name)
	})

	app.Get("/cs/{num:uint64 min(10) else 400}", func(ctx context.Context) {
		num := ctx.Params().GetUint64Default("num", 0)
		ctx.Writef("num is: %d\n", num)
	})

	// To replace the existing router with a customized one by using the iris/context.Context
	// you have to use the `app.BuildRouter` method before `app.Run` and after the routes registered.
	// You should pass your custom router's instance as the second input arg, which must completes the `router#RequestHandler`
	// interface as shown above.
	//
	// To see how you can build something even more low-level without direct iris' context support (you can do that manually as well)
	// navigate to the "custom-wrapper" example instead.
	myCustomRouter := new(customRouter)
	app.BuildRouter(app.ContextPool, myCustomRouter, app.APIBuilder, true)

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}
