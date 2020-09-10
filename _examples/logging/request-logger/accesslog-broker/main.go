package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
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
	ac.TimeFormat = "2006-01-02 15:04:05"
	// Optionally run logging after response has sent:
	// ac.Async = true
	broker := ac.Broker() // <- IMPORTANT

	app := iris.New()
	app.UseRouter(ac.Handler)

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
