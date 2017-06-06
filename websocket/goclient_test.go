// The MIT License (MIT)
//
// Copyright (c) 2017, Joseph deBlaquiere <jadeblaquiere@yahoo.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package websocket

import (
	stdContext "context"
	"encoding/binary"
	// "encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/host"
	"github.com/kataras/iris/view"
	//"github.com/kataras/iris/websocket"
)

// test server model used to test client code

type indexResponse struct {
	RequestIP string `json:"request_ip"`
	Time      int64  `json:"unix_time"`
}

type wsClient struct {
	con Connection
	wss *wsServer
}

func (wsc *wsClient) echoRawMessage(message []byte) {
	// fmt.Println("recv :", string(message))
	wsc.con.EmitMessage(message)
}

func (wsc *wsClient) echoString(message string) {
	// fmt.Println("recv :", message)
	wsc.con.Emit("echo_reply", message)
}

func (wsc *wsClient) lenString(message string) {
	// fmt.Println("recv :", message)
	wsc.con.Emit("len_reply", len(message))
}

func (wsc *wsClient) disconnect() {
	fmt.Println("client disconnect @ ", time.Now().Format("2006-01-02 15:04:05.000000"))
}

type wsServer struct {
	clients   []*wsClient
	listMutex sync.Mutex
	app       *iris.Application
	srv       *http.Server
	super     *host.Supervisor
	ws        Server
}

func (wss *wsServer) connect(con Connection) {
	wss.listMutex.Lock()
	defer wss.listMutex.Unlock()

	c := &wsClient{con: con, wss: wss}
	wss.clients = append(wss.clients, c)

	// fmt.Printf("Connect # active clients : %d\n", len(wss.clients))

	con.OnMessage(c.echoRawMessage)

	con.On("echo", c.echoString)

	con.On("len", c.lenString)

	con.OnDisconnect(c.disconnect)
}

func (wss *wsServer) disconnect(wsc *wsClient) {
	wss.listMutex.Lock()
	defer wss.listMutex.Unlock()

	l := len(wss.clients)

	if l == 0 {
		panic("WSS:trying to delete client from empty list")
	}

	for p, v := range wss.clients {
		if v == wsc {
			wss.clients[p] = wss.clients[l-1]
			wss.clients = wss.clients[:l-1]

			fmt.Printf("Disconnect # active clients : %d\n", len(wss.clients))

			return
		}
	}
	panic("WSS:trying to delete client not in list")
}

func (wss *wsServer) index(ctx context.Context) {
	t := time.Now().Unix()
	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(indexResponse{RequestIP: ctx.RemoteAddr(), Time: t})
}

func (wss *wsServer) startup() {
	wss.app = iris.New()
	wss.app.AttachView(view.HTML("./templates", ".html"))
	wss.app.Get("/", wss.index)
	// create our echo websocket server
	ws := New(Config{
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		BinaryMessages:  true,
		Endpoint:        "/echo",
	})

	ws.OnConnection(wss.connect)

	// Attach the websocket server.
	ws.Attach(wss.app)

	wss.srv = &http.Server{Addr: ":8080"}
	wss.super = host.New(wss.srv)

	go wss.app.Run(iris.Server(wss.srv), iris.WithoutBanner)
}

func (wss *wsServer) shutdown() {
	ctx, _ := stdContext.WithTimeout(stdContext.Background(), 5*time.Second)
	wss.super.Shutdown(ctx)
}

func TestConnectAndWait(t *testing.T) {
	var wss wsServer
	wss.startup()
	time.Sleep(1 * time.Second)
	d := new(WSDialer)
	client, _, err := d.Dial("ws://127.0.0.1:8080/echo", nil, Config{
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		PingPeriod:      9 * 6 * time.Second,
		PongTimeout:     60 * time.Second,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		BinaryMessages:  true,
	})
	if err != nil {
		fmt.Println("Dialer error:", err)
		t.Fail()
	}
	if client == nil {
		fmt.Println("Dialer returned nil client")
		t.Fail()
	} else {
		// the wait here is longer than the Timeout, so the server will disconnect if
		// ping/pong messages are not correctly triggered
		for i := 0; i < 65; i++ {
			// fmt.Printf("(sleeping) %s\n", time.Now().Format("2006-01-02 15:04:05.000000"))
			time.Sleep(1 * time.Second)
		}
		got_reply := false
		// fmt.Println("Dial complete")
		time.Sleep(1 * time.Second)
		client.On("echo_reply", func(s string) {
			// fmt.Println("client echo_reply", s)
			got_reply = true
		})
		// fmt.Println("ON complete")
		time.Sleep(1 * time.Second)
		client.Emit("echo", "hello")
		// fmt.Println("Emit complete")
		time.Sleep(1 * time.Second)
		if !got_reply {
			fmt.Println("No echo response")
			t.Fail()
		}
	}
	wss.shutdown()
}

func TestMixedMessages(t *testing.T) {
	var wss wsServer
	wss.startup()
	time.Sleep(1 * time.Second)
	d := new(WSDialer)
	client, _, err := d.Dial("ws://127.0.0.1:8080/echo", nil, Config{
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		PingPeriod:      9 * 6 * time.Second,
		PongTimeout:     60 * time.Second,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		BinaryMessages:  true,
	})
	if err != nil {
		fmt.Println("Dialer error:", err)
		t.Fail()
	}
	if client == nil {
		fmt.Println("Dialer returned nil client")
		t.Fail()
	} else {
		cycles := int(100)
		echo_count := int(0)
		len_count := int(0)
		raw_count := int(0)
		// fmt.Println("Dial complete")
		time.Sleep(1 * time.Second)
		client.On("echo_reply", func(s string) {
			//fmt.Println("client echo_reply", s)
			echo_count += 1
		})
		client.On("len_reply", func(i int) {
			// fmt.Printf("client len_reply %d\n", i)
			len_count += 1
		})
		client.OnMessage(func(b []byte) {
			// fmt.Println("client raw_reply", hex.EncodeToString(b))
			raw_count += 1
		})
		// fmt.Println("ON complete")
		time.Sleep(1 * time.Second)
		go func() {
			for i := 0; i < cycles; i++ {
				s := fmt.Sprintf("hello %d", i)
				client.Emit("echo", s)
			}
		}()
		go func() {
			for i := 0; i < cycles; i++ {
				s := make([]byte, i, i)
				for j := 0; j < i; j++ {
					s[j] = byte('a')
				}
				client.Emit("len", string(s))
			}
		}()
		go func() {
			for i := 0; i < cycles; i++ {
				bb := make([]byte, 8)
				binary.BigEndian.PutUint64(bb, uint64(i))
				client.EmitMessage(bb)
			}
		}()
		// fmt.Println("Emit complete")
		// wait until we complete or timeout after 1 minute
		for i := 0; i < 60; i++ {
			if (echo_count == cycles) && (len_count == cycles) && (raw_count == cycles) {
				break
			}
			time.Sleep(1 * time.Second)
		}
		// fmt.Printf("echo, len, raw = %d, %d, %d\n", echo_count, len_count, raw_count)
		if echo_count != cycles {
			fmt.Printf("echo count mismatch, %d != %d\n", echo_count, cycles)
			t.Fail()
		}
		if len_count != cycles {
			fmt.Printf("len count mismatch, %d != %d\n", len_count, cycles)
			t.Fail()
		}
		if raw_count != cycles {
			fmt.Printf("echo count mismatch, %d != %d\n", raw_count, cycles)
			t.Fail()
		}
	}
	wss.shutdown()
}
