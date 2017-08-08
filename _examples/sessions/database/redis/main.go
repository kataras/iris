package main

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"

	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/sessions/sessiondb/redis"
	"github.com/kataras/iris/sessions/sessiondb/redis/service"
)

func main() {
	// replace with your running redis' server settings:
	db := redis.New(service.Config{
		Network:     service.DefaultRedisNetwork,
		Addr:        service.DefaultRedisAddr,
		Password:    "",
		Database:    "",
		MaxIdle:     0,
		MaxActive:   0,
		IdleTimeout: service.DefaultRedisIdleTimeout,
		Prefix:      ""}) // optionally configure the bridge between your redis server

	// use go routines to query the database
	db.Async(true)
	// close connection when control+C/cmd+C
	iris.RegisterOnInterrupt(func() {
		db.Close()
	})

	sess := sessions.New(sessions.Config{Cookie: "sessionscookieid", Expires: 45 * time.Minute})

	//
	// IMPORTANT:
	//
	sess.UseDatabase(db)

	// the rest of the code stays the same.
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})
	app.Get("/set", func(ctx context.Context) {
		s := sess.Start(ctx)
		//set session values
		s.Set("name", "iris")

		//test if setted here
		ctx.Writef("All ok session setted to: %s", s.GetString("name"))
	})

	app.Get("/get", func(ctx context.Context) {
		// get a specific key, as string, if no found returns just an empty string
		name := sess.Start(ctx).GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/delete", func(ctx context.Context) {
		// delete a specific key
		sess.Start(ctx).Delete("name")
	})

	app.Get("/clear", func(ctx context.Context) {
		// removes all entries
		sess.Start(ctx).Clear()
	})

	app.Get("/destroy", func(ctx context.Context) {
		//destroy, removes the entire session data and cookie
		sess.Destroy(ctx)
	})

	app.Get("/update", func(ctx context.Context) {
		// updates expire date with a new date
		sess.ShiftExpiraton(ctx)
	})

	app.Run(iris.Addr(":8080"))
}
