package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"

	"github.com/cdren/iris/middleware/pprof"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.HTML("<h1> Please click <a href='/debug/pprof'>here</a>")
	})

	app.Any("/debug/pprof/{action:path}", pprof.New())
	//                              ___________
	app.Run(iris.Addr(":8080"))
}
