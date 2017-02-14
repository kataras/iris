## Package information

Gorillamux is a plugin for Iris which overrides the Iris' default router with the [Gorilla Mux](https://github.com/gorilla/mux)
which enables path matching using custom `regexp` ( thing that the Iris' default router doesn't supports for performance reasons).

All these without need to change any of your existing Iris code. All features are supported.

## Install

```sh
$ go get -u github.com/iris-contrib/plugin/gorillamux
```


```go
iris.Plugins.Add(gorillamux.New())
```

## [Example](https://github.com/iris-contrib/examples/tree/master/plugin_gorillamux)


```go
package main

import (
	"github.com/iris-contrib/plugin/gorillamux"
	"github.com/kataras/iris"
)

func main() {
	iris.Plugins.Add(gorillamux.New())

	// CUSTOM HTTP ERRORS ARE SUPPORTED
	// NOTE: Gorilla mux allows customization only on StatusNotFound(404)
	// Iris allows for everything, so you can register any other custom http error
	// but you have to call it manually from ctx.EmitError(status_code) // 500 for example
	// this will work because it's StatusNotFound:
	iris.Default.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.HTML(iris.StatusNotFound, "<h1> CUSTOM NOT FOUND ERROR PAGE </h1>")
	})

	// GLOBAL/PARTY MIDDLEWARE ARE SUPPORTED
	iris.Default.UseFunc(func(ctx *iris.Context) {
		println("Request: " + ctx.Path())
		ctx.Next()
	})

	// http://mydomain.com
	iris.Default.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from index")
	})

  /// -------------------------------------- IMPORTANT --------------------------------------
	/// GORILLA MUX PARAMETERS(regexp) ARE SUPPORTED
	/// http://mydomain.com/api/users/42
  /// ---------------------------------------------------------------------------------------
	iris.Default.Get("/api/users/{userid:[0-9]+}", func(ctx *iris.Context) {
		ctx.Writef("User with id: %s", ctx.Param("userid"))
	})

	// PER-ROUTE MIDDLEWARE ARE SUPPORTED
	// http://mydomain.com/other
	iris.Default.Get("/other", func(ctx *iris.Context) {
		ctx.Writef("/other 1 middleware \n")
		ctx.Next()
	}, func(ctx *iris.Context) {
		ctx.HTML(iris.StatusOK, "<b>Hello from /other</b>")
	})

	// SUBDOMAINS ARE SUPPORTED
	// http://admin.mydomain.com
	iris.Default.Party("admin.").Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from admin. subdomain!")
	})

	// WILDCARD SUBDOMAINS ARE SUPPORTED
	// http://api.mydomain.com/hi
	// http://admin.mydomain.com/hi
	// http://x.mydomain.com/hi
	// [depends on your host configuration,
	// you will see an example(win) outside of this folder].
	iris.Default.Party("*.").Get("/hi", func(ctx *iris.Context) {
		ctx.Writef("Hello from wildcard subdomain: %s", ctx.Subdomain())
	})

	// DOMAIN NAMING IS SUPPORTED
	iris.Default.Listen("mydomain.com")
	// iris.Default.Listen(":80")
}

/* HOSTS FILE LINES TO RUN THIS EXAMPLE:

127.0.0.1		mydomain.com
127.0.0.1		admin.mydomain.com
127.0.0.1		api.mydomain.com
127.0.0.1		x.mydomain.com

*/


```

> Custom domain is totally optionally, you can still use `iris.Default.Listen(":8080")` of course.
