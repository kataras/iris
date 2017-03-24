package main

import (
	"log"
	"os"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

var myLogFile *os.File

func init() {
	// open an output file
	f, err := os.Create("logs.txt")
	if err != nil {
		panic(err)
	}
	myLogFile = f
}

func myFileLogger() iris.LoggerPolicy {

	// you can use a *File or an io.Writer,
	// we want to log with timestamps so we use the log.New.
	myLogger := log.New(myLogFile, "", log.LstdFlags)

	// the logger is just a func,
	// will be used in runtime
	return func(mode iris.LogMode, message string) {
		// optionally, check for production or development log message mode
		// two modes: iris.ProdMode and iris.DevMode
		if mode == iris.ProdMode {
			// log only production-mode log messages
			myLogger.Println(message)
		}
	}
}

func main() {
	// close the log file on exit application
	// when panic or iris exited by interupt event or manually by Shutdown.
	defer func() {
		if err := myLogFile.Close(); err != nil {
			panic(err)
		}
	}()

	app := iris.New()
	app.Adapt(myFileLogger())
	app.Adapt(httprouter.New())

	app.Get("/", func(ctx *iris.Context) {
		// for the sake of simplicity, in order see the logs at the ./logs.txt:
		app.Log(iris.ProdMode, "You have requested: http://localhost/8080"+ctx.Path())

		ctx.Writef("hello")
	})

	// open http://localhost:8080
	// and watch the ./logs.txt file
	app.Listen(":8080")
}
