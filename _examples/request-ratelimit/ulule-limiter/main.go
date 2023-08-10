package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func main() {
	app := iris.New()
	app.Get("/hello", IPRateLimit(), helloWorldHandler) // 3. Use middleware
	app.Run(iris.Addr(":8080"))
}

func helloWorldHandler(ctx iris.Context) {
	err := ctx.StopWithJSON(iris.StatusOK, iris.Map{
		"message": "Hello World!",
	})
	if err != nil {
		return
	}
}

func IPRateLimit() iris.Handler {
	// 1. Configure
	rate := limiter.Rate{
		Period: 2 * time.Second,
		Limit:  1,
	}
	store := memory.NewStore()
	ipRateLimiter := limiter.New(store, rate)

	// 2. Return middleware handler
	return func(ctx iris.Context) {
		ip := ctx.RemoteAddr()
		limiterCtx, err := ipRateLimiter.Get(ctx.Request().Context(), ip)
		if err != nil {
			log.Printf("IPRateLimit - ipRateLimiter.Get - err: %v, %s on %s", err, ip, ctx.Request().URL)
			ctx.StatusCode(http.StatusInternalServerError)
			ctx.JSON(iris.Map{
				"success": false,
				"message": err,
			})
			return
		}

		ctx.Header("X-RateLimit-Limit", strconv.FormatInt(limiterCtx.Limit, 10))
		ctx.Header("X-RateLimit-Remaining", strconv.FormatInt(limiterCtx.Remaining, 10))
		ctx.Header("X-RateLimit-Reset", strconv.FormatInt(limiterCtx.Reset, 10))

		if limiterCtx.Reached {
			log.Printf("Too Many Requests from %s on %s", ip, ctx.Request().URL)
			ctx.StatusCode(http.StatusTooManyRequests)
			ctx.JSON(iris.Map{
				"success": false,
				"message": "Too Many Requests on " + ctx.Request().URL.String(),
			})
			return
		}
		ctx.Next()
	}
}
