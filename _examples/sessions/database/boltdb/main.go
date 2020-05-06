package main

import (
	"errors"
	"os"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/sessions"
	"github.com/kataras/iris/v12/sessions/sessiondb/boltdb"
)

func main() {
	db, err := boltdb.New("./sessions.db", os.FileMode(0750))
	if err != nil {
		panic(err)
	}

	// close and unlobkc the database when control+C/cmd+C pressed
	iris.RegisterOnInterrupt(func() {
		db.Close()
	})

	defer db.Close() // close and unlock the database if application errored.

	sess := sessions.New(sessions.Config{
		Cookie:       "sessionscookieid",
		Expires:      45 * time.Minute, // <=0 means unlimited life. Defaults to 0.
		AllowReclaim: true,
	})

	// The default database's values encoder and decoder
	// calls the value's `Marshal/Unmarshal` methods (if any)
	// otherwise JSON is selected,
	// the JSON format can be stored to any database and
	// it supports both builtin language types(e.g. string, int) and custom struct values.
	// Also, and the most important, the values can be
	// retrieved/logged/monitored by a third-party program
	// written in any other language as well.
	//
	// You can change this behavior by registering a custom `Transcoder`.
	// Iris provides a `GobTranscoder` which is mostly suitable
	// if your session values are going to be custom Go structs.
	// Select this if you always retrieving values through Go.
	// Don't forget to initialize a call of gob.Register when necessary.
	// Read https://golang.org/pkg/encoding/gob/ for more.
	//
	// You can also implement your own `sessions.Transcoder` and use it,
	// i.e: a transcoder which will allow(on Marshal: return its byte representation and nil error)
	// or dissalow(on Marshal: return non nil error) certain types.
	//
	// sessions.DefaultTranscoder = sessions.GobTranscoder{}

	//
	// IMPORTANT:
	//
	sess.UseDatabase(db)

	// the rest of the code stays the same.
	app := iris.New()
	app.Logger().SetLevel("debug")

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})
	app.Get("/set", func(ctx iris.Context) {
		s := sess.Start(ctx)
		// set session values
		s.Set("name", "iris")

		// test if set here
		ctx.Writef("All ok session value of the 'name' is: %s", s.GetString("name"))
	})

	app.Get("/set/{key}/{value}", func(ctx iris.Context) {
		key, value := ctx.Params().Get("key"), ctx.Params().Get("value")
		s := sess.Start(ctx)
		// set session values
		s.Set(key, value)

		// test if set here
		ctx.Writef("All ok session value of the '%s' is: %s", key, s.GetString(key))
	})

	app.Get("/get", func(ctx iris.Context) {
		// get a specific key, as string, if no found returns just an empty string
		name := sess.Start(ctx).GetString("name")

		ctx.Writef("The 'name' on the /set was: %s", name)
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
		// destroy, removes the entire session data and cookie
		sess.Destroy(ctx)
	})

	app.Get("/update", func(ctx iris.Context) {
		// updates resets the expiration based on the session's `Expires` field.
		if err := sess.ShiftExpiration(ctx); err != nil {
			if errors.Is(err, sessions.ErrNotFound) {
				ctx.StatusCode(iris.StatusNotFound)
			} else if errors.Is(err, sessions.ErrNotImplemented) {
				ctx.StatusCode(iris.StatusNotImplemented)
			} else {
				ctx.StatusCode(iris.StatusNotModified)
			}

			ctx.Writef("%v", err)
			ctx.Application().Logger().Error(err)
		}
	})

	app.Listen(":8080")
}
