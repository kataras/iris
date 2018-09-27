package main

import "github.com/kataras/iris"

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteGzip([]byte("Hello World!"))
		ctx.Header("X-Custom",
			"Headers can be set here after WriteGzip as well, because the data are kept before sent to the client when using the context's GzipResponseWriter and ResponseRecorder.")
	})

	app.Get("/2", func(ctx iris.Context) {
		// same as the `WriteGzip`.
		// However GzipResponseWriter gives you more options, like
		// reset data, disable and more, look its methods.
		ctx.GzipResponseWriter().WriteString("Hello World!")
	})

	app.Run(iris.Addr(":8080"))
}
