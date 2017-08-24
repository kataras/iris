package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()
	app.Get("/api/values/{id}", func(ctx context.Context) {
		ctx.WriteString("value")
	})

	// 24 August 2017: Iris has a built'n version updater but we don't need it
	// when benchmarking...
	app.Run(iris.Addr(":5000"), iris.WithoutVersionChecker)
}
