package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"

	"github.com/kataras/golog"
)

const deleteFileOnExit = false

func main() {
	app := iris.New()

	logFile := newLogFile()
	defer func() {
		logFile.Close()
		if deleteFileOnExit {
			os.Remove(logFile.Name())
		}
	}()

	// Handle the logs by yourself using the `app.Logger#Handle` method.
	// Return true if that handled, otherwise will print to the screen.
	// You can also use the `app.Logger#SetOutput/AddOutput` to change or add
	// multi (io.Writer) outputs if you just want to print the message
	// somewhere else than the terminal screen.
	app.Logger().Handle(func(l *golog.Log) bool {
		_, fn, line, _ := runtime.Caller(5)

		var (
			// formatted date string based on the `golog#TimeFormat`, which can be customized.
			// Or use the golog.Log#Time field to get the exact time.Time instance.
			datetime = l.FormatTime()
			// the log's message level.
			level = golog.GetTextForLevel(l.Level, false)
			// the log's message.
			message = l.Message
			// the source code line of where it is called,
			// this can differ on your app, see runtime.Caller(%d).
			source = fmt.Sprintf("%s#%d", fn, line)
		)

		// You can always use a custom json structure and json.Marshal and logFile.Write(its result)
		// but it is faster to just build your JSON string by yourself as we do below.
		jsonStr := fmt.Sprintf(`{"datetime":"%s","level":"%s","message":"%s","source":"%s"}`, datetime, level, message, source)
		fmt.Fprintln(logFile, jsonStr)

		/* Example output:
		{"datetime":"2018/10/31 13:13","level":"[INFO]","message":"My server started","source":"c:/mygopath/src/github.com/kataras/iris/_examples/http_request/request-logger/request-logger-file-json/main.go#71"}
		*/
		return true
	})

	r := newRequestLogger()

	app.Use(r)
	app.OnAnyErrorCode(r, func(ctx iris.Context) {
		ctx.HTML("<h1> Error: Please try <a href ='/'> this </a> instead.</h1>")
	})

	h := func(ctx iris.Context) {
		ctx.Writef("Hello from %s", ctx.Path())
	}

	app.Get("/", h)

	app.Get("/1", h)

	app.Get("/2", h)

	app.Logger().Info("My server started")
	// http://localhost:8080
	// http://localhost:8080/1
	// http://localhost:8080/2
	// http://lcoalhost:8080/notfoundhere
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

var excludeExtensions = [...]string{
	".js",
	".css",
	".jpg",
	".png",
	".ico",
	".svg",
}

func newRequestLogger() iris.Handler {
	c := logger.Config{
		Status: true,
		IP:     true,
		Method: true,
		Path:   true,
	}

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

	return logger.New(c)
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
