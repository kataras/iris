package main

import (
	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/middleware/pprof"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1> Please click <a href='/debug/pprof'>here</a>")
	})

	p := pprof.New()
	app.Any("/debug/pprof", p)
	app.Any("/debug/pprof/{action:path}", p)
	//                              ___________
	app.Listen(":8080")
}
