// Package main an example on how to catch dynamic subdomains - wildcard.
// On the first example (subdomains_1) we saw how to create routes for static subdomains, subdomains you know that you will have.
// Here we will see an example how to catch unknown subdomains, dynamic subdomains, like username.mydomain.com:8080.
package main

import (
	"github.com/kataras/iris"
)

// register a dynamic-wildcard subdomain to your server machine(dns/...) first, check ./hosts if you use windows.
// run this file and try to redirect: http://username1.mydomain.com:8080/ , http://username2.mydomain.com:8080/ , http://username1.mydomain.com/something, http://username1.mydomain.com/something/sadsadsa

func main() {
	app := iris.New()

	/* Keep note that you can use both type of subdomains (named and wildcard(*.) )
	   admin.mydomain.com,  and for other the Party(*.) but this is not this example's purpose

	admin := app.Party("admin.")
	{
		// admin.mydomain.com
		admin.Get("/", func(ctx iris.Context) {
			ctx.Writef("INDEX FROM admin.mydomain.com")
		})
		// admin.mydomain.com/hey
		admin.Get("/hey", func(ctx iris.Context) {
			ctx.Writef("HEY FROM admin.mydomain.com/hey")
		})
		// admin.mydomain.com/hey2
		admin.Get("/hey2", func(ctx iris.Context) {
			ctx.Writef("HEY SECOND FROM admin.mydomain.com/hey")
		})
	}*/

	// no order, you can register subdomains at the end also.
	dynamicSubdomains := app.Party("*.")
	{
		dynamicSubdomains.Get("/", dynamicSubdomainHandler)

		dynamicSubdomains.Get("/something", dynamicSubdomainHandler)

		dynamicSubdomains.Get("/something/{paramfirst}", dynamicSubdomainHandlerWithParam)
	}

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hello from mydomain.com path: %s", ctx.Path())
	})

	app.Get("/hello", func(ctx iris.Context) {
		ctx.Writef("Hello from mydomain.com path: %s", ctx.Path())
	})

	// http://mydomain.com:8080
	// http://username1.mydomain.com:8080
	// http://username2.mydomain.com:8080/something
	// http://username3.mydomain.com:8080/something/yourname
	app.Run(iris.Addr("mydomain.com:8080")) // for beginners: look ../hosts file
}

func dynamicSubdomainHandler(ctx iris.Context) {
	username := ctx.Subdomain()
	ctx.Writef("Hello from dynamic subdomain path: %s, here you can handle the route for dynamic subdomains, handle the user: %s", ctx.Path(), username)
	// if  http://username4.mydomain.com:8080/ prints:
	// Hello from dynamic subdomain path: /, here you can handle the route for dynamic subdomains, handle the user: username4
}

func dynamicSubdomainHandlerWithParam(ctx iris.Context) {
	username := ctx.Subdomain()
	ctx.Writef("Hello from dynamic subdomain path: %s, here you can handle the route for dynamic subdomains, handle the user: %s", ctx.Path(), username)
	ctx.Writef("The paramfirst is: %s", ctx.Params().Get("paramfirst"))
}
