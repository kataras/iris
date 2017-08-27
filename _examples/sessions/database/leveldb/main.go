package main

import (
	"time"

	"github.com/kataras/iris"

	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/sessions/sessiondb/leveldb"
)

func main() {
	db, _ := leveldb.New("./sessions/")

	// use different go routines to sync the database
	db.Async(true)

	// close and unlock the database when control+C/cmd+C pressed
	iris.RegisterOnInterrupt(func() {
		db.Close()
	})

	sess := sessions.New(sessions.Config{
		Cookie:  "sessionscookieid",
		Expires: 45 * time.Minute, // <=0 means unlimited life
	})

	//
	// IMPORTANT:
	//
	sess.UseDatabase(db)

	// the rest of the code stays the same.
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})
	app.Get("/set", func(ctx iris.Context) {
		s := sess.Start(ctx)
		//set session values
		s.Set("name", "iris")

		//test if setted here
		ctx.Writef("All ok session setted to: %s", s.GetString("name"))
	})

	app.Get("/set/{key}/{value}", func(ctx iris.Context) {
		key, value := ctx.Params().Get("key"), ctx.Params().Get("value")
		s := sess.Start(ctx)
		// set session values
		s.Set(key, value)

		// test if setted here
		ctx.Writef("All ok session setted to: %s", s.GetString(key))
	})

	app.Get("/get", func(ctx iris.Context) {
		// get a specific key, as string, if no found returns just an empty string
		name := sess.Start(ctx).GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/get/{key}", func(ctx iris.Context) {
		// get a specific key, as string, if no found returns just an empty string
		name := sess.Start(ctx).GetString(ctx.Params().Get("key"))

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/delete", func(ctx iris.Context) {
		// delete a specific key
		sess.Start(ctx).Delete("name")
	})

	app.Get("/clear", func(ctx iris.Context) {
		// removes all entries
		sess.Start(ctx).Clear()
	})

	app.Get("/destroy", func(ctx iris.Context) {
		//destroy, removes the entire session data and cookie
		sess.Destroy(ctx)
	})

	app.Get("/update", func(ctx iris.Context) {
		// updates expire date with a new date
		sess.ShiftExpiration(ctx)
	})

	app.Run(iris.Addr(":8080"))
}
