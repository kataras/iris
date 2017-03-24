// Package main register static subdomains, simple as parties, check ./hosts if you use windows
package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	// subdomains works with all available routers, like other features too.
	app.Adapt(httprouter.New())

	// no order, you can register subdomains at the end also.
	admin := app.Party("admin.")
	{
		// admin.mydomain.com
		admin.Get("/", func(c *iris.Context) {
			c.Writef("INDEX FROM admin.mydomain.com")
		})
		// admin.mydomain.com/hey
		admin.Get("/hey", func(c *iris.Context) {
			c.Writef("HEY FROM admin.mydomain.com/hey")
		})
		// admin.mydomain.com/hey2
		admin.Get("/hey2", func(c *iris.Context) {
			c.Writef("HEY SECOND FROM admin.mydomain.com/hey")
		})
	}

	// mydomain.com/
	app.Get("/", func(c *iris.Context) {
		c.Writef("INDEX FROM no-subdomain hey")
	})

	// mydomain.com/hey
	app.Get("/hey", func(c *iris.Context) {
		c.Writef("HEY FROM no-subdomain hey")
	})

	app.Listen("mydomain.com:80") // for beginners: look ../hosts file
}
