package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/kataras/iris"

	"github.com/kataras/iris/websocket"
)

type clientPage struct {
	Title string
	Host  string
}

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./templates", ".html")) // select the html engine to serve templates

	ws := websocket.New(websocket.Config{})

	// register the server on an endpoint.
	// see the inline javascript code i the websockets.html, this endpoint is used to connect to the server.
	app.Get("/my_endpoint", ws.Handler())

	// serve the javascript builtin client-side library,
	// see websockets.html script tags, this path is used.
	app.Any("/iris-ws.js", func(ctx iris.Context) {
		ctx.Write(websocket.ClientSource)
	})

	app.StaticWeb("/js", "./static/js") // serve our custom javascript code

	app.Get("/", func(ctx iris.Context) {
		ctx.ViewData("", clientPage{"Client Page", "localhost:8080"})
		ctx.View("client.html")
	})

	Conn := make(map[websocket.Connection]bool)
	var myChatRoom = "room1"
	var mutex = new(sync.Mutex)

	ws.OnConnection(func(c websocket.Connection) {
		c.Join(myChatRoom)
		mutex.Lock()
		Conn[c] = true
		mutex.Unlock()
		c.On("chat", func(message string) {
			if message == "leave" {
				c.Leave(myChatRoom)
				c.To(myChatRoom).Emit("chat", "Client with ID: "+c.ID()+" left from the room and cannot send or receive message to/from this room.")
				c.Emit("chat", "You have left from the room: "+myChatRoom+" you cannot send or receive any messages from others inside that room.")
				return
			}
		})
		c.OnDisconnect(func() {
			mutex.Lock()
			delete(Conn, c)
			mutex.Unlock()
			fmt.Printf("\nConnection with ID: %s has been disconnected!\n", c.ID())
		})
	})

	var delay = 1 * time.Second
	go func() {
		i := 0
		for {
			mutex.Lock()
			broadcast(Conn, fmt.Sprintf("aaaa %d\n", i))
			mutex.Unlock()
			time.Sleep(delay)
			i++
		}
	}()

	go func() {
		i := 0
		for range time.Tick(1 * time.Second) { //another way to get clock signal
			mutex.Lock()
			broadcast(Conn, fmt.Sprintf("aaaa2 %d\n", i))
			mutex.Unlock()
			time.Sleep(delay)
			i++
		}
	}()

	app.Run(iris.Addr(":8080"))
}

func broadcast(Conn map[websocket.Connection]bool, message string) {
	for k := range Conn {
		k.To("room1").Emit("chat", message)
	}
}
