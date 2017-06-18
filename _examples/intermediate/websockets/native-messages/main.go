package main

import (
	"fmt"

	"github.com/cdren/iris"
	"github.com/cdren/iris/context"

	"github.com/cdren/iris/view"
	"github.com/cdren/iris/websocket"
)

/* Native messages no need to import the iris-ws.js to the ./templates.client.html
Use of: OnMessage and EmitMessage.


NOTICE: IF YOU HAVE RAN THE PREVIOUS EXAMPLES YOU HAVE TO CLEAR YOUR BROWSER's CACHE
BECAUSE chat.js is different than the CACHED. OTHERWISE YOU WILL GET Ws is undefined from the browser's console, becuase it will use the cached.
*/

type clientPage struct {
	Title string
	Host  string
}

func main() {
	app := iris.New()

	app.AttachView(view.HTML("./templates", ".html")) // select the html engine to serve templates

	ws := websocket.New(websocket.Config{
		// the path which the websocket client should listen/registered to,
		Endpoint: "/my_endpoint",
		// to enable binary messages (useful for protobuf):
		// BinaryMessages: true,
	})

	ws.Attach(app) // adapt the websocket server, you can adapt more than one with different Endpoint

	app.StaticWeb("/js", "./static/js") // serve our custom javascript code

	app.Get("/", func(ctx context.Context) {
		ctx.ViewData("", clientPage{"Client Page", "localhost:8080"})
		ctx.View("client.html")
	})

	ws.OnConnection(func(c websocket.Connection) {

		c.OnMessage(func(data []byte) {
			message := string(data)
			c.To(websocket.Broadcast).EmitMessage([]byte("Message from: " + c.ID() + "-> " + message)) // broadcast to all clients except this
			c.EmitMessage([]byte("Me: " + message))                                                    // writes to itself
		})

		c.OnDisconnect(func() {
			fmt.Printf("\nConnection with ID: %s has been disconnected!", c.ID())
		})

	})

	app.Run(iris.Addr(":8080"))

}
