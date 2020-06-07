package main

import (
	"encoding/json"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/requestid"

	"github.com/kataras/golog"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")
	app.Logger().Handle(jsonOutput)

	app.Use(requestid.New())

	/* Example Output:
	{
	    "timestamp": 1591422944,
	    "level": "debug",
	    "message": "This is a message with data",
	    "fields": {
	        "username": "kataras"
	    },
	    "stacktrace": [
	        {
	            "function": "main.main",
	            "source": "C:/mygopath/src/github.com/kataras/iris/_examples/logging/json-logger/main.go:16"
	        }
	    ]
	}
	*/
	app.Logger().Debugf("This is a %s with data (debug prints the stacktrace too)", "message", golog.Fields{
		"username": "kataras",
	})

	/* Example Output:
	{
	    "timestamp": 1591422944,
	    "level": "info",
	    "message": "An info message",
	    "fields": {
	        "home": "https://iris-go.com"
	    }
	}
	*/
	app.Logger().Infof("An info message", golog.Fields{"home": "https://iris-go.com"})

	app.Get("/ping", ping)

	// Navigate to http://localhost:8080/ping.
	app.Listen(":8080" /*, iris.WithoutBanner*/)
}

func jsonOutput(l *golog.Log) bool {
	enc := json.NewEncoder(l.Logger.Printer) // you can change the output to a file as well.
	enc.SetIndent("", "    ")
	err := enc.Encode(l)
	return err == nil
}

func ping(ctx iris.Context) {
	/* Example Output:
	{
	    "timestamp": 1591423046,
	    "level": "debug",
	    "message": "Request path: /ping",
	    "fields": {
	        "request_id": "fc12d88a-a338-4bb9-aa5e-126f2104365c"
	    },
	    "stacktrace": [
	        {
	            "function": "main.ping",
	            "source": "C:/mygopath/src/github.com/kataras/iris/_examples/logging/json-logger/main.go:82"
	        },
	       ...
	    ]
	}
	*/
	ctx.Application().Logger().Debugf("Request path: %s", ctx.Path(), golog.Fields{
		"request_id": ctx.GetID(),
	})

	ctx.WriteString("pong")
}
