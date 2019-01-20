package main

import (
	"github.com/kataras/iris"

	"github.com/kataras/iris/sessions"
)

func main() {
	app := iris.New()
	sess := sessions.New(sessions.Config{Cookie: "myappsessionid", AllowReclaim: true})

	app.Get("/set", func(ctx iris.Context) {
		s := sess.Start(ctx)
		s.SetFlash("name", "iris")
		ctx.Writef("Message setted, is available for the next request")
	})

	app.Get("/get", func(ctx iris.Context) {
		s := sess.Start(ctx)
		name := s.GetFlashString("name")
		if name == "" {
			ctx.Writef("Empty name!!")
			return
		}
		ctx.Writef("Hello %s", name)
	})

	app.Get("/test", func(ctx iris.Context) {
		s := sess.Start(ctx)
		name := s.GetFlashString("name")
		if name == "" {
			ctx.Writef("Empty name!!")
			return
		}

		ctx.Writef("Ok you are coming from /set ,the value of the name is %s", name)
		ctx.Writef(", and again from the same context: %s", name)
	})

	app.Run(iris.Addr(":8080"))
}
