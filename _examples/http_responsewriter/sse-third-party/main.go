package main

import (
	"time"

	"github.com/kataras/iris"
	"github.com/r3labs/sse"
)

// First of all install the sse third-party package (you can use other if you don't like this approach or go ahead to the "sse" example)
// $ go get -u github.com/r3labs/sse
func main() {
	app := iris.New()
	s := sse.New()
	/*
		This creates a new stream inside of the scheduler.
		Seeing as there are no consumers, publishing a message
		to this channel will do nothing.
		Clients can connect to this stream once the iris handler is started
		by specifying stream as a url parameter, like so:
		http://localhost:8080/events?stream=messages
	*/
	s.CreateStream("messages")

	app.Any("/events", iris.FromStd(s.HTTPHandler))

	go func() {
		// You design when to send messages to the client,
		// here we just wait 5 seconds to send the first message
		// in order to give u time to open a browser window...
		time.Sleep(5 * time.Second)
		// Publish a payload to the stream.
		s.Publish("messages", &sse.Event{
			Data: []byte("ping"),
		})

		time.Sleep(3 * time.Second)
		s.Publish("messages", &sse.Event{
			Data: []byte("second message"),
		})
		time.Sleep(2 * time.Second)
		s.Publish("messages", &sse.Event{
			Data: []byte("third message"),
		})

	}() // ...

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

/* For a golang SSE client you can look at: https://github.com/r3labs/sse#example-client */
