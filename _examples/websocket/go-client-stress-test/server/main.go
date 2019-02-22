package main

import (
	"log"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"
)

const totalClients = 16000 // max depends on the OS.
const verbose = true

func main() {

	ws := websocket.New(websocket.Config{})
	ws.OnConnection(handleConnection)

	// websocket.Config{PingPeriod: ((60 * time.Second) * 9) / 10}

	go func() {
		dur := 8 * time.Second
		if totalClients >= 64000 {
			// if more than 64000 then let's no check every 8 seconds, let's do it every 24 seconds,
			// just for simplicity, either way works.
			dur = 24 * time.Second
		}
		t := time.NewTicker(dur)
		defer t.Stop()
		defer os.Exit(0)
		defer runtime.Goexit()

		var started bool
		for {
			<-t.C

			n := ws.GetTotalConnections()
			if n > 0 {
				started = true
			}

			if started {
				totalConnected := atomic.LoadUint64(&count)

				if totalConnected == totalClients {
					if n != 0 {
						log.Println("ALL CLIENTS DISCONNECTED BUT LEFTOVERS ON CONNECTIONS LIST.")
					} else {
						log.Println("ALL CLIENTS DISCONNECTED SUCCESSFULLY.")
					}
					return
				} else if n == 0 {
					log.Printf("%d/%d CLIENTS WERE NOT CONNECTED AT ALL. CHECK YOUR OS NET SETTINGS. ALL OTHER CONNECTED CLIENTS DISCONNECTED SUCCESSFULLY.\n",
						totalClients-totalConnected, totalClients)

					return
				}
			}
		}
	}()

	app := iris.New()
	app.Get("/", ws.Handler())
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
	if verbose {
		log.Printf("client [%s] disconnected!\n", c.ID())
	}
}

func handleErr(c websocket.Connection, err error) {
	log.Printf("client [%s] errored: %v\n", c.ID(), err)
}
