package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kataras/iris/v12"
)

const maxSize = 5 << 20 // 5MB

func main() {
	app := iris.New()

	app.RegisterView(iris.HTML("./templates", ".html"))

	// Serve the upload_form.html to the client.
	app.Get("/upload", func(ctx iris.Context) {
		// create a token (optionally).

		now := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(now, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))

		// render the form with the token for any use you'd like.
		// ctx.ViewData("", token)
		// or add second argument to the `View` method.
		// Token will be passed as {{.}} in the template.
		if err := ctx.View("upload_form.html", token); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	/* Read before continue.

	0. The default post max size is 32MB,
	you can extend it to read more data using the `iris.WithPostMaxMemory(maxSize)` configurator at `app.Run`,
	note that this will not be enough for your needs, read below.

	1. The faster way to check the size is using the `ctx.GetContentLength()` which returns the whole request's size
	(plus a logical number like 2MB or even 10MB for the rest of the size like headers). You can create a
	middleware to adapt this to any necessary handler.

	myLimiter := func(ctx iris.Context) {
		if ctx.GetContentLength() > maxSize { // + 2 << 20 {
			ctx.StatusCode(iris.StatusRequestEntityTooLarge)
			return
		}
		ctx.Next()
	}

	app.Post("/upload", myLimiter, myUploadHandler)

	Most clients will set the "Content-Length" header (like browsers) but it's always better to make sure that any client
	can't send data that your server can't or doesn't want to handle. This can be happen using
	the `app.Use(LimitRequestBodySize(maxSize))` (as app or route middleware)
	or the `ctx.SetMaxRequestBodySize(maxSize)` to limit the request based on a customized logic inside a particular handler, they're the same,
	read below.

	2. You can force-limit the request body size inside a handler using the `ctx.SetMaxRequestBodySize(maxSize)`,
	this will force the connection to close if the incoming data are larger (most clients will receive it as "connection reset"),
	use that to make sure that the client will not send data that your server can't or doesn't want to accept, as a fallback.

	app.Post("/upload", iris.LimitRequestBodySize(maxSize), myUploadHandler)

	OR

	app.Post("/upload", func(ctx iris.Context){
		ctx.SetMaxRequestBodySize(maxSize)

		// [...]
	})

	3. Another way is to receive the data and check the second return value's `Size` value of the `ctx.FormFile`, i.e `info.Size`, this will give you
	the exact file size, not the whole incoming request data length.

	app.Post("/", func(ctx iris.Context){
		file, info, err := ctx.FormFile("uploadfile")
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.HTML("Error while uploading: <b>" + err.Error() + "</b>")
			return
		}

		defer file.Close()

		if info.Size > maxSize {
			ctx.StatusCode(iris.StatusRequestEntityTooLarge)
			return
		}

		// [...]
	})
	*/

	// Handle the post request from the upload_form.html to the server
	app.Post("/upload", iris.LimitRequestBodySize(maxSize+1<<20), func(ctx iris.Context) {
		// Get the file from the request.
		f, fh, err := ctx.FormFile("uploadfile")
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.HTML("Error while uploading: <b>" + err.Error() + "</b>")
			return
		}
		defer f.Close()

		_, err = ctx.SaveFormFile(fh, filepath.Join("./uploads", fh.Filename))
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.HTML("Error while uploading: <b>" + err.Error() + "</b>")
			return
		}
	})

	// start the server at http://localhost:8080 with post limit at 5 MB.
	app.Listen(":8080" /* 0.*/, iris.WithPostMaxMemory(maxSize))
}
