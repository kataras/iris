// Static file serving with the fileserver controller, hardened with the BlockHidden asset validator.
//
// Without the validator, dotfiles such as .env, .git/config or .htpasswd
// inside the served directory are reachable.
//
// Learn: serve a directory with an index file and listings, and keep every dot-prefixed path out of reach.
// Before: ../typed-context
//
// Run from this directory, so that ./public resolves.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/static           # 200, serves ./public/index.html
//	curl -i http://localhost:8080/static/.env   # 404, denied by BlockHidden
//
// Next: ../multi-app
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()

	// Registers GET|HEAD /static and GET|HEAD /static/{file:path}.
	// The first argument may be a directory path, an http.FileSystem,
	// an embed.FS or an fs.FS.
	app.Router("/static", fileserver.New("./public", fileserver.Options{
		// Serve this file for directory requests.
		IndexName: "index.html",
		// Render a listing for directories that have no index file.
		ShowList: true,
		// Keep hidden entries out of the listing as well.
		ShowHidden: false,
		// Per-request gate. Returning false denies the asset: the response is
		// 404 unless the validator wrote a status code itself. BlockHidden
		// denies any path with a dot-prefixed segment, including conventional
		// ones such as .well-known. Compose your own validator on top of it
		// if you need to allow specific directories.
		AssetValidator: fileserver.BlockHidden,
	}))

	app.Get("/", func(ctx iris.Context) {
		ctx.Redirect("/static")
	})

	app.Listen(":8080")
}
