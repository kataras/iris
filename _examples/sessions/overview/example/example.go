package example

import (
	"errors"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
)

// BusinessModel is just a Go struct value that we will use in our session example,
// never save sensitive information, like passwords, here.
type BusinessModel struct {
	Name string
}

// NewApp returns a new application for showcasing the sessions feature.
func NewApp(sess *sessions.Sessions) *iris.Application {
	app := iris.New()
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

		ctx.Writef("All ok session set to: %s", session.GetString("name"))
	})

	app.Get("/get", func(ctx iris.Context) {
		session := sessions.Get(ctx)

		// get a specific value, as string,
		// if not found then it returns just an empty string.
		name := session.GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/set-struct", func(ctx iris.Context) {
		session := sessions.Get(ctx)
		session.Set("struct", BusinessModel{Name: "John Doe"})

		ctx.Writef("All ok session value of the 'struct' is: %v", session.Get("struct"))
	})

	app.Get("/get-struct", func(ctx iris.Context) {
		session := sessions.Get(ctx)
		ctx.Writef("Session value of the 'struct' is: %v", session.Get("struct"))
	})

	app.Get("/set/{key}/{value}", func(ctx iris.Context) {
		session := sessions.Get(ctx)

		key := ctx.Params().Get("key")
		value := ctx.Params().Get("value")
		session.Set(key, value)

		ctx.Writef("All ok session value of the '%s' is: %s", key, session.GetString(key))
	})

	app.Get("/get/{key}", func(ctx iris.Context) {
		session := sessions.Get(ctx)
		// get a specific key, as string, if no found returns just an empty string
		key := ctx.Params().Get("key")
		name := session.GetString(key)

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/delete", func(ctx iris.Context) {
		session := sessions.Get(ctx)
		// delete a specific key
		session.Delete("name")
	})

	app.Get("/clear", func(ctx iris.Context) {
		session := sessions.Get(ctx)
		// removes all entries.
		session.Clear()
	})

	app.Get("/update", func(ctx iris.Context) {
		session := sessions.Get(ctx)
		// shifts the expiration based on the session's `Lifetime`.
		if err := session.Man.ShiftExpiration(ctx); err != nil {
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

	app.Get("/destroy", func(ctx iris.Context) {
		session := sessions.Get(ctx)
		// Man(anager)'s Destroy, removes the entire session data and cookie
		session.Man.Destroy(ctx)
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
	app.Get("/set-immutable", func(ctx iris.Context) {
		session := sessions.Get(ctx)

		business := []BusinessModel{{Name: "Edward"}, {Name: "value 2"}}
		session.SetImmutable("businessEdit", business)
		businessGet := session.Get("businessEdit").([]BusinessModel)

		// try to change it, if we used `Set` instead of `SetImmutable` this
		// change will affect the underline array of the session's value "businessEdit", but now it will not.
		businessGet[0].Name = "Gabriel"
	})

	app.Get("/get-immutable", func(ctx iris.Context) {
		valSlice := sessions.Get(ctx).Get("businessEdit")
		if valSlice == nil {
			ctx.HTML("please navigate to the <a href='/set_immutable'>/set-immutable</a> first")
			return
		}

		firstModel := valSlice.([]BusinessModel)[0]
		// businessGet[0].Name is equal to Edward initially
		if firstModel.Name != "Edward" {
			panic("Report this as a bug, immutable data cannot be changed from the caller without re-SetImmutable")
		}

		ctx.Writef("[]businessModel[0].Name remains: %s", firstModel.Name)

		// the name should remains "Edward"
	})

	return app
}
