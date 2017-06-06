package main

import (
	"github.com/kataras/iris"

	"github.com/kataras/iris/typescript" // optionally
	"github.com/kataras/iris/typescript/editor"
)

func main() {
	app := iris.New()
	// adapt a router, order doesn't matters

	// optionally but good to have, I didn't put inside editor or the editor in the typescript compiler adaptors
	// because you may use tools like gulp and you may use the editor without the typescript compiler adaptor.
	// but if you need auto-compilation on .ts, we have a solution:
	ts := typescript.New()
	ts.Config.Dir = "./www/scripts/"
	ts.Attach(app) // attach the typescript compiler adaptor

	editorConfig := editor.Config{
		Hostname:   "localhost",
		Port:       4444,
		WorkingDir: "./www/scripts/", // "/path/to/the/client/side/directory/",
		Username:   "myusername",
		Password:   "mypassword",
	}
	e := editor.New(editorConfig)
	e.Attach(app) // attach the editor

	app.StaticWeb("/", "./www") // serve the index.html

	app.Run(iris.Addr(":8080"))
}
