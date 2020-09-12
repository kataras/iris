package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
)

// Default line format:
// Time|Latency|Code|Method|Path|IP|Path Params Query Fields|Bytes Received|Bytes Sent|Request|Response|
//
// Read the example and its comments carefully.
func makeAccessLog() *accesslog.AccessLog {
	// Initialize a new access log middleware.
	ac := accesslog.File("./access.log")

	// Defaults to true. Change to false for better performance.
	ac.RequestBody = false
	ac.ResponseBody = false
	ac.BytesReceived = false
	ac.BytesSent = false

	// Defaults to false.
	ac.Async = false

	return ac
}

func main() {
	ac := makeAccessLog()
	defer ac.Close()

	app := iris.New()
	// Register the middleware (UseRouter to catch http errors too).
	app.UseRouter(ac.Handler)

	app.Get("/", indexHandler)

	app.Listen(":8080")
}

func indexHandler(ctx iris.Context) {
	ctx.WriteString("OK")
}
