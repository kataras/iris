// Access logging: one structured record per request, with your own fields.
//
// accesslog.New(w) writes to any io.Writer (accesslog.File(path) and
// accesslog.FileUnbuffered(path) open a file instead). SetFormatter picks the
// output shape - &accesslog.JSON{}, &accesslog.CSV{} or
// &accesslog.Template{Text: "..."}; without one the default pipe-delimited
// text format is used. AddFields attaches values extracted from every request,
// and accesslog.GetFields(ctx) lets a single handler add its own.
//
// Register the middleware with UseRouter so 404s and 405s are recorded too,
// and Close it on shutdown to flush buffered output.
//
// Learn: write one structured log record per request, add your own fields from
// middleware or a handler, and skip noisy endpoints.
// Before: ../request-id
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/                                        # 200, one JSON log line
//	curl -X POST http://localhost:8080/orders -d '{"sku":"IRIS-14"}'   # 201, the log line carries order_id
//	curl http://localhost:8080/health                                  # 200, skipped: no log line
//	curl http://localhost:8080/nope                                    # 404, still logged
//
// Next: ../cache
package main

import (
	"os"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/requestid"
)

func main() {
	app := iris.New()

	ac := makeAccessLog()
	defer ac.Close() // flushes and releases the writers.

	// UseRouter runs before routing, so unmatched requests are logged as well.
	app.UseRouter(ac.Handler)
	app.Use(requestid.New(requestid.ServerOnlyGenerator))

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"hello": "world"})
	})

	app.Post("/orders", func(ctx iris.Context) {
		// accesslog.GetFields adds fields for this request only, so a handler
		// can record what it alone knows.
		accesslog.GetFields(ctx).Set("order_id", 42)
		// ctx.Created sends 201 and wraps a bare identifier as {"id": ...}.
		ctx.Created(42, nil)
	})

	// accesslog.SkipHandler is the middleware form of accesslog.Skip(ctx):
	// it keeps noisy endpoints such as health probes out of the log.
	app.Get("/health", accesslog.SkipHandler, func(ctx iris.Context) {
		ctx.WriteString("OK")
	})

	// With the production stack, iris.NewBuilder().LogRequests(true) installs a
	// pre-tuned instance and registers its Close as a shutdown closer. To keep
	// this configuration instead, pass it to the same slot:
	//
	//	app := iris.NewBuilder().
	//		Prefix("/api").
	//		AllowOrigin("*").
	//		Compression(true).
	//		LogRequestsWith(makeAccessLog()).
	//		// Health, Errors, Services, middlewares, API...
	//		Build()

	app.Listen(":8080")
}

func makeAccessLog() *accesslog.AccessLog {
	ac := accesslog.New(os.Stdout)

	// One compact JSON object per line, which is what log collectors want.
	// HumanTime prints the TimeFormat-formatted time instead of a unix
	// timestamp.
	ac.SetFormatter(&accesslog.JSON{HumanTime: true})

	ac.Clock = accesslog.UTC // log timestamps in UTC.
	ac.TimeFormat = "2006-01-02 15:04:05"
	ac.IP = true                // include the remote address.
	ac.RequestBody = true       // log the request payload.
	ac.ResponseBody = false     // response bodies are usually noise.
	ac.BytesReceivedBody = true // sizes of the bodies only, which is cheap.
	ac.BytesSentBody = true
	// ac.OnlyFailures = true         // log 4xx/5xx responses only.
	// ac.Async = true                // print from a separate goroutine.

	// AddFields runs for every request. Anything set here is written to the
	// record's "fields" list.
	ac.AddFields(func(ctx iris.Context, f *accesslog.Fields) {
		f.Set("request_id", requestid.Get(ctx))
		if ua := ctx.Header("User-Agent"); ua != "" {
			f.Set("user_agent", ua)
		}
		// The route template groups the dynamic paths that "path" splits
		// apart: /users/{id:int} instead of /users/42. Unmatched requests
		// carry the framework's fallback route, named router.NoRouteName,
		// which would otherwise be reported as "/".
		if route := ctx.CurrentRoute(); route != nil && route.GetName() != router.NoRouteName {
			f.Set("route", route.GetPath())
		}
	})

	return ac
}
