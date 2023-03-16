package main

import (
	"reflect"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/view"
)

func main() {
	e := iris.Jet(nil, ".jet") // You can still use a file system though.
	e.AddFunc("greet", func(args view.JetArguments) reflect.Value {
		msg := "Hello, " + args.Get(0).String() + "!"
		return reflect.ValueOf(msg)
	})
	err := e.ParseTemplate("program.jet", `<h1>{{greet(.Name)}}</h1>`)
	if err != nil {
		panic(err)
	}
	e.Reload(true)

	app := iris.New()
	app.RegisterView(e)
	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	if err := ctx.View("program.jet", iris.Map{
		"Name": "Gerasimos",
	}); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
