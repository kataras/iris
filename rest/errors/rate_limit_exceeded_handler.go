package errors

import (
	"github.com/kataras/iris/v14/context"

	"github.com/kataras/iris/v14/middleware/rate"
)

// DefaultRateLimitExceededHandler is the default handler which is being used when the `ExceedHandler` is not being passed.
var DefaultRateLimitExceededHandler = func(ctx *context.Context) {
	tryAgainDuration := rate.Get(ctx).Delay.String()
	ResourceExhausted.Detailf(ctx, "rate limit exceeded", "you may try again in %s", tryAgainDuration)
}
