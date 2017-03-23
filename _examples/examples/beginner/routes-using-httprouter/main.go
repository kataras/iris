package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func hello(ctx *iris.Context) {
	ctx.Writef("Hello from %s", ctx.Path())
}

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	app.Adapt(httprouter.New())

	app.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.HTML(iris.StatusNotFound, "<h1>Custom not found handler </h1>")
	})

	app.Get("/", hello)
	app.Get("/users/:userid", func(ctx *iris.Context) {
		ctx.Writef("Hello user with  id: %s", ctx.Param("userid"))
	})

	app.Get("/myfiles/*file", func(ctx *iris.Context) {
		ctx.HTML(iris.StatusOK, "Hello, the dynamic path after /myfiles is:<br/> <b>"+ctx.Param("file")+"</b>")
	})

	app.Get("/users/:userid/messages/:messageid", func(ctx *iris.Context) {
		ctx.HTML(iris.StatusOK, `Message from user with id:<br/> <b>`+ctx.Param("userid")+`</b>,
            message id: <b>`+ctx.Param("messageid")+`</b>`)
	})

	// http://127.0.0.1:8080/users/42
	// http://127.0.0.1:8080/myfiles/mydirectory/myfile.zip
	// http://127.0.0.1:8080/users/42/messages/1
	app.Listen(":8080")
}
