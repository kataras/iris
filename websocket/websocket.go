/*Package websocket provides rich websocket support for the iris web framework.

Source code and other details for the project are available at GitHub:

   https://github.com/kataras/iris/tree/master/websocket

Installation

    $ go get -u github.com/kataras/iris/websocket


Example code:


	package main

	import (
		"fmt"

		"github.com/kataras/iris"
		"github.com/kataras/iris/context"

		"github.com/kataras/iris/websocket"
	)

	func main() {
		app := iris.New()

		app.Get("/", func(ctx context.Context) {
			ctx.ServeFile("websockets.html", false)
		})

		setupWebsocket(app)

		// x2
		// http://localhost:8080
		// http://localhost:8080
		// write something, press submit, see the result.
		app.Run(iris.Addr(":8080"))
	}

	func setupWebsocket(app *iris.Application) {
		// create our echo websocket server
		ws := websocket.New(websocket.Config{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		})
		ws.OnConnection(handleConnection)

		// register the server's endpoint.
		// see the inline javascript code in the websockets.html,
		// this endpoint is used to connect to the server.
		app.Get("/echo", ws.Handler())

		// serve the javascript built'n client-side library,
		// see websockets.html script tags, this path is used.
		app.Any("/iris-ws.js", func(ctx context.Context) {
			ctx.Write(websocket.ClientSource)
		})
	}

	func handleConnection(c websocket.Connection) {
		// Read events from browser
		c.On("chat", func(msg string) {
			// Print the message to the console
			fmt.Printf("%s sent: %s\n", c.Context().RemoteAddr(), msg)
			// Write message back to the client message owner:
			// c.Emit("chat", msg)
			c.To(websocket.Broadcast).Emit("chat", msg)
		})
	}

*/
package websocket
