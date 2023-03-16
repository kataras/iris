package main

import (
	"log"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/websocket"
)

type clientPage struct {
	Title string
	Host  string
}

func main() {
	app := iris.New()

	app.RegisterView(iris.HTML("./templates", ".html")) // select the html engine to serve templates

	// Almost all features of neffos are disabled because no custom message can pass
	// when app expects to accept and send only raw websocket native messages.
	// When only allow native messages is a fact?
	// When the registered namespace is just one and it's empty
	// and contains only one registered event which is the `OnNativeMessage`.
	// When `Events{...}` is used instead of `Namespaces{ "namespaceName": Events{...}}`
	// then the namespace is empty "".
	ws := websocket.New(websocket.DefaultGorillaUpgrader, websocket.Events{
		websocket.OnNativeMessage: func(nsConn *websocket.NSConn, msg websocket.Message) error {
			log.Printf("Server got: %s from [%s]", msg.Body, nsConn.Conn.ID())

			nsConn.Conn.Server().Broadcast(nsConn, msg)
			return nil
		},
	})

	ws.OnConnect = func(c *websocket.Conn) error {
		log.Printf("[%s] Connected to server!", c.ID())
		return nil
	}

	ws.OnDisconnect = func(c *websocket.Conn) {
		log.Printf("[%s] Disconnected from server", c.ID())
	}

	app.HandleDir("/js", iris.Dir("./static/js")) // serve our custom javascript code.

	// register the server on an endpoint.
	// see the inline javascript code i the websockets.html, this endpoint is used to connect to the server.
	app.Get("/my_endpoint", websocket.Handler(ws))

	app.Get("/", func(ctx iris.Context) {
		if err := ctx.View("client.html", clientPage{"Client Page", "localhost:8080"}); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	// Target some browser windows/tabs to http://localhost:8080 and send some messages,
	// see the static/js/chat.js,
	// note that the client is using only the browser's native WebSocket API instead of the neffos one.
	app.Listen(":8080")
}
