package main

import (
	"os"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

// get a filename based on the date, file logs works that way the most times
// but these are just a sugar, you can directly attach a new file logger with .AttachLogger(io.Writer)
func todayFilename() string {
	today := time.Now().Format("Jan 02 2006")
	return today + ".txt"
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

func main() {
	f := newLogFile()
	defer f.Close()

	app := iris.New()
	// attach the file as logger, remember, iris' app logger is just an io.Writer.
	app.AttachLogger(f)

	app.Get("/", func(ctx context.Context) {
		// for the sake of simplicity, in order see the logs at the ./_today_.txt
		ctx.Application().Log("Request: %s\r\n", ctx.Path())
		ctx.Writef("hello")
	})

	// navigate to http://localhost:8080
	// and open the ./logs.txt file
	if err := app.Run(iris.Addr(":8080"), iris.WithoutBanner); err != nil {
		app.Log("Shutdown with error: %v", err)

	}
}
