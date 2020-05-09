package main

// developers can use any library to add a custom cookie encoder/decoder.
// At this example we use the gorilla's securecookie package:
// $ go get github.com/gorilla/securecookie
// $ go run main.go

import (
	"github.com/kataras/iris/v12"

	"github.com/gorilla/securecookie"
)

func main() {
	app := newApp()
	// http://localhost:8080/cookies/name/value
	// http://localhost:8080/cookies/name
	// http://localhost:8080/cookies/remove/name
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	r := app.Party("/cookies")
	{
		r.Use(useSecureCookies())

		// Set A Cookie.
		r.Get("/{name}/{value}", func(ctx iris.Context) {
			name := ctx.Params().Get("name")
			value := ctx.Params().Get("value")

			ctx.SetCookieKV(name, value)

			ctx.Writef("cookie added: %s = %s", name, value)
		})

		// Retrieve A Cookie.
		r.Get("/{name}", func(ctx iris.Context) {
			name := ctx.Params().Get("name")

			value := ctx.GetCookie(name)

			ctx.WriteString(value)
		})

		r.Get("/remove/{name}", func(ctx iris.Context) {
			name := ctx.Params().Get("name")

			ctx.RemoveCookie(name)

			ctx.Writef("cookie %s removed", name)
		})
	}

	return app
}

func useSecureCookies() iris.Handler {
	var (
		hashKey  = securecookie.GenerateRandomKey(64)
		blockKey = securecookie.GenerateRandomKey(32)

		s = securecookie.New(hashKey, blockKey)
	)

	return func(ctx iris.Context) {
		ctx.AddCookieOptions(iris.CookieEncoding(s))
		ctx.Next()
	}
}
