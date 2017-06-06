package main

import (
	"fmt"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"

	"github.com/kataras/iris/websocket"
)

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

func main() {
	app := iris.New()

	// create our echo websocket server
	ws := websocket.New(websocket.Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Endpoint:        "/echo",
	})

	ws.OnConnection(handleConnection)

	// Adapt the application to the websocket server.
	ws.Attach(app)

	app.Get("/", func(ctx context.Context) {
		ctx.ServeFile("websockets.html", false) // second parameter: enable gzip?
	})

	// x2
	// http://localhost:8080
	// http://localhost:8080
	// write something, press submit, see the result.
	app.Run(iris.Addr(":8080"))
}
