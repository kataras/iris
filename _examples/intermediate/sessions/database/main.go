package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"

	"github.com/cdren/iris/sessions"
	"github.com/cdren/iris/sessions/sessiondb/redis"
	"github.com/cdren/iris/sessions/sessiondb/redis/service"
)

func main() {
	// replace with your running redis' server settings:
	db := redis.New(service.Config{Network: service.DefaultRedisNetwork,
		Addr:          service.DefaultRedisAddr,
		Password:      "",
		Database:      "",
		MaxIdle:       0,
		MaxActive:     0,
		IdleTimeout:   service.DefaultRedisIdleTimeout,
		Prefix:        "",
		MaxAgeSeconds: service.DefaultRedisMaxAgeSeconds}) // optionally configure the bridge between your redis server

	mySessions := sessions.New(sessions.Config{Cookie: "mysessionid"})

	//
	// IMPORTANT:
	//
	mySessions.UseDatabase(db)

	// the rest of the code stays the same.
	app := iris.New()
	// Attach the session manager we just created
	app.AttachSessionManager(mySessions)

	app.Get("/", func(ctx context.Context) {
		ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})
	app.Get("/set", func(ctx context.Context) {

		//set session values
		ctx.Session().Set("name", "iris")

		//test if setted here
		ctx.Writef("All ok session setted to: %s", ctx.Session().GetString("name"))
	})

	app.Get("/get", func(ctx context.Context) {
		// get a specific key, as string, if no found returns just an empty string
		name := ctx.Session().GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/delete", func(ctx context.Context) {
		// delete a specific key
		ctx.Session().Delete("name")
	})

	app.Get("/clear", func(ctx context.Context) {
		// removes all entries
		ctx.Session().Clear()
	})

	app.Get("/destroy", func(ctx context.Context) {
		//destroy, removes the entire session data and cookie
		ctx.SessionDestroy()
	})

	app.Run(iris.Addr(":8080"))
}
