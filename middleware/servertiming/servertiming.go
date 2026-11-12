/*
Package servertiming provides middleware for the Iris Web Framework to add Server-Timing headers to HTTP responses.

It has no settings, so following the plain-handler rule its entry point is the `Handler` function itself; `Start`/`End` and `WriteHeader` add metrics of your own.

The Server-Timing header allows servers to communicate performance metrics about the request-response cycle to the client.
This can help developers diagnose and optimize server-side performance by providing insights into where time is being spent on the server.

The package includes:
- A middleware handler to automatically measure and add the total processing time of each request to the Server-Timing header.
- Functions to manually start and end timing metrics for specific operations within request handlers.
- A function to manually write the Server-Timing header with custom metrics.
*/
package servertiming

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/recorder"
)

func init() {
	context.SetHandlerName("iris/middleware/servertiming.*", "iris.servertiming")
}

// Metric represents a single server timing metric.
type Metric struct {
	Name        string        `json:"name" yaml:"Name"`
	Description string        `json:"description" yaml:"Description"`
	Duration    time.Duration `json:"duration" yaml:"Duration"`

	// internal use for Start and End.
	start time.Time
}

// Handler is a middleware that adds the Server-Timing header to the response.
// Read more at: https://www.w3.org/TR/server-timing/#the-server-timing-header-field.
func Handler(ctx *context.Context) {
	// Need to record the response to be able to write the final headers,
	// the next handler(s) may write content to the client directly,
	// so we won't have the chance if we don't record.
	previous := ctx.ResponseWriter()
	if rr, isNew := recorder.AcquireResponseRecorder(previous); isNew {
		// release only if acquired here, otherwise let it be released by the
		// handler that acquired it. The previous writer goes back on the Context
		// first: a handler registered outside this one reads the Context's writer
		// after the chain returns, and a released recorder may already belong to
		// another request.
		defer func() {
			ctx.SetResponseWriter(previous)
			recorder.ReleaseResponseRecorder(rr)
		}()
		ctx.SetResponseWriter(rr)
	}

	start := time.Now()

	// Call the next handler.
	ctx.Next()

	// Measure the elapsed time.
	elapsed := time.Since(start)

	// Create server timing metrics.
	metrics := []*Metric{
		{
			Name:        "app",
			Description: "Application processing",
			Duration:    elapsed,
		},
	}

	// Build the Server-Timing header value.
	WriteMetrics(ctx, metrics)
}

const (
	serverTimingMetricsContextKey = "iris.servertiming.metrics"
	serverTimingHeaderKey         = "Server-Timing"
)

// Start starts a new server timing metric.
// It should be called before the operation that you want to measure.
// The metric will be ended when the End function is called with the same name.
// The name should be unique for each metric.
// The description is optional.
//
// Example:
//
//	servertiming.Start(ctx, "db", "Database processing")
//	// [...your database operation here]
//	servertiming.End(ctx, "db")
//
// OR start multiple metrics at once:
//
//	servertiming.Start(ctx, "db", "Database processing")
//	servertiming.Start(ctx, "cache", "Cache processing")
//	// [...your database and cache operations here]
//	servertiming.End(ctx, "db")
//	servertiming.End(ctx, "cache")
func Start(ctx *context.Context, name, description string) {
	now := time.Now() // as close as possible to the operation that we want to measure.

	var metrics []*Metric
	if v := ctx.Values().Get(serverTimingMetricsContextKey); v != nil {
		metrics = v.([]*Metric)
	}

	t := &Metric{
		Name:        name,
		Description: description,
		Duration:    0,
		start:       now,
	}
	ctx.Values().Set(serverTimingMetricsContextKey, append(metrics, t))
}

// End ends a server timing metric.
// It should be called after the operation that you want to measure
// and before server sent any body or status write operation to the client.
// The name should be the same as the one used in the Start function.
// If the duration is not provided, the elapsed time will be calculated.
// If all metrics are done, the Server-Timing header will be written.
//
// Reports whether all started metrics have been written to the response's headers.
func End(ctx *context.Context, name string) bool {
	end := time.Now() // as close as possible to the operation that we want to measure.

	var metrics []*Metric
	if v := ctx.Values().Get(serverTimingMetricsContextKey); v != nil {
		metrics = v.([]*Metric)
	}

	if len(metrics) == 0 {
		return false
	}

	var metricsDoneCount int // count the metrics done, if all done, then write the final header.

	for _, metric := range metrics { // a pointer to the metrics, so we can update the duration to the context too.
		dur := metric.Duration
		if dur == 0 {
			if metric.Name == name {
				// Measure the elapsed time.
				dur = end.Sub(metric.start)
				metric.Duration = dur
			}
		}

		if dur != 0 {
			metricsDoneCount++
		}
	}

	if metricsDoneCount == len(metrics) {
		// we remove all metrics, so the next one of batches with Start(s) can be fresh.
		ctx.Values().Remove(serverTimingMetricsContextKey)
		WriteMetrics(ctx, metrics)

		// gc: drop the element references. The slice header itself is a local, so there is
		// nothing further to release here.
		for i := range metrics {
			metrics[i] = nil
		}

		return true
	}

	return false
}

// WriteMetrics writes the Server-Timing header to the response.
// It should be called after all metrics are done.
// If no metrics are provided, the header will not be written.
func WriteMetrics(ctx *context.Context, metrics []*Metric) {
	if len(metrics) == 0 {
		return
	}

	headers := ctx.ResponseWriter().Header()

	var headerValue strings.Builder // can be reused for each metric.
	for _, metric := range metrics {
		// Other way:
		// if i > 0 {
		// 	headerValue.WriteString(", ")
		// }
		// Escape both fields: they end up in a structured header, and Start(ctx,
		// name, desc) is an easy place to pass something a caller influenced. A
		// quote or a semicolon in a description otherwise breaks parsing for the
		// whole header.
		headerValue.WriteString(sanitizeMetricName(metric.Name))
		if metric.Description != "" {
			headerValue.WriteString(";desc=")
			headerValue.WriteString(strconv.Quote(sanitizeMetricValue(metric.Description)))
		}
		if metric.Duration > 0 {
			// Note(@kataras): nanoseconds, not Milliseconds().
			//
			// Milliseconds() truncates to a whole number FIRST, so every
			// sub-millisecond metric - which is most of them - printed as "0.00".
			headerValue.WriteString(fmt.Sprintf(";dur=%.2f", float64(metric.Duration.Nanoseconds())/1e6))
		}

		headers[serverTimingHeaderKey] = append(headers[serverTimingHeaderKey], headerValue.String())
		headerValue.Reset()
	}

	// header := headerValue.String()
	// headers.Add("Server-Timing", header)
}

// sanitizeMetricName keeps a metric name to the token characters the
// Server-Timing grammar allows, so a name can never break the header.
func sanitizeMetricName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_', r == '.':
			return r
		default:
			return '_'
		}
	}, name)
}

// sanitizeMetricValue strips control characters from a description before it is
// quoted into the header.
func sanitizeMetricValue(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}
