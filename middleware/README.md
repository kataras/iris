# Middleware
Iris has it's build'n small middleware(s) here

# Structure
**All Iris build'n middleware(s)* belong here, to this folder 'iris/middleware' **
middleware is also the package name.




Each middleware is written to file(s) but no to different folder(s),
that's means that iris/middleware doesn't have any children folder.



All middleware(s) have share code via the same package name so be carefuly when you write a midleware, we need sharing between them but no conficts.


----------------------------

# How to write
Simple, Import iris and use it to the middleware

```go
import (
	iris "github.com/kataras/iris"
)
```

Each middleware must export only one Function which returns an object which implements the iris.Handler (func Serve(ctx *iris.Context){})


----------------------------

# Why? Three reasons

1. Easier import path for all middleware(s) (e.g github.com/kataras/iris/middleware vs github.com/kataras/iris/middleware/gzip github.com/kataras/iris/middleware/othermiddleware and so on)
2. Minimize the code that Iris middleware(s) will re-uses
3. Protect ourself from import circles between middleware(s).

# How to use a middleware

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware"
)

type Page struct {
	Title string
}

func main() {
	iris.Templates("./_examples/compression_gzip/templates/*.html")
	// here is How to use a middleware
	iris.Use(middleware.Gzip(middleware.DefaultCompression))
	//
	iris.Get("/public/*static", iris.Static("./_examples/compression_gzip/static/", "/public/"))

	iris.Get("/", func(c *iris.Context) {
		c.RenderFile("index.html", Page{"My Index Title"})
	})

	iris.Listen(":8080")
}

```