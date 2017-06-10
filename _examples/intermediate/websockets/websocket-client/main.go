// Copyright 2017 Joseph deBlaquiere. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"
)

// Peer provides all the handling code which works with either the server or
// client side of the websocket connection, defining a common protocol. Server
// and client terms are a bit of a misnomer in this case because the connection
// is asynchronous and bidirectional. The only difference is that one peer
// listens for incoming connections and the other peer dials the connection.
type Peer struct {
	c websocket.ClientConnection
}

func SetupPeer(client websocket.ClientConnection) (p *Peer) {
	p = new(Peer)
	p.c = client
	p.c.On("message", p.HandleMessage)
	p.c.OnDisconnect(p.HandleDisconnect)
	return p
}

func (p *Peer) HandleMessage(s string) {
	fmt.Printf(" [@ %s] %s", time.Now().Format("15:04:05.000"), s)
}

func (p *Peer) HandleDisconnect() {
	fmt.Println("[peer disconnect... exiting]")
	os.Exit(0)
}

func (p *Peer) Run() {
	fmt.Println("[peer connected... chat away!]")
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		p.c.Emit("message", text)
	}
}

func connect(con websocket.Connection) {
	//Note : once connection established, same as client side: Setup, Run
	//(though callback needs to return so we start a goroutine for Run)
	p := SetupPeer(con)
	go p.Run()
}

func main() {

	// use same config for connection for both listener and dialer
	config := websocket.Config{
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		PingPeriod:     9 * 6 * time.Second,
		PongTimeout:    60 * time.Second,
		BinaryMessages: true,
		Endpoint:       "/chat",
	}

	// try to dial twice before giving up and starting listener
	for tries := 2; tries > 0; tries-- {
		d := new(websocket.Dialer)
		client, _, err := d.Dial("ws://127.0.0.1:8080/chat", nil, config)
		if err == nil {
			//Note : once connection established, same as server: Setup, Run
			p := SetupPeer(client)
			p.Run()
		}
		time.Sleep(1 * time.Second)
	}

	fmt.Println("[No server found, starting listener]")
	fmt.Println("[please start another instance to dial as a peer]")

	// fell through, start app(listener)
	app := iris.New()

	// Attach to the websocket endpoint.
	ws := websocket.New(config)
	ws.OnConnection(connect)
	ws.Attach(app)

	//listen for incoming connection
	app.Run(iris.Server(&http.Server{Addr: ":8080"}), iris.WithoutBanner)
}
