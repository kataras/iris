package main

import (
	"log"

	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"

	"github.com/kataras/neffos"
)

const namespace = "default"

// if namespace is empty then simply neffos.Events{...} can be used instead.
var serverEvents = neffos.Namespaces{
	namespace: neffos.Events{
		neffos.OnNamespaceConnected: func(nsConn *neffos.NSConn, msg neffos.Message) error {
			// with `websocket.GetContext` you can retrieve the Iris' `Context`.
			ctx := websocket.GetContext(nsConn.Conn)

			log.Printf("[%s] connected to namespace [%s] with IP [%s]",
				nsConn, msg.Namespace,
				ctx.RemoteAddr())
			return nil
		},
		neffos.OnNamespaceDisconnect: func(nsConn *neffos.NSConn, msg neffos.Message) error {
			log.Printf("[%s] disconnected from namespace [%s]", nsConn, msg.Namespace)
			return nil
		},
		"chat": func(nsConn *neffos.NSConn, msg neffos.Message) error {
			// room.String() returns -> NSConn.String() returns -> Conn.String() returns -> Conn.ID()
			log.Printf("[%s] sent: %s", nsConn, string(msg.Body))

			// Write message back to the client message owner with:
			// nsConn.Emit("chat", msg)
			// Write message to all except this client with:
			nsConn.Conn.Server().Broadcast(nsConn, msg)
			return nil
		},
	},
}

func main() {
	app := iris.New()
	websocketServer := neffos.New(
		websocket.DefaultGorillaUpgrader, /* DefaultGobwasUpgrader can be used too. */
		serverEvents)

	// serves the endpoint of ws://localhost:8080/echo
	app.Get("/echo", websocket.Handler(websocketServer))

	// serves the browser-based websocket client.
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("./browser/index.html", false)
	})

	// serves the npm browser websocket client usage example.
	app.HandleDir("/browserify", "./browserify")

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}
