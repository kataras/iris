package main

import "github.com/kataras/iris/v12"

func newApp() *iris.Application {
	app := iris.New()
	// This will create a new "www" subdomain
	// and redirect root-level domain requests
	// to that one:
	www := app.WWW()
	www.Get("/", info)
	www.Get("/about", info)
	www.Get("/contact", info)

	www.PartyFunc("/api/users", func(r iris.Party) {
		r.Get("/", info)
		r.Get("/{id:uint64}", info)

		r.Post("/", info)

		r.Put("/{id:uint64}", info)
	})

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
	// http://www.mydomain.com/hi
	// http://www.mydomain.com/about
	// http://www.mydomain.com/contact
	// http://www.mydomain.com/api/users
	// http://www.mydomain.com/api/users/42
	app.Listen("mydomain.com:80")
}

func info(ctx iris.Context) {
	method := ctx.Method()
	subdomain := ctx.Subdomain()
	path := ctx.Path()

	ctx.Writef("\nInfo\n\n")
	ctx.Writef("Method: %s\nSubdomain: %s\nPath: %s", method, subdomain, path)
}
