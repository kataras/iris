# Middleware
Iris has it's build'n small middleware(s) here

# Structure
**All Iris build'n middleware(s)* belong here, to this folder 'iris/middleware' **
middleware is also the package name.



Each middleware is written to it's own folder.

----------------------------

# How to write
Simple, Import iris and use it to the middleware

```go
import (
	iris "github.com/kataras/iris"
)
```

Each middleware is recommending that exports only one Function which returns an object which implements the iris.Handler (func Serve(ctx *iris.Context){}). [Look here for an example](https://github.com/kataras/iris/blob/master/middleware/gzip/gzip.go#L79)

----------------------------


# How to use a middleware

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/gzip"
)

type Page struct {
	Title string
}

func main() {
	iris.Templates("./_examples/compression_gzip/templates/*.html")
	// here is How to use a middleware
	iris.Use(gzip.Gzip(gzip.DefaultCompression))
	//
	iris.Get("/public/*static", iris.Static("./_examples/compression_gzip/static/", "/public/"))

	iris.Get("/", func(c *iris.Context) {
		c.RenderFile("index.html", Page{"My Index Title"})
	})

	iris.Listen(":8080")
}

```