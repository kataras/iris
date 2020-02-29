package main

import (
	"github.com/kataras/iris/v12/_benchmarks/iris-mvc/controllers"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	mvc.New(app.Party("/api/values/{id}")).
		Handle(new(controllers.ValuesController))

	app.Listen(":5000")
}

// +2MB/s faster than the previous implementation, 0.4MB/s difference from the raw handlers.
