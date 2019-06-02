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
		websocket.OnNamespaceConnected: func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("[%s] connected to namespace [%s]", c, msg.Namespace)
			return nil
		},
		websocket.OnNamespaceDisconnect: func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("[%s] disconnected from namespace [%s]", c, msg.Namespace)
			return nil
		},
		"chat": func(c *websocket.NSConn, msg websocket.Message) error {
			log.Printf("[%s] sent: %s", c.Conn.ID(), string(msg.Body))

			// Write message back to the client message owner with:
			// c.Emit("chat", msg)
			// Write message to all except this client with:
			c.Conn.Server().Broadcast(c, msg)
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
