package main

import (
	"strings"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/arl/statsviz"
)

// $ go get github.com/arl/statsviz
func main() {
	app := iris.New()
	// Register a router wrapper for this one.
	statsvizPath := "/debug/statsviz"
	serveRoot := statsviz.IndexAtRoot(statsvizPath)
	serveWS := statsviz.NewWsHandler(time.Second)
	app.UseRouter(func(ctx iris.Context) {
		// You can optimize this if branch, I leave it to you as an exercise.
		if strings.HasPrefix(ctx.Path(), statsvizPath+"/ws") {
			serveWS(ctx.ResponseWriter(), ctx.Request())
		} else if strings.HasPrefix(ctx.Path(), statsvizPath) {
			serveRoot(ctx.ResponseWriter(), ctx.Request())
		} else {
			ctx.Next()
		}
	})
	//

	// Register other routes.
	app.Get("/", index)

	// Navigate to: http://localhost:8080/debug/statsviz/
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.WriteString("Hello, World!")
}
