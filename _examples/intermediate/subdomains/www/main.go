package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func newApp() *iris.Application {
	app := iris.New()

	app.Get("/", info)
	app.Get("/about", info)
	app.Get("/contact", info)

	usersAPI := app.Party("/api/users")
	{
		usersAPI.Get("/", info)
		usersAPI.Get("/{id:int}", info)

		usersAPI.Post("/", info)

		usersAPI.Put("/{id:int}", info)
	}

	www := app.Party("www.")
	{
		// get all routes that are registered so far, including all "Parties":
		currentRoutes := app.GetRoutes()
		// register them to the www subdomain/vhost as well:
		for _, r := range currentRoutes {
			if _, err := www.Handle(r.Method, r.Path, r.Handlers...); err != nil {
				app.Log("%s for www. failed: %v", r.Path, err)
			}
		}
	}

	return app
}

func main() {
	app := newApp()
	// http://iris-go.com
	// http://iris-go.com/about
	// http://iris-go.com/contact
	// http://iris-go.com/api/users
	// http://iris-go.com/api/users/42

	// http://www.iris-go.com
	// http://www.iris-go.com/about
	// http://www.iris-go.com/contact
	// http://www.iris-go.com/api/users
	// http://www.iris-go.com/api/users/42
	if err := app.Run(iris.Addr("iris-go.com:80")); err != nil {
		panic(err)
	}
}

func info(ctx context.Context) {
	method := ctx.Method()
	subdomain := ctx.Subdomain()
	path := ctx.Path()

	ctx.Writef("\nInfo\n\n")
	ctx.Writef("Method: %s\nSubdomain: %s\nPath: %s", method, subdomain, path)
}
