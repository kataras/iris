package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// GET: http://localhost:8080
	app.Get("/", info)

	// GET: http://localhost:8080/profile/anyusername
	app.Get("/profile/{username:string}", info)
	// GET: http://localhost:8080/profile/anyusername/backups/any/number/of/paths/here
	app.Get("/profile/{username:string}/backups/{filepath:path}", info)
	// Favicon

	// GET: http://localhost:8080/favicon.ico
	app.Favicon("./public/images/favicon.ico")

	// Static assets

	// GET: http://localhost:8080/assets/css/bootstrap.min.css
	//	    maps to ./public/assets/css/bootstrap.min.css file at system location.
	// GET: http://localhost:8080/assets/js/react.min.js
	//      maps to ./public/assets/js/react.min.js file at system location.
	app.StaticWeb("/assets", "./public/assets")

	/* OR

	// GET: http://localhost:8080/js/react.min.js
	// 		maps to ./public/assets/js/react.min.js file at system location.
	app.StaticWeb("/js", "./public/assets/js")

	// GET: http://localhost:8080/css/bootstrap.min.css
	// 		maps to ./public/assets/css/bootstrap.min.css file at system location.
	app.StaticWeb("/css", "./public/assets/css")

	*/

	// Grouping

	usersRoutes := app.Party("/users")
	// GET: http://localhost:8080/users/help
	usersRoutes.Get("/help", func(ctx iris.Context) {
		ctx.Writef("GET / -- fetch all users\n")
		ctx.Writef("GET /$ID -- fetch a user by id\n")
		ctx.Writef("POST / -- create new user\n")
		ctx.Writef("PUT /$ID -- update an existing user\n")
		ctx.Writef("DELETE /$ID -- delete an existing user\n")
	})

	// GET: http://localhost:8080/users
	usersRoutes.Get("/", func(ctx iris.Context) {
		ctx.Writef("get all users")
	})

	// GET: http://localhost:8080/users/42
	// **/users/42 and /users/help works after iris version 7.0.5**
	usersRoutes.Get("/{id:int}", func(ctx iris.Context) {
		id, _ := ctx.Params().GetInt("id")
		ctx.Writef("get user by id: %d", id)
	})

	// POST: http://localhost:8080/users
	usersRoutes.Post("/", func(ctx iris.Context) {
		username, password := ctx.PostValue("username"), ctx.PostValue("password")
		ctx.Writef("create user for username= %s and password= %s", username, password)
	})

	// PUT: http://localhost:8080/users
	usersRoutes.Put("/{id:int}", func(ctx iris.Context) {
		id, _ := ctx.Params().GetInt("id") // or .Get to get its string represatantion.
		username := ctx.PostValue("username")
		ctx.Writef("update user for id= %d and new username= %s", id, username)
	})

	// DELETE: http://localhost:8080/users/42
	usersRoutes.Delete("/{id:int}", func(ctx iris.Context) {
		id, _ := ctx.Params().GetInt("id")
		ctx.Writef("delete user by id: %d", id)
	})

	// Subdomains, depends on the host, you have to edit the hosts or nginx/caddy's configuration if you use them.
	//
	// See more subdomains examples at _examples/subdomains folder.
	adminRoutes := app.Party("admin.")

	// GET: http://admin.localhost:8080
	adminRoutes.Get("/", info)
	// GET: http://admin.localhost:8080/settings
	adminRoutes.Get("/settings", info)

	// Wildcard/dynamic subdomain
	dynamicSubdomainRoutes := app.Party("*.")

	// GET: http://any_thing_here.localhost:8080
	dynamicSubdomainRoutes.Get("/", info)

	app.Delete("/something", func(ctx iris.Context) {
		name := ctx.URLParam("name")
		ctx.Writef(name)
	})

	// GET: http://localhost:8080/
	// GET: http://localhost:8080/profile/anyusername
	// GET: http://localhost:8080/profile/anyusername/backups/any/number/of/paths/here

	// GET: http://localhost:8080/users/help
	// GET: http://localhost:8080/users
	// GET: http://localhost:8080/users/42
	// POST: http://localhost:8080/users
	// PUT: http://localhost:8080/users
	// DELETE: http://localhost:8080/users/42
	// DELETE: http://localhost:8080/something?name=iris

	// GET: http://admin.localhost:8080
	// GET: http://admin.localhost:8080/settings
	// GET: http://any_thing_here.localhost:8080
	app.Run(iris.Addr(":8080"))
}

func info(ctx iris.Context) {
	method := ctx.Method()       // the http method requested a server's resource.
	subdomain := ctx.Subdomain() // the subdomain, if any.

	// the request path (without scheme and host).
	path := ctx.Path()
	// how to get all parameters, if we don't know
	// the names:
	paramsLen := ctx.Params().Len()

	ctx.Params().Visit(func(name string, value string) {
		ctx.Writef("%s = %s\n", name, value)
	})
	ctx.Writef("\nInfo\n\n")
	ctx.Writef("Method: %s\nSubdomain: %s\nPath: %s\nParameters length: %d", method, subdomain, path, paramsLen)
}
