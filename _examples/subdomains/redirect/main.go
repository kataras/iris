// Package main shows how to register a simple 'www' subdomain,
// using the `app.WWW` method, which will register a router wrapper which will
// redirect all 'mydomain.com' requests to 'www.mydomain.com'.
// Check the 'hosts' file to see how to test the 'mydomain.com' on your local machine.
package main

import "github.com/kataras/iris"

const addr = "mydomain.com:80"

func main() {
	app := newApp()

	// http(s)://mydomain.com, will be redirect to http(s)://www.mydomain.com.
	// The `www` variable is the `app.Subdomain("www")`.
	//
	// app.WWW() wraps the router so it can redirect all incoming requests
	// that comes from 'http(s)://mydomain.com/%path%' (www is missing)
	// to `http(s)://www.mydomain.com/%path%`.
	//
	// Try:
	// http://mydomain.com             -> http://www.mydomain.com
	// http://mydomain.com/users       -> http://www.mydomain.com/users
	// http://mydomain.com/users/login -> http://www.mydomain.com/users/login
	app.Run(iris.Addr(addr))
}

func newApp() *iris.Application {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("This will never be executed.")
	})

	www := app.Subdomain("www") // <- same as app.Party("www.")
	www.Get("/", index)

	// www is an `iris.Party`, use it like you already know, like grouping routes.
	www.PartyFunc("/users", func(p iris.Party) { // <- same as www.Party("/users").Get(...)
		p.Get("/", usersIndex)
		p.Get("/login", getLogin)
	})

	// redirects mydomain.com/%anypath% to www.mydomain.com/%anypath%.
	// First argument is the 'from' and second is the 'to/target'.
	app.SubdomainRedirect(app, www)

	// SubdomainRedirect works for multi-level subdomains as well:
	// subsub := www.Subdomain("subsub") // subsub.www.mydomain.com
	// subsub.Get("/", func(ctx iris.Context) { ctx.Writef("subdomain is: " + ctx.Subdomain()) })
	// app.SubdomainRedirect(subsub, www)
	//
	// If you need to redirect any subdomain to 'www' then:
	// app.SubdomainRedirect(app.WildcardSubdomain(), www)
	// If you need to redirect from a subdomain to the root domain then:
	// app.SubdomainRedirect(app.Subdomain("mysubdomain"), app)
	//
	// Note that app.Party("mysubdomain.") and app.Subdomain("mysubdomain")
	// is the same exactly thing, the difference is that the second can omit the last dot('.').

	return app
}

func index(ctx iris.Context) {
	ctx.Writef("This is the www.mydomain.com endpoint.")
}

func usersIndex(ctx iris.Context) {
	ctx.Writef("This is the www.mydomain.com/users endpoint.")
}

func getLogin(ctx iris.Context) {
	ctx.Writef("This is the www.mydomain.com/users/login endpoint.")
}
