// Package main shows how to send continuous event messages to the clients through server side events via a broker.
// Read more at:
// https://robots.thoughtbot.com/writing-a-server-sent-events-server-in-go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/iris"
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

func (b *Broker) ServeHTTP(ctx iris.Context) {
	// Make sure that the writer supports flushing.
	//
	flusher, ok := ctx.ResponseWriter().(http.Flusher)
	if !ok {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString("Streaming unsupported!")
		return
	}

	// Set the headers related to event streaming, you can omit the "application/json" if you send plain text.
	// If you developer a go client, you must have: "Accept" : "application/json, text/event-stream" header as well.
	ctx.ContentType("application/json, text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	// We also add a Cross-origin Resource Sharing header so browsers on different domains can still connect.
	ctx.Header("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry.
	messageChan := make(chan []byte)

	// Signal the broker that we have a new connection.
	b.newClients <- messageChan

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		b.closingClients <- messageChan
	}()

	// Listen to connection close and un-register messageChan.
	notify := ctx.ResponseWriter().(http.CloseNotifier).CloseNotify()

	go func() {
		<-notify
		b.closingClients <- messageChan
	}()

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
	app.Run(iris.Addr(":8080"))
}
