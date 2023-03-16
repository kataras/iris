package main

import (
	"github.com/kataras/iris/v12"

	"github.com/quic-go/quic-go/http3"
)

/*
	$ go get github.com/quic-go/quic-go@master
*/

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hello from Index")
	})

	// app.Configure(iris.WithOptimizations, or any other core config here)
	// app.Build()
	// http3.ListenAndServe(":443", "./localhost.cert", "./localhost.key", app)
	// OR:
	app.Run(iris.Raw(func() error {
		return http3.ListenAndServe(":443", "./localhost.cert", "./localhost.key", app)
	}))
}
