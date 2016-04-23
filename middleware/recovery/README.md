## Middleware information

This folder contains a middleware for safety recover the server from panic

## How to use

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/recovery"
	"os"
)

func main() {

	iris.Use(recovery.New(os.Stderr)) // optional parameter is the writer which the stack of the panic will be printed

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Write("Hi, let's panic")
		panic("errorrrrrrrrrrrrrrr")
	})

	println("Server is running at :8080")
	iris.Listen(":8080")
}

```
