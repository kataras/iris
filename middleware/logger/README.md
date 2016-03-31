## Middleware information

### THIS IS NOT READY YET

This folder contains a middleware for the  build'n Iris logger but for the requests.

## How to use
```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
)

func main() {

	iris.UseFunc(logger.Logger())
	// or iris.Use(logger.LoggerHandler())
	// or iris.UseFunc(iris.HandlerFunc(logger.LoggerHandler())
	// or iris.Get("/", logger.Logger(), func (ctx *iris.Context){}) 
	// or iris.Get("/", iris.HandlerFunc(logger.LoggerHandler()), func (ctx *iris.Context){}) 
	
	// ...
	// or iris.UseFunc(logger.CustomLogger(writer io.Writer, prefix string, flag int))	
	// and so on...
	
	
		
	iris.Get("/", func(ctx *iris.Context) {
	
	})
	
	
	println("Server is running at :8080")
	iris.Listen(":8080")

}

```