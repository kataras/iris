package main

import (
	"github.com/kataras/iris/_benchmarks/iris-mvc/controllers"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

func main() {
	app := iris.New()
	mvc.New(app.Party("/api/values/{id}")).
		Handle(new(controllers.ValuesController))

	app.Run(iris.Addr(":5000"))
}

// +2MB/s faster than the previous implementation, 0.4MB/s difference from the raw handlers.
