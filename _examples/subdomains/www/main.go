package main

import (
	"github.com/kataras/iris"
)

func newApp() *iris.Application {
	app := iris.New()

	app.Get("/", info)
	app.Get("/about", info)
	app.Get("/contact", info)

	app.PartyFunc("/api/users", func(r iris.Party) {
		r.Get("/", info)
		r.Get("/{id:uint64}", info)

		r.Post("/", info)

		r.Put("/{id:uint64}", info)
	}) /* <- same as:
	 usersAPI := app.Party("/api/users")
	 {  // those brackets are just syntactic-sugar things.
		// This method is rarely used but you can make use of it when you want
	    // scoped variables to that code block only.
		usersAPI.Get/Post...
	 }
	 usersAPI.Get/Post...
	*/

	www := app.Party("www.")
	{
		// Just to show how you can get all routes and copy them to another
		// party or subdomain:
		// Get all routes that are registered so far, including all "Parties" and subdomains:
		currentRoutes := app.GetRoutes()
		// Register them to the www subdomain/vhost as well:
		for _, r := range currentRoutes {
			www.Handle(r.Method, r.Tmpl().Src, r.Handlers...)
		}

		// http://www.mydomain.com/hi
		www.Get("/hi", func(ctx iris.Context) {
			ctx.Writef("hi from www.mydomain.com")
		})
	}
	// See also the "subdomains/redirect" to register redirect router wrappers between subdomains,
	// i.e mydomain.com to www.mydomain.com (like facebook does for SEO reasons(;)).

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
