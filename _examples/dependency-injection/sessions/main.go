package main

import (
	"time"

	"github.com/kataras/iris/v12/_examples/dependency-injection/sessions/routes"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
)

func main() {
	app := iris.New()
	sessionManager := sessions.New(sessions.Config{
		Cookie:       "site_session_id",
		Expires:      60 * time.Minute,
		AllowReclaim: true,
	})

	// Session is automatically binded through `sessions.Get(ctx)`
	// if a *sessions.Session input argument is present on the handler's function,
	// which `routes.Index` does.
	app.Use(sessionManager.Handler())

	// Method: GET
	// Path: http://localhost:8080
	app.ConfigureContainer(registerRoutes)

	app.Listen(":8080")
}

func registerRoutes(api *iris.APIContainer) {
	api.Get("/", routes.Index)
}
