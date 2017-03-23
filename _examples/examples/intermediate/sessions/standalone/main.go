package main

import (
	"time"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/sessions"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger()) // enable all (error) logs
	app.Adapt(httprouter.New()) // select the httprouter as the servemux

	mySessions := sessions.New(sessions.Config{
		// Cookie string, the session's client cookie name, for example: "mysessionid"
		//
		// Defaults to "irissessionid"
		Cookie: "mysessionid",
		// it's time.Duration, from the time cookie is created, how long it can be alive?
		// 0 means no expire.
		// -1 means expire when browser closes
		// or set a value, like 2 hours:
		Expires: time.Hour * 2,
		// the length of the sessionid's cookie's value
		CookieLength: 32,
		// if you want to invalid cookies on different subdomains
		// of the same host, then enable it
		DisableSubdomainPersistence: false,
		// want to be crazy safe? Take a look at the "securecookie" example folder.
	})

	// OPTIONALLY:
	// import "gopkg.in/kataras/iris.v6/adaptors/sessions/sessiondb/redis"
	// or import "github.com/kataras/go-sessions/sessiondb/$any_available_community_database"
	// mySessions.UseDatabase(redis.New(...))

	app.Adapt(mySessions) // Adapt the session manager we just created.

	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})
	app.Get("/set", func(ctx *iris.Context) {

		//set session values
		ctx.Session().Set("name", "iris")

		//test if setted here
		ctx.Writef("All ok session setted to: %s", ctx.Session().GetString("name"))
	})

	app.Get("/get", func(ctx *iris.Context) {
		// get a specific key, as string, if no found returns just an empty string
		name := ctx.Session().GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/delete", func(ctx *iris.Context) {
		// delete a specific key
		ctx.Session().Delete("name")
	})

	app.Get("/clear", func(ctx *iris.Context) {
		// removes all entries
		ctx.Session().Clear()
	})

	app.Get("/destroy", func(ctx *iris.Context) {

		//destroy, removes the entire session data and cookie
		ctx.SessionDestroy()
	})
	// Note about Destroy:
	//
	// You can destroy a session outside of a handler too, using the:
	// mySessions.DestroyByID
	// mySessions.DestroyAll

	app.Listen(":8080")
}
