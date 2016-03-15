package main

import (
	"crypto/md5"
	"fmt"
	"github.com/kataras/iris"
	"io"
	"os"
	"strconv"
	"time"
)

func main() {
	// you can debug path with get working directory
	// s, _ := os.Getwd()
	// println(s)
	//
	iris.Templates("./_examples/file_upload_simple/*")

	// Serve the form.html to the user
	iris.Get("/upload", func(ctx *iris.Context) {
		//these are optionaly you can just call RenderFile("form.html",{})
		//create the token
		now := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(now, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))
		//render the form with the token for any use you like
		ctx.RenderFile("form.html", token)
	})

	// Handle the post request from the form.html to the server
	iris.Post("/upload", func(ctx *iris.Context) {
		// Set maxMemory
		/*
			After you call ParseMultipartForm, the file will be saved in the server memory with maxMemory size.
			If the file size is larger than maxMemory, the rest of the data will be saved in a system temporary file
		*/
		ctx.Request.ParseMultipartForm(32 << 20) //32MB

		// Get the file from the request
		file, info, _ := ctx.Request.FormFile("uploadfile")
		defer file.Close()
		fname := info.Filename

		// Create a file with the same name
		// assuming that you have a folder named 'uploads'
		out, err := os.OpenFile("./uploads/"+fname, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer out.Close()

		io.Copy(out, file)

	})

	fmt.Println("Iris is listening on :8080")
	iris.Listen("8080")

}
