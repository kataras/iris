// Package main shows how to send continuous event messages to the clients through server side events via a broker.
// Read more at:
// https://robots.thoughtbot.com/writing-a-server-sent-events-server-in-go
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/iris"
	// Note:
	// For some reason the latest vscode-go language extension does not provide enough intelligence (parameters documentation and go to definition features)
	// for the `iris.Context` alias, therefore if you use VS Code, import the original import path of the `Context`, that will do it:
	"github.com/kataras/iris/context"
)

// A Broker holds open client connections,
// listens for incoming events on its Notifier channel
// and broadcast event data to all registered connections.
type Broker struct {

	// Events are pushed to this channel by the main events-gathering routine.
	Notifier chan []byte

	// New client connections.
	newClients chan chan []byte

	// Closed client connections.
	closingClients chan chan []byte

	// Client connections registry.
	clients map[chan []byte]bool
}

// NewBroker returns a new broker factory.
func NewBroker() *Broker {
	b := &Broker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	// Set it running - listening and broadcasting events.
	go b.listen()

	return b
}

// Listen on different channels and act accordingly.
func (b *Broker) listen() {
	for {
		select {
		case s := <-b.newClients:
			// A new client has connected.
			// Register their message channel.
			b.clients[s] = true

			golog.Infof("Client added. %d registered clients", len(b.clients))

		case s := <-b.closingClients:
			// A client has dettached and we want to
			// stop sending them messages.
			delete(b.clients, s)
			golog.Warnf("Removed client. %d registered clients", len(b.clients))

		case event := <-b.Notifier:
			// We got a new event from the outside!
			// Send event to all connected clients.
			for clientMessageChan := range b.clients {
				clientMessageChan <- event
			}
		}
	}
}

func (b *Broker) ServeHTTP(ctx context.Context) {
	// Make sure that the writer supports flushing.

	flusher, ok := ctx.ResponseWriter().Flusher()
	if !ok {
		ctx.StatusCode(iris.StatusHTTPVersionNotSupported)
		ctx.WriteString("Streaming unsupported!")
		return
	}

	// Set the headers related to event streaming, you can omit the "application/json" if you send plain text.
	// If you develop a go client, you must have: "Accept" : "application/json, text/event-stream" header as well.
	ctx.ContentType("application/json, text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	// We also add a Cross-origin Resource Sharing header so browsers on different domains can still connect.
	ctx.Header("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry.
	messageChan := make(chan []byte)

	// Signal the broker that we have a new connection.
	b.newClients <- messageChan

	// Listen to connection close or when the entire request handler chain exits and un-register messageChan.
	// using the `ctx.ResponseWriter().CloseNotifier()` and `defer` for this single handler of the route:
	/*
		notifier, ok := ctx.ResponseWriter().CloseNotifier()
		if ok {
			go func() {
				<-notifier.CloseNotify()
				b.closingClients <- messageChan
			}()
		}

		defer func() {
			b.closingClients <- messageChan
		}()
	*/
	// or by using the `ctx.OnClose`, which will take care all of the above for you:
	ctx.OnClose(func() {
		// Remove this client from the map of connected clients
		// when this handler exits.
		b.closingClients <- messageChan
	})

	// block waiting for messages broadcast on this connection's messageChan.
	for {
		// Write to the ResponseWriter.
		// Server Sent Events compatible.
		ctx.Writef("data:%s\n\n", <-messageChan)
		// or json: data:{obj}.

		// Flush the data immediatly instead of buffering it for later.
		flusher.Flush()
	}
}

type event struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

func main() {
	broker := NewBroker()

	go func() {
		for {
			time.Sleep(2 * time.Second)

			now := time.Now()
			evt := event{
				Timestamp: now.Unix(),
				Message:   fmt.Sprintf("the time is %v", now.Format(time.RFC1123)),
			}
			evtBytes, err := json.Marshal(evt)
			if err != nil {
				golog.Errorf("receiving event failure: %v", err)
				continue
			}

			golog.Infof("Receiving event")
			broker.Notifier <- evtBytes
		}
	}()

	// Iris web server.
	app := iris.New()
	app.Get("/", broker.ServeHTTP)

	// http://localhost:8080
	// TIP: If you make use of it inside a web frontend application
	// then checkout the "optional.sse.js.html" to use the javascript's API for SSE,
	// it will also remove the browser's "loading" indicator while receiving those event messages.
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}
