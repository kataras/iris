// A chat room over WebSockets.
//
// Iris websockets are built on neffos, which adds namespaces and named events
// on top of the raw socket: a client dials, connects to a namespace, and from
// then on both sides emit events with a body. The server side is a map of
// event names to handlers; a handler gets the connection it came from and
// the message, and it can answer, broadcast to the rest of the namespace or
// join rooms. Because of that protocol the browser uses the neffos.js client
// (loaded from a CDN below) rather than a bare WebSocket object.
//
// websocket.New(server, options) is the Iris handler that upgrades the
// request; it must be the last handler on its route. websocket.GetContext
// gives the upgrade request's Context back inside an event, for the remote
// address, a cookie or the authenticated user.
//
// Learn: serve a websocket endpoint, handle connect and custom events, and
// broadcast a message to everyone else in the namespace.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	Open http://localhost:8080 in two browser tabs and type in either one.      # each tab sees the other's messages and joins
//	curl -i localhost:8080/ws                                                   # 400: not a websocket upgrade
//
// Next: ../../14-testing/iris-httptest
package main

import (
	_ "embed"
	"log"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/websocket"
	"github.com/kataras/neffos"
	"github.com/kataras/neffos/gorilla"
)

//go:embed index.html
var page string

const namespace = "default"

func main() {
	server := neffos.New(gorilla.DefaultUpgrader, neffos.Namespaces{
		namespace: neffos.Events{
			// Fired when a client connects to the namespace.
			neffos.OnNamespaceConnected: func(c *neffos.NSConn, msg neffos.Message) error {
				ctx := websocket.GetContext(c.Conn) // the upgrade request, read-only.
				log.Printf("[%s] joined from %s", c, ctx.RemoteAddr())

				c.BroadcastOthers(neffos.Message{Namespace: namespace, Event: "chat", Body: []byte(c.String() + " joined")})
				return nil
			},
			neffos.OnNamespaceDisconnect: func(c *neffos.NSConn, msg neffos.Message) error {
				log.Printf("[%s] left", c)
				return nil
			},
			// A custom event. The browser emits "chat" with the typed text; the
			// server relays it to everyone else in the namespace.
			"chat": func(c *neffos.NSConn, msg neffos.Message) error {
				log.Printf("[%s] says: %s", c, msg.Body)
				msg.Body = []byte(c.String() + ": " + string(msg.Body))
				c.BroadcastOthers(msg)
				return nil
			},
		},
	})
	defer server.Close()

	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML(page)
	})

	// The websocket handler is last on its route; anything before it (an
	// authentication guard, for instance) runs on the upgrade request.
	app.Any("/ws", websocket.New(server, websocket.Options{}))

	app.Listen(":8080")
}
