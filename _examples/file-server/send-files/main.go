package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	app.Get("/", download)
	app.Get("/download", downloadWithRateLimit)

	app.Listen(":8080")
}

func download(ctx iris.Context) {
	src := "./files/first.zip"
	ctx.SendFile(src, "client.zip")
}

func downloadWithRateLimit(ctx iris.Context) {
	// REPLACE THAT WITH A BIG LOCAL FILE OF YOUR OWN.
	src := "./files/first.zip"
	dest := "" /* optionally, keep it empty to resolve the filename based on the "src" */

	// Limit download speed to ~50Kb/s with a burst of 100KB.
	limit := 50.0 * iris.KB
	burst := 100 * iris.KB
	ctx.SendFileWithRate(src, dest, limit, burst)
}
