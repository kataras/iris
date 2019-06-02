package main

import (
	"log"

	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"
)

const namespace = "default"

// if namespace is empty then simply websocket.Events{...} can be used instead.
var serverEvents = websocket.Namespaces{
	namespace: websocket.Events{
		websocket.OnNamespaceConnected: func(nsConn *websocket.NSConn, msg websocket.Message) error {
			log.Printf("[%s] connected to namespace [%s] with IP [%s]",
				nsConn, msg.Namespace,
				// with `GetContext` you can retrieve the Iris' `Context`, alternatively
				// you can use the `nsConn.Conn.Socket().Request()` to get the raw `*http.Request`.
				websocket.GetContext(nsConn.Conn).RemoteAddr())
			return nil
		},
		websocket.OnNamespaceDisconnect: func(nsConn *websocket.NSConn, msg websocket.Message) error {
			log.Printf("[%s] disconnected from namespace [%s]", nsConn, msg.Namespace)
			return nil
		},
		"chat": func(nsConn *websocket.NSConn, msg websocket.Message) error {
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
	websocketServer := websocket.New(
		websocket.DefaultGorillaUpgrader, /*DefaultGobwasUpgrader can be used as well*/
		serverEvents)

	// serves the endpoint of ws://localhost:8080/echo
	app.Get("/echo", websocket.Handler(websocketServer))

	// serves the browser-based websocket client.
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("./browser/index.html", false)
	})

	// serves the npm browser websocket client usage example.
	app.StaticWeb("/browserify", "./browserify")

	app.Run(iris.Addr(":8080"))
}
