package main

// developers can use any library to add a custom cookie encoder/decoder.
// At this example we use the gorilla's securecookie package:
// $ go get github.com/gorilla/securecookie
// $ go run main.go

import (
	"github.com/kataras/iris"

	"github.com/gorilla/securecookie"
)

var (
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey  = []byte("the-big-and-secret-fash-key-here")
	blockKey = []byte("lot-secret-of-characters-big-too")
	sc       = securecookie.New(hashKey, blockKey)
)

func newApp() *iris.Application {
	app := iris.New()

	// Set A Cookie.
	app.Get("/cookies/{name}/{value}", func(ctx iris.Context) {
		name := ctx.Params().Get("name")
		value := ctx.Params().Get("value")

		ctx.SetCookieKV(name, value, iris.CookieEncode(sc.Encode)) // <--

		ctx.Writef("cookie added: %s = %s", name, value)
	})

	// Retrieve A Cookie.
	app.Get("/cookies/{name}", func(ctx iris.Context) {
		name := ctx.Params().Get("name")

		value := ctx.GetCookie(name, iris.CookieDecode(sc.Decode)) // <--

		ctx.WriteString(value)
	})

	// Delete A Cookie.
	app.Delete("/cookies/{name}", func(ctx iris.Context) {
		name := ctx.Params().Get("name")

		ctx.RemoveCookie(name) // <--

		ctx.Writef("cookie %s removed", name)
	})

	return app
}

func main() {
	app := newApp()
	app.Run(iris.Addr(":8080"))
}
