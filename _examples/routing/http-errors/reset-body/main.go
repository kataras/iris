package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
)

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()
	app.Use(iris.Compression)

	app.OnAnyErrorCode(onErrorCode)
	app.Get("/", handler)

	app.Configure(iris.WithResetOnFireErrorCode)
	return app
}

// This is the default error handler Iris uses for any error codes.
func onErrorCode(ctx iris.Context) {
	if err := ctx.GetErr(); err != nil {
		ctx.WriteString(err.Error())
	} else {
		ctx.WriteString(iris.StatusText(ctx.GetStatusCode()))
	}
}

func handler(ctx iris.Context) {
	ctx.Record()

	ctx.WriteString("This should NOT be written")

	// [....something bad happened after we "write"]
	err := fmt.Errorf("custom error")
	ctx.StopWithError(iris.StatusBadRequest, err)
}
