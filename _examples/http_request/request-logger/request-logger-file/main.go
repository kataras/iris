package main

import (
	"os"
	"strings"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
)

const deleteFileOnExit = true

func main() {
	app := iris.New()
	r, close := newRequestLogger()
	defer close()

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

	// http://localhost:8080
	// http://localhost:8080/1
	// http://localhost:8080/2
	// http://lcoalhost:8080/notfoundhere
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

// get a filename based on the date, file logs works that way the most times
// but these are just a sugar.
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

var excludeExtensions = [...]string{
	".js",
	".css",
	".jpg",
	".png",
	".ico",
	".svg",
}

func newRequestLogger() (h iris.Handler, close func() error) {
	close = func() error { return nil }

	c := logger.Config{
		Status:  true,
		IP:      true,
		Method:  true,
		Path:    true,
		Columns: true,
	}

	logFile := newLogFile()
	close = func() error {
		err := logFile.Close()
		if deleteFileOnExit {
			err = os.Remove(logFile.Name())
		}
		return err
	}

	c.LogFunc = func(now time.Time, latency time.Duration, status, ip, method, path string, message interface{}, headerMessage interface{}) {
		output := logger.Columnize(now.Format("2006/01/02 - 15:04:05"), latency, status, ip, method, path, message, headerMessage)
		logFile.Write([]byte(output))
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

	h = logger.New(c)

	return
}
