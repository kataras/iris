package main // See https://github.com/kataras/iris/issues/1601

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/kataras/iris/v12/middleware/basicauth"
	"github.com/kataras/iris/v12/middleware/requestid"
	"github.com/kataras/iris/v12/sessions"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

func makeAccessLog() *accesslog.AccessLog {
	// Optionally, let's Go with log rotation.
	pathToAccessLog := "./access_log.%Y%m%d%H%M"
	w, err := rotatelogs.New(
		pathToAccessLog,
		rotatelogs.WithMaxAge(24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour))
	if err != nil {
		panic(err)
	}

	// Initialize a new access log middleware.
	// Accepts an `io.Writer`.
	ac := accesslog.New(w)
	ac.TimeFormat = "2006-01-02 15:04:05"

	// Example of adding more than one field to the logger.
	// Here we logging all the session values this request has.
	//
	// You can also add fields per request handler,
	// look below to the `fieldsHandler` function.
	// Note that this method can override a key stored by a handler's fields.
	ac.AddFields(func(ctx iris.Context, fields *accesslog.Fields) {
		if sess := sessions.Get(ctx); sess != nil {
			fields.Set("session_id", sess.ID())

			sess.Visit(func(k string, v interface{}) {
				fields.Set(k, v)
			})
		}
	})
	// Add a custom field of "auth" when basic auth is available.
	ac.AddFields(func(ctx iris.Context, fields *accesslog.Fields) {
		if username, password, ok := ctx.Request().BasicAuth(); ok {
			fields.Set("auth", username+":"+password)
		}
	})

	return ac

	/*
		Use a file directly:
		ac := accesslog.File("./access.log")

		Log after the response was sent (defaults to false):
		ac.Async = true

		Force-protect writer with locks.
		On this example this is not required:
		ac.LockWriter = true"

		// To disable request and response calculations
		// (enabled by default but slows down the whole operation if Async is false):
		ac.RequestBody = false
		ac.ResponseBody = false
		ac.BytesReceived = false
		ac.BytesSent = false

		Add second output:
		ac.AddOutput(app.Logger().Printer)
		OR:
		accesslog.New(io.MultiWriter(w, os.Stdout))

		Change format (after output was set):
		ac.SetFormatter(&accesslog.JSON{Indent: "  "})

		Modify the output format and customize the order
		with the Template formatter:
		ac.SetFormatter(&accesslog.Template{
		    Text: "{{.Now.Format .TimeFormat}}|{{.Latency}}|{{.Method}}|{{.Path}}|{{.RequestValuesLine}}|{{.Code}}|{{.BytesReceivedLine}}|{{.BytesSentLine}}|{{.Request}}|{{.Response}}|\n",
		    // Default ^
		})
	*/
}

func main() {
	ac := makeAccessLog()

	defer ac.Close()
	iris.RegisterOnInterrupt(func() {
		ac.Close()
	})

	app := iris.New()
	// Register the middleware (UseRouter to catch http errors too).
	app.UseRouter(ac.Handler)
	//

	// Register other middlewares...
	app.UseRouter(requestid.New())

	// Register some routes...
	app.HandleDir("/", iris.Dir("./public"))

	app.Get("/user/{username}", userHandler)
	app.Post("/read_body", readBodyHandler)
	app.Get("/html_response", htmlResponse)

	basicAuth := basicauth.Default(map[string]string{
		"admin": "admin",
	})
	app.Get("/admin", basicAuth, adminHandler)

	sess := sessions.New(sessions.Config{Cookie: "my_session_id", AllowReclaim: true})
	app.Get("/session", sess.Handler(), sessionHandler)

	app.Get("/fields", fieldsHandler)
	//

	app.Listen(":8080")
}

func readBodyHandler(ctx iris.Context) {
	var request interface{}
	if err := ctx.ReadBody(&request); err != nil {
		ctx.StopWithPlainError(iris.StatusBadRequest, err)
		return
	}

	ctx.JSON(iris.Map{"message": "OK", "data": request})
}

func userHandler(ctx iris.Context) {
	ctx.Writef("Hello, %s!", ctx.Params().Get("username"))
}

func htmlResponse(ctx iris.Context) {
	ctx.HTML("<h1>HTML Response</h1>")
}

func adminHandler(ctx iris.Context) {
	username, password, _ := ctx.Request().BasicAuth()
	// of course you don't want that in production:
	ctx.HTML("<h2>Username: %s</h2><h3>Password: %s</h3>", username, password)
}

func sessionHandler(ctx iris.Context) {
	sess := sessions.Get(ctx)
	sess.Set("session_test_key", "session_test_value")

	ctx.WriteString("OK")
}

func fieldsHandler(ctx iris.Context) {
	now := time.Now()
	logFields := accesslog.GetFields(ctx)
	defer func() {
		logFields.Set("fieldsHandler_latency", time.Since(now).Round(time.Second))
	}()

	// simulate a heavy job...
	time.Sleep(2 * time.Second)
	ctx.WriteString("OK")
}
