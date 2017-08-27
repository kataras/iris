package main

import (
	"github.com/kataras/iris"
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
		// get all routes that are registered so far, including all "Parties" but subdomains:
		currentRoutes := app.GetRoutes()
		// register them to the www subdomain/vhost as well:
		for _, r := range currentRoutes {
			www.Handle(r.Method, r.Path, r.Handlers...)
		}
	}

	return app
}

func main() {
	app := newApp()
	// http://mydomain.com
	// http://mydomain.com/about
	// http://imydomain.com/contact
	// http://mydomain.com/api/users
	// http://mydomain.com/api/users/42

	// http://www.mydomain.com
	// http://www.mydomain.com/about
	// http://www.mydomain.com/contact
	// http://www.mydomain.com/api/users
	// http://www.mydomain.com/api/users/42
	if err := app.Run(iris.Addr("mydomain.com:80")); err != nil {
		panic(err)
	}
}

func info(ctx iris.Context) {
	method := ctx.Method()
	subdomain := ctx.Subdomain()
	path := ctx.Path()

	ctx.Writef("\nInfo\n\n")
	ctx.Writef("Method: %s\nSubdomain: %s\nPath: %s", method, subdomain, path)
}
