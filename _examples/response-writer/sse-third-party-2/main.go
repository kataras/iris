package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/alexandrevicenzi/go-sse"
	"github.com/kataras/iris/v12"
)

// Install the sse third-party package.
// $ go get -u github.com/alexandrevicenzi/go-sse
//
// Documentation: https://pkg.go.dev/github.com/alexandrevicenzi/go-sse
func main() {
	s := sse.NewServer(&sse.Options{
		// Increase default retry interval to 10s.
		RetryInterval: 10 * 1000,
		// CORS headers
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, OPTIONS",
			"Access-Control-Allow-Headers": "Keep-Alive,X-Requested-With,Cache-Control,Content-Type,Last-Event-ID",
		},
		// Custom channel name generator
		ChannelNameFunc: func(request *http.Request) string {
			return request.URL.Path
		},
		// Print debug info
		Logger: log.New(os.Stdout, "go-sse: ", log.Ldate|log.Ltime|log.Lshortfile),
	})

	defer s.Shutdown()

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("./index.html")
	})
	app.Get("/events/{channel}", iris.FromStd(s))

	go func() {
		for {
			s.SendMessage("/events/channel-1", sse.SimpleMessage(time.Now().Format("2006/02/01/ 15:04:05")))
			time.Sleep(5 * time.Second)
		}
	}()

	go func() {
		i := 0
		for {
			i++
			s.SendMessage("/events/channel-2", sse.SimpleMessage(strconv.Itoa(i)))
			time.Sleep(5 * time.Second)
		}
	}()

	app.Listen(":3000")
}
