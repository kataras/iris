package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
)

func init() {
	os.Mkdir("./uploads", 0700)
}

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))
	// Serve assets (e.g. javascript, css).
	// app.HandleDir("/public", "./public")

	app.Get("/", index)

	app.Get("/upload", uploadView)
	app.Post("/upload", upload)

	app.HandleDir("/files", "./uploads", iris.DirOptions{
		Gzip:     true,
		ShowList: true,
		DirList:  iris.DirListRich,
	})

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.Redirect("/upload")
}

func uploadView(ctx iris.Context) {
	now := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(now, 10))
	token := fmt.Sprintf("%x", h.Sum(nil))

	ctx.View("upload.html", token)
}

const maxSize = 1 * iris.GB

func upload(ctx iris.Context) {
	ctx.SetMaxRequestBodySize(maxSize)

	_, err := ctx.UploadFormFiles("./uploads", beforeSave)
	if err != nil {
		ctx.StopWithError(iris.StatusPayloadTooRage, err)
		return
	}

	ctx.Redirect("/files")
}

func beforeSave(ctx iris.Context, file *multipart.FileHeader) {
	ip := ctx.RemoteAddr()
	ip = strings.ReplaceAll(ip, ".", "_")
	ip = strings.ReplaceAll(ip, ":", "_")

	file.Filename = ip + "-" + file.Filename
}
