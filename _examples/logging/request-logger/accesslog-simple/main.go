package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
)

// Read the example and its comments carefully.
func makeAccessLog() *accesslog.AccessLog {
	// Initialize a new access log middleware.
	ac := accesslog.File("./access.log")

	// The default configuration:
	ac.Delim = '|'
	ac.TimeFormat = "2006-01-02 15:04:05"
	ac.Async = false
	ac.IP = true
	ac.BytesReceivedBody = true
	ac.BytesSentBody = true
	ac.BytesReceived = false
	ac.BytesSent = false
	ac.BodyMinify = true
	ac.RequestBody = true
	ac.ResponseBody = false
	ac.KeepMultiLineError = true
	ac.PanicLog = accesslog.LogHandler

	// Default line format if formatter is missing:
	// Time|Latency|Code|Method|Path|IP|Path Params Query Fields|Bytes Received|Bytes Sent|Request|Response|
	//
	// Set Custom Formatter:
	ac.SetFormatter(&accesslog.JSON{
		Indent:    "  ",
		HumanTime: true,
	})
	// ac.SetFormatter(&accesslog.CSV{})
	// ac.SetFormatter(&accesslog.Template{Text: "{{.Code}}"})

	return ac
}

func main() {
	ac := makeAccessLog()
	defer ac.Close() // Close the underline file.

	app := iris.New()
	// Register the middleware (UseRouter to catch http errors too).
	app.UseRouter(ac.Handler)

	app.Get("/", indexHandler)

	app.Listen(":8080")
}

func indexHandler(ctx iris.Context) {
	ctx.WriteString("OK")
}
