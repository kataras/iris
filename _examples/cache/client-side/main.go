// Package main shows how you can use the `WriteWithExpiration`
// based on the "modtime", if it's newer than the request header then
// it will refresh the contents, otherwise will let the client (99.9% the browser)
// to handle the cache mechanism, it's faster than iris.Cache because server-side
// has nothing to do and no need to store the responses in the memory.
package main

import (
	"fmt"
	"time"

	"github.com/kataras/iris"
)

var modtime = time.Now()

func greet(ctx iris.Context) {
	ctx.Header("X-Custom", "my  custom header")
	response := fmt.Sprintf("Hello World! %s", time.Now())
	ctx.WriteWithExpiration([]byte(response), modtime)
}

func main() {
	app := iris.New()
	app.Get("/", greet)
	app.Run(iris.Addr(":8080"))
}
