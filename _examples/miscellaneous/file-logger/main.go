package main

import (
	"os"
	"time"

	"github.com/kataras/iris"
)

// Get a filename based on the date, just for the sugar.
func todayFilename() string {
	today := time.Now().Format("Jan 02 2006")
	return today + ".txt"
}

func newLogFile() *os.File {
	filename := todayFilename()
	// Open the file, this will append to the today's file if server restarted.
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	return f
}

func main() {
	f := newLogFile()
	defer f.Close()

	app := iris.New()
	// Attach the file as logger, remember, iris' app logger is just an io.Writer.
	// Use the following code if you need to write the logs to file and console at the same time.
	// app.Logger().SetOutput(io.MultiWriter(f, os.Stdout))
	app.Logger().SetOutput(f)

	app.Get("/ping", func(ctx iris.Context) {
		// for the sake of simplicity, in order see the logs at the ./_today_.txt
		ctx.Application().Logger().Infof("Request path: %s", ctx.Path())
		ctx.WriteString("pong")
	})

	// Navigate to http://localhost:8080/ping
	// and open the ./logs{TODAY}.txt file.
	if err := app.Run(iris.Addr(":8080"), iris.WithoutBanner, iris.WithoutServerError(iris.ErrServerClosed)); err != nil {
		app.Logger().Warn("Shutdown with error: " + err.Error())
	}
}
