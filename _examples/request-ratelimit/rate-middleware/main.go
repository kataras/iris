package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/rate"
)

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")

	// * http://localhost:8080/v1
	// * http://localhost:8080/v1/other
	// * http://localhost:8080/v2/list (with X-API-Key request header)
	//   Read more at: https://en.wikipedia.org/wiki/Token_bucket
	//
	// Alternatives:
	//   * https://github.com/iris-contrib/middleware/blob/master/throttler/_example/main.go
	//     Read more at: https://en.wikipedia.org/wiki/Generic_cell_rate_algorithm
	//   * https://github.com/iris-contrib/middleware/tree/master/tollboothic/_examples/limit-handler
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	v1 := app.Party("/v1")
	{
		// Register the rate limiter middleware at the "/v1" subrouter.
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
		limitV1 := rate.Limit(1, 5, rate.PurgeEvery(time.Minute, 5*time.Minute))
		// rate.Every helper:
		//			   rate.Limit(rate.Every(time.Minute), 5)
		v1.Use(limitV1)

		v1.Get("/", index)
		v1.Get("/other", other)
	}

	v2 := app.Party("/v2")
	{
		v2.Use(useAPIKey)
		// Initialize a new rate limit middleware to limit requests
		// per API Key(see `useAPIKey` below) instead of client's Remote IP Address.
		limitV2 := rate.Limit(rate.Every(time.Minute), 300, rate.PurgeEvery(5*time.Minute, 15*time.Minute))
		v2.Use(limitV2)

		v2.Get("/list", list)
	}

	return app
}

func useAPIKey(ctx iris.Context) {
	apiKey := ctx.GetHeader("X-API-Key")
	if apiKey == "" { // [validate your API Key here...]
		ctx.StopWithStatus(iris.StatusForbidden)
		return
	}

	// Change the method that rate limit matches the requests with a specific user
	// and set our own api key as theirs identifier.
	rate.SetIdentifier(ctx, apiKey)
	ctx.Next()
}

func list(ctx iris.Context) {
	ctx.JSON(iris.Map{"key": "value"})
}

func index(ctx iris.Context) {
	ctx.HTML("<h1>Index Page</h1>")
}

func other(ctx iris.Context) {
	ctx.HTML("<h1>Other Page</h1>")
}

// Note: Use `ctx.SendFileWithRate` to use a download rate limiter instead.
