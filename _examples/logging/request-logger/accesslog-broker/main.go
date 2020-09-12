package main

import (
	"os"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/kataras/iris/v12/middleware/recover"
)

func main() {
	/*
		On this example we will make use of the logs broker.
		A handler will listen for any incoming logs and render
		those logs as chunks of JSON to the client (e.g. browser) at real-time.
		Note that this ^ can be done with Server-Sent Events but for the
		sake of the example we'll do it using Transfer-Encoding: chunked.
	*/

	ac := accesslog.File("./access.log")
	defer ac.Close()
	ac.AddOutput(os.Stdout)

	ac.RequestBody = true
	// Set to false to print errors as one line:
	// ac.KeepMultiLineError = false
	// Set the "depth" of a panic trace:
	ac.PanicLog = accesslog.LogHandler // or LogCallers or LogStack

	// Optionally run logging after response has sent:
	// ac.Async = true
	broker := ac.Broker() // <- IMPORTANT

	app := iris.New()
	app.UseRouter(ac.Handler)
	app.UseRouter(recover.New())

	app.OnErrorCode(iris.StatusNotFound, notFoundHandler)

	app.Get("/panic", testPanic)
	app.Get("/", indexHandler)
	app.Get("/profile/{username}", profileHandler)
	app.Post("/read_body", readBodyHandler)

	// register the /logs route,
	// registers a listener and prints the incoming logs.
	// Optionally, skip logging this handler.
	app.Get("/logs", accesslog.SkipHandler, logsHandler(broker))

	// http://localhost:8080/logs to see the logs at real-time.
	app.Listen(":8080")
}

func notFoundHandler(ctx iris.Context) {
	// ctx.Application().Logger().Infof("Not Found Handler for: %s", ctx.Path())

	suggestPaths := ctx.FindClosest(3)
	if len(suggestPaths) == 0 {
		ctx.WriteString("The page you're looking does not exist.")
		return
	}

	ctx.HTML("Did you mean?<ul>")
	for _, s := range suggestPaths {
		ctx.HTML(`<li><a href="%s">%s</a></li>`, s, s)
	}
	ctx.HTML("</ul>")
}

func indexHandler(ctx iris.Context) {
	ctx.HTML("<h1>Index</h1>")
}

func profileHandler(ctx iris.Context) {
	username := ctx.Params().Get("username")
	ctx.HTML("Hello, <strong>%s</strong>!", username)
}

func readBodyHandler(ctx iris.Context) {
	var request interface{}
	if err := ctx.ReadBody(&request); err != nil {
		ctx.StopWithPlainError(iris.StatusBadRequest, err)
		return
	}

	ctx.JSON(iris.Map{"message": "OK", "data": request})
}

func testPanic(ctx iris.Context) {
	panic("PANIC HERE")
}

func logsHandler(b *accesslog.Broker) iris.Handler {
	return func(ctx iris.Context) {
		// accesslog.Skip(ctx) // or inline skip.
		logs := b.NewListener() // <- IMPORTANT

		ctx.Header("Transfer-Encoding", "chunked")
		notifyClose := ctx.Request().Context().Done()
		for {
			select {
			case <-notifyClose:
				b.CloseListener(logs) // <- IMPORTANT

				err := ctx.Request().Context().Err()
				ctx.Application().Logger().Infof("Listener closed [%v], loop end.", err)
				return
			case log := <-logs: // <- IMPORTANT
				ctx.JSON(log, iris.JSON{Indent: "  ", UnescapeHTML: true})
				ctx.ResponseWriter().Flush()
			}
		}
	}
}
