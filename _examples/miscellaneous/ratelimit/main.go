package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/rate"
)

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")

	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	// Register the rate limiter middleware at the root router.
	//
	// Fist and second input parameters:
	//	  Allow 1 request per second, with a maximum burst size of 5.
	//
	// Third optional variadic input parameter:
	//    Can be a cleanup function.
	//    Iris provides a cleanup function that will check for old entries and remove them.
	//    You can customize it, e.g. check every 1 minute
	//    if a client's last visit was 5 minutes ago ("old" entry)
	//    and remove it from the memory.
	rateLimiter := rate.Limit(1, 5, rate.PurgeEvery(time.Minute, 5*time.Minute))
	app.Use(rateLimiter)

	// Routes.
	app.Get("/", index)
	app.Get("/other", other)

	return app
}

func index(ctx iris.Context) {
	ctx.HTML("<h1>Index Page</h1>")
}

func other(ctx iris.Context) {
	ctx.HTML("<h1>Other Page</h1>")
}
