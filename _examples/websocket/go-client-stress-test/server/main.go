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

const (
	endpoint     = "localhost:8080"
	totalClients = 16000 // max depends on the OS.
	verbose      = false
	maxC         = 0
)

func main() {
	ws := websocket.New(websocket.Config{})
	ws.OnConnection(handleConnection)

	// websocket.Config{PingPeriod: ((60 * time.Second) * 9) / 10}

	go func() {
		dur := 4 * time.Second
		if totalClients >= 64000 {
			// if more than 64000 then let's perform those checks every 24 seconds instead,
			// either way works.
			dur = 24 * time.Second
		}
		t := time.NewTicker(dur)
		defer func() {
			t.Stop()
			printMemUsage()
			os.Exit(0)
		}()

		var started bool
		for {
			<-t.C

			n := ws.GetTotalConnections()
			if n > 0 {
				started = true
				if maxC > 0 && n > maxC {
					log.Printf("current connections[%d] > MaxConcurrentConnections[%d]", n, maxC)
					return
				}
			}

			if started {
				disconnectedN := atomic.LoadUint64(&totalDisconnected)
				connectedN := atomic.LoadUint64(&totalConnected)
				if disconnectedN == totalClients && connectedN == totalClients {
					if n != 0 {
						log.Println("ALL CLIENTS DISCONNECTED BUT LEFTOVERS ON CONNECTIONS LIST.")
					} else {
						log.Println("ALL CLIENTS DISCONNECTED SUCCESSFULLY.")
					}
					return
				} else if n == 0 {
					log.Printf("%d/%d CLIENTS WERE NOT CONNECTED AT ALL. CHECK YOUR OS NET SETTINGS. THE REST CLIENTS WERE DISCONNECTED SUCCESSFULLY.\n",
						totalClients-totalConnected, totalClients)

					return
				}
			}
		}
	}()

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		c := ws.Upgrade(ctx)
		handleConnection(c)
		c.Wait()
	})
	app.Run(iris.Addr(endpoint), iris.WithoutServerError(iris.ErrServerClosed))
}

var totalConnected uint64

func handleConnection(c websocket.Connection) {
	if c.Err() != nil {
		log.Fatalf("[%d] upgrade failed: %v", atomic.LoadUint64(&totalConnected)+1, c.Err())
		return
	}

	atomic.AddUint64(&totalConnected, 1)
	c.OnError(func(err error) { handleErr(c, err) })
	c.OnDisconnect(func() { handleDisconnect(c) })
	c.On("chat", func(message string) {
		c.To(websocket.Broadcast).Emit("chat", c.ID()+": "+message)
	})
}

var totalDisconnected uint64

func handleDisconnect(c websocket.Connection) {
	newC := atomic.AddUint64(&totalDisconnected, 1)
	if verbose {
		log.Printf("[%d] client [%s] disconnected!\n", newC, c.ID())
	}
}

func handleErr(c websocket.Connection, err error) {
	log.Printf("client [%s] errorred: %v\n", c.ID(), err)
}

func toMB(b uint64) uint64 {
	return b / 1024 / 1024
}

func printMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("Alloc = %v MiB", toMB(m.Alloc))
	log.Printf("\tTotalAlloc = %v MiB", toMB(m.TotalAlloc))
	log.Printf("\tSys = %v MiB", toMB(m.Sys))
	log.Printf("\tNumGC = %v\n", m.NumGC)
	log.Printf("\tNumGoRoutines = %d\n", runtime.NumGoroutine())
}
