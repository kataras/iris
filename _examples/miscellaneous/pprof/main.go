package main

import (
	"github.com/kataras/iris"

	"github.com/kataras/iris/middleware/pprof"
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
	app.Run(iris.Addr(":8080"))
}
