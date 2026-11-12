// Package httpcost measures the CPU, memory and time cost of serving a request.
// Its entry point is New(Options), which returns a Handler that stores a *Metric
// on the Context; read it back with Get.
package httpcost

import "github.com/kataras/iris/v14/context"

func init() {
	// Note(@kataras): the pattern must name THIS package's import path.
	// It read "iris/middleware/cache/httpcost*", a path that does not exist, so
	// the handler showed up in route traces as a raw Go symbol.
	context.SetHandlerName("iris/middleware/httpcost.*", "iris.httpcost")
	context.SetHandlerName("iris/auth/identity.NewCostHandler*", "iris.httpcost")
}

// New returns an Iris Handler that measures request cost
// using the provided Options. It starts a RunningProcess before
// ctx.Next() and stops it afterward, storing the resulting Metric
// into the request context.
// It should be registered before the handlers you want to measure.
//
// Usage:
//
//	app.Use(httpcost.New(httpcost.DefaultOptions))
func New(opts Options) context.Handler {
	computer := NewComputer(opts)
	return func(ctx *context.Context) {
		proc := computer.Start()
		// defer func() {
		// 	if err := recover(); err != nil {
		// 	}
		// No, if we have a panic don't calculate the cost, it is on us.
		// }()
		ctx.Next()
		metric := proc.Stop(ctx.ResponseWriter().Written())
		ctx.Values().Set(costMetricContextKey, metric)
	}
}

const costMetricContextKey = "iris.httpcost.metric"

// Get retrieves the cost Metric from the provided context.
// It returns the Metric and a boolean indicating if it was found.
func Get(ctx *context.Context) (*Metric, bool) {
	return ctx.ValueAs[*Metric](costMetricContextKey)
}
