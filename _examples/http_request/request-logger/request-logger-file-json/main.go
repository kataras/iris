package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
)

const deleteFileOnExit = false

func newRequestLogger(newWriter io.Writer) iris.Handler {
	c := logger.Config{}

	//	we don't want to use the logger
	// to log requests to assets and etc
	c.AddSkipper(func(ctx iris.Context) bool {
		path := ctx.Path()
		for _, ext := range excludeExtensions {
			if strings.HasSuffix(path, ext) {
				return true
			}
		}
		return false
	})

	c.LogFuncCtx = func(ctx iris.Context, latency time.Duration) {
		datetime := time.Now().Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
		customHandlerMessage := ctx.Values().GetString("log_message")

		file, line := ctx.HandlerFileLine()
		source := fmt.Sprintf("%s:%d", file, line)

		// this will just append a line without an array of javascript objects, readers of this file should read one line per log javascript object,
		// however, you can improve it even more, this is just a simple example on how to use the `LogFuncCtx`.
		jsonStr := fmt.Sprintf(`{"datetime":"%s","level":"%s","source":"%s","latency": "%s","status": %d,"method":"%s","path":"%s","message":"%s"}`,
			datetime, "INFO", source, latency.String(), ctx.GetStatusCode(), ctx.Method(), ctx.Path(), customHandlerMessage)

		fmt.Fprintln(newWriter, jsonStr)
	}

	return logger.New(c)
}

func h(ctx iris.Context) {
	ctx.Values().Set("log_message", "something to give more info to the request logger")

	ctx.Writef("Hello from %s", ctx.Path())
}

func main() {
	app := iris.New()

	logFile := newLogFile()
	defer func() {
		logFile.Close()
		if deleteFileOnExit {
			os.Remove(logFile.Name())
		}
	}()

	r := newRequestLogger(logFile)

	app.Use(r)
	app.OnAnyErrorCode(r, func(ctx iris.Context) {
		ctx.HTML("<h1> Error: Please try <a href ='/'> this </a> instead.</h1>")
	})

	app.Get("/", h)

	app.Get("/1", h)

	app.Get("/2", h)

	app.Get("/", h)

	// http://localhost:8080
	// http://localhost:8080/1
	// http://localhost:8080/2
	// http://lcoalhost:8080/notfoundhere
	app.Listen(":8080", iris.WithoutServerError(iris.ErrServerClosed))
}

var excludeExtensions = [...]string{
	".js",
	".css",
	".jpg",
	".png",
	".ico",
	".svg",
}

// get a filename based on the date, file logs works that way the most times
// but these are just a sugar.
func todayFilename() string {
	today := time.Now().Format("Jan 02 2006")
	return today + ".json"
}

func newLogFile() *os.File {
	filename := todayFilename()
	// open an output file, this will append to the today's file if server restarted.
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	return f
}
