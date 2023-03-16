package main

import (
	"github.com/kataras/iris/v12"
)

func newApp() *iris.Application {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// registers a custom handler for 404 not found http (error) status code,
	// fires when route not found or manually by ctx.StatusCode(iris.StatusNotFound).
	app.OnErrorCode(iris.StatusNotFound, notFoundHandler)

	// GET -> HTTP Method
	// / -> Path
	// func(ctx iris.Context) -> The route's handler.
	//
	// Third receiver should contains the route's handler(s), they are executed by order.
	app.Handle("GET", "/", func(ctx iris.Context) {
		ctx.HTML("Hello from " + ctx.Path()) // Hello from /
	})

	app.Get("/home", func(ctx iris.Context) {
		ctx.Writef(`Same as app.Handle("GET", "/", [...])`)
	})

	// Different path parameters types in the same path.
	// Note that: fallback should registered first e.g. {path} {string},
	// because the handler on this case is executing from last to top.
	app.Get("/u/{p:path}", func(ctx iris.Context) {
		ctx.Writef(":string, :int, :uint, :alphabetical and :path in the same path pattern.")
	})

	app.Get("/u/{username:string}", func(ctx iris.Context) {
		ctx.Writef("before username (string), current route name: %s\n", ctx.RouteName())
		ctx.Next()
	}, func(ctx iris.Context) {
		ctx.Writef("username (string): %s", ctx.Params().Get("username"))
	})

	app.Get("/u/{firstname:alphabetical}", func(ctx iris.Context) {
		ctx.Writef("before firstname (alphabetical), current route name: %s\n", ctx.RouteName())
		ctx.Next()
	}, func(ctx iris.Context) {
		ctx.Writef("firstname (alphabetical): %s", ctx.Params().Get("firstname"))
	})

	app.Get("/u/{id:int}", func(ctx iris.Context) {
		ctx.Writef("before id (int), current route name: %s\n", ctx.RouteName())
		ctx.Next()
	}, func(ctx iris.Context) {
		ctx.Writef("id (int): %d", ctx.Params().GetIntDefault("id", 0))
	})

	app.Get("/u/{uid:uint}", func(ctx iris.Context) {
		ctx.Writef("before uid (uint), current route name: %s\n", ctx.RouteName())
		ctx.Next()
	}, func(ctx iris.Context) {
		ctx.Writef("uid (uint): %d", ctx.Params().GetUintDefault("uid", 0))
	})

	/*
		/u/some/path/here maps to :path
		/u/abcd maps to :alphabetical (if :alphabetical registered otherwise :string)
		/u/42 maps to :uint (if :uint registered otherwise :int)
		/u/-1 maps to :int (if :int registered otherwise :string)
		/u/abcd123 maps to :string
	*/

	// Pssst, don't forget dynamic-path example for more "magic"!
	app.Get("/api/users/{userid:uint64 min(1)}", func(ctx iris.Context) {
		userID, err := ctx.Params().GetUint64("userid")
		if err != nil {
			ctx.Writef("error while trying to parse userid parameter," +
				"this will never happen if :uint64 is being used because if it's not a valid uint64 it will fire Not Found automatically.")
			ctx.StatusCode(iris.StatusBadRequest)
			return
		}

		ctx.JSON(map[string]interface{}{
			// you can pass any custom structured go value of course.
			"user_id": userID,
		})
	})
	// app.Post("/", func(ctx iris.Context){}) -> for POST http method.
	// app.Put("/", func(ctx iris.Context){})-> for "PUT" http method.
	// app.Delete("/", func(ctx iris.Context){})-> for "DELETE" http method.
	// app.Options("/", func(ctx iris.Context){})-> for "OPTIONS" http method.
	// app.Trace("/", func(ctx iris.Context){})-> for "TRACE" http method.
	// app.Head("/", func(ctx iris.Context){})-> for "HEAD" http method.
	// app.Connect("/", func(ctx iris.Context){})-> for "CONNECT" http method.
	// app.Patch("/", func(ctx iris.Context){})-> for "PATCH" http method.
	// app.Any("/", func(ctx iris.Context){}) for all http methods.

	// More than one route can contain the same path with a different http mapped method.
	// You can catch any route creation errors with:
	// route, err := app.Get(...)
	// set a name to a route: route.Name = "myroute"

	// You can also group routes by path prefix, sharing middleware(s) and done handlers.

	adminRoutes := app.Party("/admin", adminMiddleware)

	adminRoutes.Done(func(ctx iris.Context) { // executes always last if ctx.Next()
		ctx.Application().Logger().Infof("response sent to " + ctx.Path())
	})
	// adminRoutes.Layout("/views/layouts/admin.html") // set a view layout for these routes, see more at view examples.

	// GET: http://localhost:8080/admin
	adminRoutes.Get("/", func(ctx iris.Context) {
		// [...]
		ctx.StatusCode(iris.StatusOK) // default is 200 == iris.StatusOK
		ctx.HTML("<h1>Hello from admin/</h1>")

		ctx.Next() // in order to execute the party's "Done" Handler(s)
	})

	// GET: http://localhost:8080/admin/login
	adminRoutes.Get("/login", func(ctx iris.Context) {
		// [...]
	})
	// POST: http://localhost:8080/admin/login
	adminRoutes.Post("/login", func(ctx iris.Context) {
		// [...]
	})

	// subdomains, easier than ever, should add localhost or 127.0.0.1 into your hosts file,
	// etc/hosts on unix or C:/windows/system32/drivers/etc/hosts on windows.
	v1 := app.Party("v1.")
	{ // braces are optional, it's just type of style, to group the routes visually.

		// http://v1.localhost:8080
		// Note: for versioning-specific features checkout the _examples/routing/versioning instead.
		v1.Get("/", func(ctx iris.Context) {
			ctx.HTML(`Version 1 API. go to <a href="/api/users">/api/users</a>`)
		})

		usersAPI := v1.Party("/api/users")
		{
			// http://v1.localhost:8080/api/users
			usersAPI.Get("/", func(ctx iris.Context) {
				ctx.Writef("All users")
			})
			// http://v1.localhost:8080/api/users/42
			usersAPI.Get("/{userid:int}", func(ctx iris.Context) {
				ctx.Writef("user with id: %d", ctx.Params().GetIntDefault("userid", 0))
			})
		}
	}

	// wildcard subdomains.
	wildcardSubdomain := app.WildcardSubdomain()
	{
		wildcardSubdomain.Get("/", func(ctx iris.Context) {
			ctx.Writef("Subdomain can be anything, now you're here from: %s", ctx.Subdomain())
		})
	}

	return app
}

