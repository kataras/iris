## Middleware information

This folder contains a middleware for the  build'n Iris logger but for the requests.

## How to use
```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
)

func main() {

	iris.UseFunc(logger.Default())
	// or iris.Use(logger.DefaultHandler())
	// or iris.UseFunc(iris.HandlerFunc(logger.DefaultHandler())
	// or iris.Get("/", logger.Default(), func (ctx *iris.Context){})
	// or iris.Get("/", iris.HandlerFunc(logger.DefaultHandler()), func (ctx *iris.Context){})

	// Custom settings:
	// ...
	// iris.UseFunc(logger.Custom(writer io.Writer, prefix string, flag int))
	// and so on...

	// Custom options:
	// ...
	// iris.UseFunc(logger.Default(logger.Options{IP:false}))  // don't log the ip
	// or iris.UseFunc(logger.Custom(writer io.Writer, prefix string, flag int, logger.Options{IP:false}))
	// and so on...

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Write("hello")
	})

	iris.Get("/1", func(ctx *iris.Context) {
		ctx.Write("hello")
	})

	iris.Get("/3", func(ctx *iris.Context) {
		ctx.Write("hello")
	})

	// IF YOU WANT LOGGER TO LOGS THE HTTP ERRORS ALSO THEN:
	// FUTURE: iris.OnError(404, logger.Default(logger.Options{Latency: false}))

	// NOW:
	errorLogger := logger.Default(logger.Options{Latency: false}) //here we just disable to log the latency, no need for error pages I think
	// yes we have options look at the logger.Options inside middleware/logger.go
	iris.OnError(404, func(ctx *iris.Context) {
		errorLogger.Serve(ctx)
		ctx.Write("My Custom 404 error page ")
	})
	//

	println("Server is running at :80")
	iris.Listen(":80")

}


```
