package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/_benchmarks/iris-mvc/controllers"
)

func main() {
	app := iris.New()
	app.Controller("/api/values/{id}", new(controllers.ValuesController))

	// 24 August 2017: Iris has a built'n version updater but we don't need it
	// when benchmarking...
	app.Run(iris.Addr(":5000"), iris.WithoutVersionChecker)
}
