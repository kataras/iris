package counter

import "github.com/kataras/iris/v14/context"

func init() {
	context.SetHandlerName("iris/middleware/counter.*", "iris.counter")
}

// RequestCounter returns a new handler which increments the counter before serving the request
// and decrements the counter after the request is done.
//
// Usage:
//
//	app := iris.New()
//	c := NewCounter[uint64]()
//	app.Use(RequestCounter(c))
//
//	[...]
//	currentRequestCount := c.Get()
//
// OR use the Notifications channel to listen for counter changes:
//
//	for notif := range c.Notifications() {
//	  log.Printf("Counter value changed from %d to %d", notif.OldValue, notif.NewValue)
//	}
//
// You may use the GlobalCounter as the counter argument:
//
//	app.Use(RequestCounter(GlobalCounter))
//
// Other ideas you may use the Counter:
// * Rate Limiting
// Use the counter to limit the number of requests a client can make in a given period.
// This helps prevent abuse and ensures fair usage.
//
// * Concurrency Limiting
// Track the number of concurrent requests being processed.
// If the counter exceeds a certain limit, you can reject or queue incoming requests.
func RequestCounter[T Number](counter *Counter[T]) context.Handler {
	return func(ctx *context.Context) {
		counter.Increment()
		// Note(@kataras): defer, so a panicking handler still decrements.
		//
		// Without it one panic permanently inflates the in-flight count, and a
		// recovery middleware does not help because it unwinds past this frame.
		// For the "Concurrency Limiting" use documented above, enough panics
		// saturate the counter at Max and the limiter rejects everything forever.
		defer counter.Decrement()
		ctx.Next()
	}
}
