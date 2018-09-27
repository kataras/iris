package main

import (
	"github.com/kataras/iris"

	"github.com/kataras/iris/typescript/editor"
)

func main() {
	app := iris.New()
	app.StaticWeb("/scripts", "./www/scripts") // serve the scripts
	// when you edit a typescript file from the alm-tools
	// it compiles it to javascript, have fun!

	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("./www/index.html", false)
	})

	editorConfig := editor.Config{
		Hostname:   "localhost",
		Port:       4444,
		WorkingDir: "./www/scripts/", // "/path/to/the/client/side/directory/",
		Username:   "myusername",
		Password:   "mypassword",
	}
	e := editor.New(editorConfig)
	e.Run(app.Logger().Infof) // start the editor's server

	// http://localhost:8080
	// http://localhost:4444
	app.Run(iris.Addr(":8080"))
	e.Stop()
}
