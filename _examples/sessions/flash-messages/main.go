package main

import (
	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/sessions"
)

func main() {
	app := iris.New()

	sess := sessions.New(sessions.Config{Cookie: "_session_id", AllowReclaim: true})
	app.Use(sess.Handler())

	app.Get("/set", func(ctx iris.Context) {
		s := sessions.Get(ctx)
		s.SetFlash("name", "iris")
		ctx.Writef("Message set, is available for the next request")
	})

	app.Get("/get", func(ctx iris.Context) {
		s := sessions.Get(ctx)
		name := s.GetFlashString("name")
		if name == "" {
			ctx.Writef("Empty name!!")
			return
		}
		ctx.Writef("Hello %s", name)
	})

	app.Get("/test", func(ctx iris.Context) {
		s := sessions.Get(ctx)
		name := s.GetFlashString("name")
		if name == "" {
			ctx.Writef("Empty name!!")
			return
		}

		ctx.Writef("Ok you are coming from /set ,the value of the name is %s", name)
		ctx.Writef(", and again from the same context: %s", name)
	})

	app.Listen(":8080")
}