func main() {
	app := newApp()

	// http://localhost:8080
	// http://localhost:8080/home
	// http://localhost:8080/api/users/42
	// http://localhost:8080/admin
	// http://localhost:8080/admin/login
	//
	// http://localhost:8080/api/users/0
	// http://localhost:8080/api/users/blabla
	// http://localhost:8080/wontfound
	//
	// http://localhost:8080/u/abcd
	// http://localhost:8080/u/42
	// http://localhost:8080/u/-1
	// http://localhost:8080/u/abcd123
	// http://localhost:8080/u/some/path/here
	//
	// if hosts edited:
	//  http://v1.localhost:8080
	//  http://v1.localhost:8080/api/users
	//  http://v1.localhost:8080/api/users/42
	//  http://anything.localhost:8080
	app.Listen(":8080")
}

func adminMiddleware(ctx iris.Context) {
	// [...]
	ctx.Next() // to move to the next handler, or don't that if you have any auth logic.
}

func notFoundHandler(ctx iris.Context) {
	ctx.HTML("Custom route for 404 not found http code, here you can render a view, html, json <b>any valid response</b>.")
}

// Notes:
// A path parameter name should contain only alphabetical letters, symbols, containing '_' and numbers are NOT allowed.
// If route failed to be registered, the app will panic without any warnings
// if you didn't catch the second return value(error) on .Handle/.Get....

// See "file-server/single-page-application" to see how another feature, "WrapRouter", works.
