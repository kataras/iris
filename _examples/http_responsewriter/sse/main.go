// Package main shows how to send continuous event messages to the clients through SSE via a broker.
// Read details at: https://www.w3schools.com/htmL/html5_serversentevents.asp and
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

	// Listen to connection close and when the entire request handler chain exits(this handler here) and un-register messageChan.
	ctx.OnClose(func() {
		// Remove this client from the map of connected clients
		// when this handler exits.
		b.closingClients <- messageChan
	})

	// Block waiting for messages broadcast on this connection's messageChan.
	for {
		// Write to the ResponseWriter.
		// Server Sent Events compatible.
		ctx.Writef("data: %s\n\n", <-messageChan)
		// or json: data:{obj}.

		// Flush the data immediately instead of buffering it for later.
		flusher.Flush()
	}
}

type event struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

const script = `<script type="text/javascript">
if(typeof(EventSource) !== "undefined") {
	console.log("server-sent events supported");
	var client = new EventSource("http://localhost:8080/events");
	var index = 1;
	client.onmessage = function (evt) {
		console.log(evt);
		// it's not required that you send and receive JSON, you can just output the "evt.data" as well.
		dataJSON = JSON.parse(evt.data)
		var table = document.getElementById("messagesTable");
		var row = table.insertRow(index);
		var cellTimestamp = row.insertCell(0);
		var cellMessage = row.insertCell(1);
		cellTimestamp.innerHTML = dataJSON.timestamp;
		cellMessage.innerHTML = dataJSON.message;
		index++;

		window.scrollTo(0,document.body.scrollHeight);
	};
} else {
	document.getElementById("header").innerHTML = "<h2>SSE not supported by this client-protocol</h2>";
}
</script>`

func main() {
	broker := NewBroker()

	go func() {
		for {
			time.Sleep(2 * time.Second)

			now := time.Now()
			evt := event{
				Timestamp: now.Unix(),
				Message:   fmt.Sprintf("Hello at %s", now.Format(time.RFC1123)),
			}

			evtBytes, err := json.Marshal(evt)
			if err != nil {
				golog.Error(err)
				continue
			}

			broker.Notifier <- evtBytes
		}
	}()

	app := iris.New()
	app.Get("/", func(ctx context.Context) {
		ctx.HTML(
			`<html><head><title>SSE</title>` + script + `</head>
				<body>
					<h1 id="header">Waiting for messages...</h1>
					<table id="messagesTable" border="1">
						<tr>
							<th>Timestamp (server)</th>
							<th>Message</th>
						</tr>
					</table>
				</body>
			 </html>`)
	})

	app.Get("/events", broker.ServeHTTP)

	// http://localhost:8080
	// http://localhost:8080/events
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}
