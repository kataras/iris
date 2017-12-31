package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/kataras/iris"
)

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
		ctx.View("upload_form.html", token)
	})

	// Handle the post request from the upload_form.html to the server
	app.Post("/upload", func(ctx iris.Context) {
		// iris.LimitRequestBodySize(32 <<20) as middleware to a route
		// or use ctx.SetMaxRequestBodySize(32 << 20)
		// to limit the whole request body size,
		//
		// or let the configuration option at app.Run for global setting
		// for POST/PUT methods, including uploads of course.

		// Get the file from the request.
		file, info, err := ctx.FormFile("uploadfile")

		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.HTML("Error while uploading: <b>" + err.Error() + "</b>")
			return
		}

		defer file.Close()
		fname := info.Filename

		// Create a file with the same name
		// assuming that you have a folder named 'uploads'
		out, err := os.OpenFile("./uploads/"+fname,
			os.O_WRONLY|os.O_CREATE, 0666)

		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.HTML("Error while uploading: <b>" + err.Error() + "</b>")
			return
		}
		defer out.Close()

		io.Copy(out, file)
	})

	// start the server at http://localhost:8080 with post limit at 32 MB.
	app.Run(iris.Addr(":8080"), iris.WithPostMaxMemory(32<<20))
}
