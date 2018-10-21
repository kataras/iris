package main

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions"
)

var sess = sessions.New(sessions.Config{
	Cookie:  ".cookiesession.id",
	Expires: time.Minute,
})

func main() {
	app := iris.New()

	app.Get("/setget", h)
	/*
	 Test them one by one by these methods:
	 app.Get("/get", getHandler)
	 app.Post("/set", postHandler)
	 app.Delete("/del", delHandler)
	*/

	app.Run(iris.Addr(":5000"))
}

// Set and Get
func h(ctx context.Context) {
	session := sess.Start(ctx)
	session.Set("key", "value")

	value := session.GetString("key")
	if value == "" {
		ctx.WriteString("NOT_OK")
		return
	}

	ctx.WriteString(value)
}

// Get
func getHandler(ctx context.Context) {
	session := sess.Start(ctx)
	value := session.GetString("key")
	if value == "" {
		ctx.WriteString("NOT_OK")
		return
	}
	ctx.WriteString(value)
}

// Set
func postHandler(ctx context.Context) {
	session := sess.Start(ctx)
	session.Set("key", "value")
	ctx.WriteString("OK")
}

// Delete
func delHandler(ctx context.Context) {
	session := sess.Start(ctx)
	session.Delete("key")
	ctx.WriteString("OK")
}
