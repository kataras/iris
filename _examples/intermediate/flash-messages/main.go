package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/sessions"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())
	sess := sessions.New(sessions.Config{Cookie: "myappsessionid"})
	app.Adapt(sess)

	app.Get("/set", func(ctx *iris.Context) {
		ctx.Session().SetFlash("name", "iris")
		ctx.Writef("Message setted, is available for the next request")
	})

	app.Get("/get", func(ctx *iris.Context) {
		name := ctx.Session().GetFlashString("name")
		if name == "" {
			ctx.Writef("Empty name!!")
			return
		}
		ctx.Writef("Hello %s", name)
	})

	app.Get("/test", func(ctx *iris.Context) {
		name := ctx.Session().GetFlashString("name")
		if name == "" {
			ctx.Writef("Empty name!!")
			return
		}

		ctx.Writef("Ok you are comming from /set ,the value of the name is %s", name)
		ctx.Writef(", and again from the same context: %s", name)

	})

	app.Listen(":8080")
}
