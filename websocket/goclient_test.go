// Copyright 2017 Joseph deBlaquiere. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	stdContext "context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/host"
	"github.com/kataras/iris/view"
	//"github.com/kataras/iris/websocket"
)

func validateInterface(con ClientConnection) bool {
	return true
}

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

func (wsc *wsClient) reverseString(message string) {
	// fmt.Println("recv :", message)
	chars := []rune(message)
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	wsc.con.Emit("reverse_reply", string(chars))
}

func (wsc *wsClient) lenString(message string) {
	// fmt.Println("recv :", message)
	wsc.con.Emit("len_reply", len(message))
}

func (wsc *wsClient) disconnect() {
	// fmt.Println("client disconnect @ ", time.Now().Format("2006-01-02 15:04:05.000000"))
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

	// fail compile here if server connection doesn't satisfy interface
	validateInterface(con)
	c := &wsClient{con: con, wss: wss}
	wss.clients = append(wss.clients, c)

	// fmt.Printf("Connect # active clients : %d\n", len(wss.clients))

	con.OnMessage(c.echoRawMessage)
	con.On("echo", c.echoString)
	con.On("len", c.lenString)
	con.On("reverse", c.reverseString)
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
			// fmt.Printf("Disconnect # active clients : %d\n", len(wss.clients))
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
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		BinaryMessages:  true,
		Endpoint:        "/echo",
	})

	ws.OnConnection(wss.connect)

	// Attach the websocket server.
	ws.Attach(wss.app)

	wss.srv = &http.Server{Addr: ":8080"}
	wss.super = host.New(wss.srv)

	go wss.app.Run(iris.Server(wss.srv), iris.WithoutBanner)

	tries := 0
	for {
		_, err := http.Get("http://127.0.0.1:8080/")
		if err == nil {
			break
		}
		tries += 1
		if tries > 30 {
			fmt.Println("Server not responding")
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func (wss *wsServer) shutdown() {
	ctx, _ := stdContext.WithTimeout(stdContext.Background(), 5*time.Second)
	wss.super.Shutdown(ctx)
}

func TestConnectAndWait(t *testing.T) {
	var wss wsServer
	var client ClientConnection
	var err error
	tries_left := int(5)
	wss.startup()
	d := new(WSDialer)
	client = nil
	for (client == nil) && (tries_left > 0) {
		client, _, err = d.Dial("ws://127.0.0.1:8080/echo", nil, Config{
			ReadTimeout:     60 * time.Second,
			WriteTimeout:    60 * time.Second,
			PingPeriod:      9 * 6 * time.Second,
			PongTimeout:     60 * time.Second,
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			BinaryMessages:  true,
		})
		if err != nil {
			fmt.Println("Dialer error:", err)
			if tries_left > 0 {
				time.Sleep(1 * time.Second)
				tries_left -= 1
			} else {
				t.Fail()
			}
		}
	}
	// fail compile here if client doesn't satisfy common interface
	validateInterface(client)
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
		client.On("echo_reply", func(s string) {
			// fmt.Println("client echo_reply", s)
			got_reply = true
		})
		// fmt.Println("ON complete")
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

func TestMixedMessagesConcurrency(t *testing.T) {
	var wss wsServer
	var client ClientConnection
	var err error
	tries_left := int(5)
	wss.startup()
	d := new(WSDialer)
	client = nil
	for (client == nil) && (tries_left > 0) {
		client, _, err = d.Dial("ws://127.0.0.1:8080/echo", nil, Config{
			ReadTimeout:     60 * time.Second,
			WriteTimeout:    60 * time.Second,
			PingPeriod:      9 * 6 * time.Second,
			PongTimeout:     60 * time.Second,
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			BinaryMessages:  true,
		})
		if err != nil {
			fmt.Println("Dialer error:", err)
			if tries_left > 0 {
				time.Sleep(1 * time.Second)
				tries_left -= 1
			} else {
				t.Fail()
			}
		}
	}
	if client == nil {
		fmt.Println("Dialer returned nil client")
		t.Fail()
	} else {
		cycles := int32(500)
		echo_count := int32(0)
		len_count := int32(0)
		reverse_count := int32(0)
		raw_count := int32(0)
		// fmt.Println("Dial complete")
		client.On("echo_reply", func(s string) {
			//fmt.Println("client echo_reply", s)
			atomic.AddInt32(&echo_count, 1)
		})
		client.On("len_reply", func(i int) {
			// fmt.Printf("client len_reply %d\n", i)
			atomic.AddInt32(&len_count, 1)
		})
		client.On("reverse_reply", func(s string) {
			// fmt.Println("client reverse_reply", s)
			atomic.AddInt32(&reverse_count, 1)
		})
		client.OnMessage(func(b []byte) {
			// fmt.Println("client raw_reply", hex.EncodeToString(b))
			atomic.AddInt32(&raw_count, 1)
		})
		// fmt.Println("ON complete")
		var wg sync.WaitGroup
		wg.Add(4)
		go func() {
			defer wg.Done()
			for i := 0; i < int(cycles); i++ {
				s := fmt.Sprintf("hello %d", i)
				if client.Emit("echo", s) != nil {
					fmt.Println("error serializing echo request:", s)
					t.Fail()
				}
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < int(cycles); i++ {
				s := fmt.Sprintf("hello %d", i)
				if client.Emit("reverse", s) != nil {
					fmt.Println("error serializing reverse request:", s)
					t.Fail()
				}
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < int(cycles); i++ {
				s := make([]byte, i, i)
				for j := 0; j < i; j++ {
					s[j] = byte('a')
				}
				if client.Emit("len", string(s)) != nil {
					fmt.Println("error serializing len request:", string(s))
					t.Fail()
				}
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < int(cycles); i++ {
				bb := make([]byte, 8)
				binary.BigEndian.PutUint64(bb, uint64(i))
				if client.EmitMessage(bb) != nil {
					fmt.Println("error serializing raw request:", hex.EncodeToString(bb))
					t.Fail()
				}
			}
		}()
		// ensure all messages sent
		wg.Wait()
		// fmt.Println("Emit complete")
		// wait until we complete or timeout after 1 minute
		for i := 0; i < 60; i++ {
			if (atomic.LoadInt32(&echo_count) == cycles) &&
				(atomic.LoadInt32(&len_count) == cycles) &&
				(atomic.LoadInt32(&reverse_count) == cycles) &&
				(atomic.LoadInt32(&raw_count) == cycles) {
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

func TestServerDisconnect(t *testing.T) {
	var wss wsServer
	var client ClientConnection
	var err error
	connected := true
	tries_left := int(5)
	wss.startup()
	d := new(WSDialer)
	client = nil
	for (client == nil) && (tries_left > 0) {
		client, _, err = d.Dial("ws://127.0.0.1:8080/echo", nil, Config{
			ReadTimeout:     60 * time.Second,
			WriteTimeout:    60 * time.Second,
			PingPeriod:      9 * 6 * time.Second,
			PongTimeout:     60 * time.Second,
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			BinaryMessages:  true,
		})
		if err != nil {
			fmt.Println("Dialer error:", err)
			if tries_left > 0 {
				time.Sleep(1 * time.Second)
				tries_left -= 1
			} else {
				t.Fail()
			}
		}
	}
	if client == nil {
		fmt.Println("Dialer returned nil client")
		t.Fail()
	} else {
		got_reply := false
		client.On("echo_reply", func(s string) {
			// fmt.Println("client echo_reply", s)
			got_reply = true
		})
		client.OnDisconnect(func() {
			// fmt.Println("client echo_reply", s)
			connected = false
		})
		client.Emit("echo", "hello")
		// fmt.Println("Emit complete")
		time.Sleep(1 * time.Second)
		if !got_reply {
			fmt.Println("No echo response")
			t.Fail()
		}
	}
	c := wss.clients[0]
	c.con.Disconnect()
	tries_left = 5
	for connected && (tries_left > 0) {
		time.Sleep(1 * time.Second)
		tries_left -= 1
	}
	if connected {
		fmt.Println("Disconnect not received by client")
		t.Fail()
	}
	wss.shutdown()
}

func TestNoServerDisconnect(t *testing.T) {
	var wss wsServer
	var client ClientConnection
	var err error
	connected := true
	tries_left := int(5)
	wss.startup()
	d := new(WSDialer)
	client = nil
	for (client == nil) && (tries_left > 0) {
		client, _, err = d.Dial("ws://127.0.0.1:8080/echo", nil, Config{
			ReadTimeout:     60 * time.Second,
			WriteTimeout:    60 * time.Second,
			PingPeriod:      9 * 6 * time.Second,
			PongTimeout:     60 * time.Second,
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			BinaryMessages:  true,
		})
		if err != nil {
			fmt.Println("Dialer error:", err)
			if tries_left > 0 {
				time.Sleep(1 * time.Second)
				tries_left -= 1
			} else {
				t.Fail()
			}
		}
	}
	if client == nil {
		fmt.Println("Dialer returned nil client")
		t.Fail()
	} else {
		got_reply := false
		client.On("echo_reply", func(s string) {
			// fmt.Println("client echo_reply", s)
			got_reply = true
		})
		client.OnDisconnect(func() {
			// fmt.Println("client echo_reply", s)
			connected = false
		})
		client.Emit("echo", "hello")
		// fmt.Println("Emit complete")
		time.Sleep(1 * time.Second)
		if !got_reply {
			fmt.Println("No echo response")
			t.Fail()
		}
	}
	tries_left = 5
	for connected && (tries_left > 0) {
		time.Sleep(1 * time.Second)
		tries_left -= 1
	}
	if connected == false {
		fmt.Println("Client disconnected unexpectedly")
		t.Fail()
	}
	wss.shutdown()
}

func TestClientDisconnect(t *testing.T) {
	var wss wsServer
	var client ClientConnection
	var err error
	connected := true
	tries_left := int(5)
	wss.startup()
	d := new(WSDialer)
	client = nil
	for (client == nil) && (tries_left > 0) {
		client, _, err = d.Dial("ws://127.0.0.1:8080/echo", nil, Config{
			ReadTimeout:     60 * time.Second,
			WriteTimeout:    60 * time.Second,
			PingPeriod:      9 * 6 * time.Second,
			PongTimeout:     60 * time.Second,
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			BinaryMessages:  true,
		})
		if err != nil {
			fmt.Println("Dialer error:", err)
			if tries_left > 0 {
				time.Sleep(1 * time.Second)
				tries_left -= 1
			} else {
				t.Fail()
			}
		}
	}
	if client == nil {
		fmt.Println("Dialer returned nil client")
		t.Fail()
	} else {
		got_reply := false
		client.On("echo_reply", func(s string) {
			// fmt.Println("client echo_reply", s)
			got_reply = true
		})
		client.OnDisconnect(func() {
			// fmt.Println("client echo_reply", s)
			connected = false
		})
		client.Emit("echo", "hello")
		// fmt.Println("Emit complete")
		time.Sleep(1 * time.Second)
		if !got_reply {
			fmt.Println("No echo response")
			t.Fail()
		}
	}
	client.Disconnect()
	tries_left = 5
	for connected && (tries_left > 0) {
		time.Sleep(1 * time.Second)
		tries_left -= 1
	}
	if connected == true {
		fmt.Println("No Disconnect Received")
		t.Fail()
	}
	wss.shutdown()
}
