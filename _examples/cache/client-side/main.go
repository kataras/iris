// Package main shows how you can use the `WriteWithExpiration`
// based on the "modtime", if it's newer than the request header then
// it will refresh the contents, otherwise will let the client (99.9% the browser)
// to handle the cache mechanism, it's faster than iris.Cache because server-side
// has nothing to do and no need to store the responses in the memory.
package main

import (
	"time"

	"github.com/kataras/iris"
)

const refreshEvery = 10 * time.Second

func main() {
	app := iris.New()
	app.Use(iris.Cache304(refreshEvery))
	// same as:
	// app.Use(func(ctx iris.Context) {
	// 	now := time.Now()
	// 	if modified, err := ctx.CheckIfModifiedSince(now.Add(-refresh)); !modified && err == nil {
	// 		ctx.WriteNotModified()
	// 		return
	// 	}

	// 	ctx.SetLastModified(now)

	// 	ctx.Next()
	// })

	app.Get("/", greet)
	app.Run(iris.Addr(":8080"))
}

func greet(ctx iris.Context) {
	ctx.Header("X-Custom", "my  custom header")
	ctx.Writef("Hello World! %s", time.Now())
}
