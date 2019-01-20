package main

import (
	"github.com/kataras/iris"

	"github.com/didip/tollbooth"
	"github.com/iris-contrib/middleware/tollboothic"
)

// $ go get github.com/didip/tollbooth
// $ go run main.go

func main() {
	app := iris.New()

	limiter := tollbooth.NewLimiter(1, nil)
	//
	// or create a limiter with expirable token buckets
	// This setting means:
	// create a 1 request/second limiter and
	// every token bucket in it will expire 1 hour after it was initially set.
	// limiter := tollbooth.NewLimiter(1, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})

	app.Get("/", tollboothic.LimitHandler(limiter), func(ctx iris.Context) {
		ctx.HTML("<b>Hello, world!</b>")
	})

	app.Run(iris.Addr(":8080"))
}

// Read more at: https://github.com/didip/tollbooth
