package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/view"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	app.Adapt(httprouter.New())
	app.Adapt(view.HTML("./templates", ".html"))

	// Serve the form.html to the user
	app.Get("/upload", func(ctx *iris.Context) {
		//create a token (optionally)

		now := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(now, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))
		//render the form with the token for any use you like
		ctx.Render("upload_form.html", token)
	})

	// Handle the post request from the upload_form.html to the server
	app.Post("/upload", iris.LimitRequestBodySize(10<<20),
		func(ctx *iris.Context) {
			// or use ctx.SetMaxRequestBodySize(10 << 20)
			//to limit the uploaded file(s) size.

			// Get the file from the request
			file, info, err := ctx.FormFile("uploadfile")

			if err != nil {
				ctx.HTML(iris.StatusInternalServerError,
					"Error while uploading: <b>"+err.Error()+"</b>")
				return
			}

			defer file.Close()
			fname := info.Filename

			// Create a file with the same name
			// assuming that you have a folder named 'uploads'
			out, err := os.OpenFile("./uploads/"+fname,
				os.O_WRONLY|os.O_CREATE, 0666)

			if err != nil {
				ctx.HTML(iris.StatusInternalServerError,
					"Error while uploading: <b>"+err.Error()+"</b>")
				return
			}
			defer out.Close()

			io.Copy(out, file)
		})

	// start the server at 127.0.0.1:8080
	app.Listen(":8080")
}
