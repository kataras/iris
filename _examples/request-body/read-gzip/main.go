package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")

	app.Listen(":8080")
}

type payload struct {
	Message string `json:"message"`
}

func newApp() *iris.Application {
	app := iris.New()

	// GzipReader is a middleware which enables gzip decompression,
	// when client sends gzip compressed data.
	//
	// A shortcut of:
	// func(ctx iris.Context) {
	//	ctx.GzipReader(true)
	//	ctx.Next()
	// }
	app.Use(iris.GzipReader)

	app.Post("/", func(ctx iris.Context) {
		// Bind incoming gzip compressed JSON to "p".
		var p payload
		if err := ctx.ReadJSON(&p); err != nil {
			ctx.StopWithError(iris.StatusBadRequest, err)
			return
		}

		// Send back the message as plain text.
		ctx.WriteString(p.Message)
	})

	return app
}
