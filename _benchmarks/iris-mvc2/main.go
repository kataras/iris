package main

/// TODO: remove this on the "master" branch, or even replace it
// with the "iris-mvc" (the new implementatioin is even faster, close to handlers version,
// with bindings or without).

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/_benchmarks/iris-mvc2/controllers"
	"github.com/kataras/iris/mvc2"
)

func main() {
	app := iris.New()
	mvc2.New().Controller(app.Party("/api/values/{id}"), new(controllers.ValuesController))
	app.Run(iris.Addr(":5000"), iris.WithoutVersionChecker)
}
