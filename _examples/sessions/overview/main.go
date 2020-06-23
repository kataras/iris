package main

import (
	"time"

	"github.com/kataras/iris/v12/_examples/sessions/overview/example"
	"github.com/kataras/iris/v12/sessions"
)

func main() {
	sess := sessions.New(sessions.Config{
		// Cookie string, the session's client cookie name, for example: "_session_id"
		//
		// Defaults to "irissessionid"
		Cookie: "_session_id",
		// it's time.Duration, from the time cookie is created, how long it can be alive?
		// 0 means no expire, unlimited life.
		// -1 means expire when browser closes
		// or set a value, like 2 hours:
		Expires: time.Hour * 2,
		// if you want to invalid cookies on different subdomains
		// of the same host, then enable it.
		// Defaults to false.
		DisableSubdomainPersistence: false,
		// Allow getting the session value stored by the request from the same request.
		AllowReclaim: true,
		/*
			SessionIDGenerator: func(ctx iris.Context) string {
				id:= ctx.GetHeader("X-Session-Id")
				if id == "" {
					id = // [generate ID here and set the header]
					ctx.Header("X-Session-Id", id)
				}

				return id
			},
		*/
	})

	app := example.NewApp(sess)
	app.Listen(":8080")
}
