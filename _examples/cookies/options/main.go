package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := newApp()

	// http://localhost:8080/set/name1/value1
	// http://localhost:8080/get/name1
	// http://localhost:8080/remove/name1
	app.Listen(":8080", iris.WithLogLevel("debug"))
}

func newApp() *iris.Application {
	app := iris.New()
	app.Use(withCookieOptions)

	app.Get("/set/{name}/{value}", setCookie)
	app.Get("/get/{name}", getCookie)
	app.Get("/remove/{name}", removeCookie)

	return app
}

func withCookieOptions(ctx iris.Context) {
	// Register cookie options for request-lifecycle.
	// To register per cookie, just add the CookieOption
	// on the last variadic input argument of
	// SetCookie, SetCookieKV, UpsertCookie, RemoveCookie
	// and GetCookie Context methods.
	//
	//  * CookieAllowReclaim
	//  * CookieAllowSubdomains
	//  * CookieSecure
	//  * CookieHTTPOnly
	//  * CookieSameSite
	//  * CookiePath
	//  * CookieCleanPath
	//  * CookieExpires
	//  * CookieEncoding
	ctx.AddCookieOptions(iris.CookieAllowReclaim())
	ctx.Next()
}

func setCookie(ctx iris.Context) {
	name := ctx.Params().Get("name")
	value := ctx.Params().Get("value")

	ctx.SetCookieKV(name, value)

	// By-default net/http does not remove or set the Cookie on the Request object.
	//
	// With the `CookieAllowReclaim` option, whenever you set or remove a cookie
	// it will be also reflected in the Request object immediately (of the same request lifecycle)
	// therefore, any of the next handlers in the chain are not holding the old value.
	valueIsAvailableInRequestObject := ctx.GetCookie(name)
	ctx.Writef("cookie %s=%s", name, valueIsAvailableInRequestObject)
}

func getCookie(ctx iris.Context) {
	name := ctx.Params().Get("name")

	value := ctx.GetCookie(name)
	ctx.WriteString(value)
}

func removeCookie(ctx iris.Context) {
	name := ctx.Params().Get("name")

	ctx.RemoveCookie(name)

	removedFromRequestObject := ctx.GetCookie(name) // CookieAllowReclaim feature.
	ctx.Writef("cookie %s removed, value should be empty=%s", name, removedFromRequestObject)
}
