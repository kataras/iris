package main

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket2"
)

const totalClients = 1200

func main() {
	app := iris.New()

	// websocket.Config{PingPeriod: ((60 * time.Second) * 9) / 10}
	ws := websocket.New(websocket.Config{})
	ws.OnConnection(handleConnection)
	app.Get("/socket", ws.Handler())

	go func() {
		t := time.NewTicker(2 * time.Second)
		for {
			<-t.C

			conns := ws.GetConnections()
			for _, conn := range conns {
				// fmt.Println(conn.ID())
				// Do nothing.
				_ = conn
			}

			if atomic.LoadUint64(&count) == totalClients {
				fmt.Println("ALL CLIENTS DISCONNECTED SUCCESSFULLY.")
				t.Stop()
				os.Exit(0)
				return
			}
		}
	}()

	app.Run(iris.Addr(":8080"))
}

func handleConnection(c websocket.Connection) {
	c.OnError(func(err error) { handleErr(c, err) })
	c.OnDisconnect(func() { handleDisconnect(c) })
	c.On("chat", func(message string) {
		c.To(websocket.Broadcast).Emit("chat", c.ID()+": "+message)
	})
}

var count uint64

func handleDisconnect(c websocket.Connection) {
	atomic.AddUint64(&count, 1)
	fmt.Printf("client [%s] disconnected!\n", c.ID())
}

func handleErr(c websocket.Connection, err error) {
	fmt.Printf("client [%s] errored: %v\n", c.ID(), err)
}
