package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/versioning"
)

func main() {
	app := iris.New()

	app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.WriteString(`Root not found handler.
        This will be applied everywhere except the /api/* requests.`)
	})

	api := app.Party("/api")
	// Optional, set version aliases (literal strings).
	// We use `UseRouter` instead of `Use`
	// to handle HTTP errors per version, but it's up to you.
	api.UseRouter(versioning.Aliases(versioning.AliasMap{
		// If no version provided by the client, default it to the "1.0.0".
		versioning.Empty: "1.0.0",
		// If a "latest" version is provided by the client,
		// set the version to be compared to "3.0.0".
		"latest": "3.0.0",
	}))

	/*
	   A version is extracted through the versioning.GetVersion function,
	   request headers:
	   - Accept-Version: 1.0.0
	   - Accept: application/json; version=1.0.0
	   You can customize it by setting a version based on the request context:
	   api.Use(func(ctx *context.Context) {
	       if version := ctx.URLParam("version"); version != "" {
	           versioning.SetVersion(ctx, version)
	       }

	       ctx.Next()
	   })
	   OR: api.Use(versioning.FromQuery("version", ""))
	*/

	// |----------------|
	// | The fun begins |
	// |----------------|

	// Create a new Group, which is a compatible Party,
	// based on version constraints.
	v1 := versioning.NewGroup(api, ">=1.0.0 <2.0.0")
	// To mark an API version as deprecated use the Deprecated method.
	// v1.Deprecated(versioning.DefaultDeprecationOptions)

	// Optionally, set custom view engine and path
	// for templates based on the version.
	v1.RegisterView(iris.HTML("./v1", ".html"))

	// Optionally, set custom error handler(s) based on the version.
	// Keep in mind that if you do this, you will
	// have to register error handlers
	// for the rest of the parties as well.
	v1.OnErrorCode(iris.StatusNotFound, testError("v1"))

	// Register resources based on the version.
	v1.Get("/", testHandler("v1"))
	v1.Get("/render", testView)

	// Do the same for version 2 and version 3,
	// for the sake of the example.
	v2 := versioning.NewGroup(api, ">=2.0.0 <3.0.0")
	v2.RegisterView(iris.HTML("./v2", ".html"))
	v2.OnErrorCode(iris.StatusNotFound, testError("v2"))
	v2.Get("/", testHandler("v2"))
	v2.Get("/render", testView)

	v3 := versioning.NewGroup(api, ">=3.0.0 <4.0.0")
	v3.RegisterView(iris.HTML("./v3", ".html"))
	v3.OnErrorCode(iris.StatusNotFound, testError("v3"))
	v3.Get("/", testHandler("v3"))
	v3.Get("/render", testView)

	app.Listen(":8080")
}

func testHandler(v string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"version": v,
			"message": "Hello, world!",
		})
	}
}

func testError(v string) iris.Handler {
	return func(ctx iris.Context) {
		ctx.Writef("not found: %s", v)
	}
}

func testView(ctx iris.Context) {
	if err := ctx.View("index.html"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
