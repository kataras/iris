package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
)

type businessModel struct {
	Name string
}

func main() {
	app := iris.New()
	sess := sessions.New(sessions.Config{
		// Cookie string, the session's client cookie name, for example: "mysessionid"
		//
		// Defaults to "irissessionid"
		Cookie: "mysessionid",
		// it's time.Duration, from the time cookie is created, how long it can be alive?
		// 0 means no expire.
		// -1 means expire when browser closes
		// or set a value, like 2 hours:
		Expires: time.Hour * 2,
		// if you want to invalid cookies on different subdomains
		// of the same host, then enable it.
		// Defaults to false.
		DisableSubdomainPersistence: false,
	})

	app.Use(sess.Handler()) // session is always non-nil inside handlers now.

	app.Get("/", func(ctx iris.Context) {
		session := sessions.Get(ctx) // same as sess.Start(ctx, cookieOptions...)
		if session.Len() == 0 {
			ctx.HTML(`no session values stored yet. Navigate to: <a href="/set">set page</a>`)
			return
		}

		ctx.HTML("<ul>")
		session.Visit(func(key string, value interface{}) {
			ctx.HTML("<li> %s = %v </li>", key, value)
		})

		ctx.HTML("</ul>")
	})

	// set session values.
	app.Get("/set", func(ctx iris.Context) {
		session := sessions.Get(ctx)
		session.Set("name", "iris")

		// test if set here.
		ctx.Writef("All ok session set to: %s", session.GetString("name"))

		// Set will set the value as-it-is,
		// if it's a slice or map
		// you will be able to change it on .Get directly!
		// Keep note that I don't recommend saving big data neither slices or maps on a session
		// but if you really need it then use the `SetImmutable` instead of `Set`.
		// Use `SetImmutable` consistently, it's slower.
		// Read more about muttable and immutable go types: https://stackoverflow.com/a/8021081
	})

	app.Get("/set/{key}/{value}", func(ctx iris.Context) {
		key, value := ctx.Params().Get("key"), ctx.Params().Get("value")

		session := sessions.Get(ctx)
		session.Set(key, value)

		// test if set here
		ctx.Writef("All ok session value of the '%s' is: %s", key, session.GetString(key))
	})

	app.Get("/get", func(ctx iris.Context) {
		// get a specific value, as string,
		// if not found then it returns just an empty string.
		name := sessions.Get(ctx).GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/delete", func(ctx iris.Context) {
		// delete a specific key
		sessions.Get(ctx).Delete("name")
	})

	app.Get("/clear", func(ctx iris.Context) {
		// removes all entries.
		sessions.Get(ctx).Clear()
	})

	app.Get("/update", func(ctx iris.Context) {
		// updates expire date.
		sess.ShiftExpiration(ctx)
	})

	app.Get("/destroy", func(ctx iris.Context) {
		// destroy, removes the entire session data and cookie
		// sess.Destroy(ctx)
		// or
		sessions.Get(ctx).Destroy()
	})
	// Note about Destroy:
	//
	// You can destroy a session outside of a handler too, using the:
	// sess.DestroyByID
	// sess.DestroyAll

	// remember: slices and maps are muttable by-design
	// The `SetImmutable` makes sure that they will be stored and received
	// as immutable, so you can't change them directly by mistake.
	//
	// Use `SetImmutable` consistently, it's slower than `Set`.
	// Read more about muttable and immutable go types: https://stackoverflow.com/a/8021081
	app.Get("/set_immutable", func(ctx iris.Context) {
		business := []businessModel{{Name: "Edward"}, {Name: "value 2"}}
		session := sessions.Get(ctx)
		session.SetImmutable("businessEdit", business)
		businessGet := session.Get("businessEdit").([]businessModel)

		// try to change it, if we used `Set` instead of `SetImmutable` this
		// change will affect the underline array of the session's value "businessEdit", but now it will not.
		businessGet[0].Name = "Gabriel"
	})

	app.Get("/get_immutable", func(ctx iris.Context) {
		valSlice := sessions.Get(ctx).Get("businessEdit")
		if valSlice == nil {
			ctx.HTML("please navigate to the <a href='/set_immutable'>/set_immutable</a> first")
			return
		}

		firstModel := valSlice.([]businessModel)[0]
		// businessGet[0].Name is equal to Edward initially
		if firstModel.Name != "Edward" {
			panic("Report this as a bug, immutable data cannot be changed from the caller without re-SetImmutable")
		}

		ctx.Writef("[]businessModel[0].Name remains: %s", firstModel.Name)

		// the name should remains "Edward"
	})

	app.Run(iris.Addr(":8080"))
}
