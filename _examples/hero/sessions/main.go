package main

import (
	"time"

	"github.com/kataras/iris/_examples/hero/sessions/routes"

	"github.com/kataras/iris"
	"github.com/kataras/iris/hero" // <- IMPORTANT
	"github.com/kataras/iris/sessions"
)

func main() {
	app := iris.New()
	sessionManager := sessions.New(sessions.Config{
		Cookie:       "site_session_id",
		Expires:      60 * time.Minute,
		AllowReclaim: true,
	})

	// Register
	// dynamic dependencies like the *sessions.Session, from `sessionManager.Start(ctx) *sessions.Session` <- it accepts a Context and it returns
	// something -> this is called dynamic request-time dependency and that "something" can be used to your handlers as input parameters,
	// no limit about the number of dependencies, each handler will be builded once, before the server ran and it will use only dependencies that it needs.
	hero.Register(sessionManager.Start)
	// convert any function to an iris Handler, their input parameters are being resolved using the unique Iris' blazing-fast dependency injection
	// for services or dynamic dependencies like the *sessions.Session, from sessionManager.Start(ctx) *sessions.Session) <- it accepts a context and it returns
	// something-> this is called dynamic request-time dependency.
	indexHandler := hero.Handler(routes.Index)

	// Method: GET
	// Path: http://localhost:8080
	app.Get("/", indexHandler)

	app.Run(
		iris.Addr(":8080"),
		iris.WithoutServerError(iris.ErrServerClosed),
	)
}
