package main

// Run first `go run main.go server`
// and `go run main.go client` as many times as you want.
// Originally written by: github.com/antlaw to describe an old issue.
import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"

	xwebsocket "golang.org/x/net/websocket"
)

// WS is the current websocket connection
var WS *xwebsocket.Conn

// $ go run main.go server
// $ go run main.go client
func main() {
	if len(os.Args) == 2 && strings.ToLower(os.Args[1]) == "server" {
		ServerLoop()
	} else if len(os.Args) == 2 && strings.ToLower(os.Args[1]) == "client" {
		ClientLoop()
	} else {
		fmt.Println("wsserver [server|client]")
	}
}

/////////////////////////////////////////////////////////////////////////
// client side
func sendUntilErr(sendInterval int) {
	i := 1
	for {
		time.Sleep(time.Duration(sendInterval) * time.Second)
		err := SendMessage("2", "all", "objectupdate", "2.UsrSchedule_v1_1")
		if err != nil {
			fmt.Println("failed to send join message", err.Error())
			return
		}
		fmt.Println("objectupdate", i)
		i++
	}
}

func recvUntilErr() {
	var msg = make([]byte, 2048)
	var n int
	var err error
	i := 1
	for {
		if n, err = WS.Read(msg); err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("%v Received: %s.%v\n", time.Now(), string(msg[:n]), i)
		i++
	}

}

//ConnectWebSocket connect a websocket to host
func ConnectWebSocket() error {
	var origin = "http://localhost/"
	var url = "ws://localhost:8080/socket"
	var err error
	WS, err = xwebsocket.Dial(url, "", origin)
	return err
}

// CloseWebSocket closes the current websocket connection
func CloseWebSocket() error {
	if WS != nil {
		return WS.Close()
	}
	return nil
}

// SendMessage broadcast a message to server
func SendMessage(serverID, to, method, message string) error {
	buffer := []byte(message)
	return SendtBytes(serverID, to, method, buffer)
}

// SendtBytes broadcast a message to server
func SendtBytes(serverID, to, method string, message []byte) error {
	// look https://github.com/kataras/iris/blob/master/websocket/message.go , client.go and client.js
	// to understand the buffer line:
	buffer := []byte(fmt.Sprintf("%s%v;0;%v;%v;", websocket.DefaultEvtMessageKey, method, serverID, to))
	buffer = append(buffer, message...)
	_, err := WS.Write(buffer)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// ClientLoop connects to websocket server, the keep send and recv dataS
func ClientLoop() {
	for {
		time.Sleep(time.Second)
		err := ConnectWebSocket()
		if err != nil {
			fmt.Println("failed to connect websocket", err.Error())
			continue
		}
		// time.Sleep(time.Second)
		err = SendMessage("2", "all", "join", "dummy2")
		go sendUntilErr(1)
		recvUntilErr()
		err = CloseWebSocket()
		if err != nil {
			fmt.Println("failed to close websocket", err.Error())
		}
	}

}

/////////////////////////////////////////////////////////////////////////
// server side

// OnConnect handles incoming websocket connection
func OnConnect(c websocket.Connection) {
	fmt.Println("socket.OnConnect()")
	c.On("join", func(message string) { OnJoin(message, c) })
	c.On("objectupdate", func(message string) { OnObjectUpdated(message, c) })
	// ok works too c.EmitMessage([]byte("dsadsa"))
	c.OnDisconnect(func() { OnDisconnect(c) })

}

// ServerLoop listen and serve websocket requests
func ServerLoop() {
	app := iris.New()

	ws := websocket.New(websocket.Config{})

	// register the server on an endpoint.
	// see the inline javascript code i the websockets.html, this endpoint is used to connect to the server.
	app.Get("/socket", ws.Handler())

	ws.OnConnection(OnConnect)
	app.Run(iris.Addr(":8080"))
}

// OnJoin handles Join broadcast group request
func OnJoin(message string, c websocket.Connection) {
	t := time.Now()
	c.Join("server2")
	fmt.Println("OnJoin() time taken:", time.Since(t))
}

// OnObjectUpdated broadcasts to all client an incoming message
func OnObjectUpdated(message string, c websocket.Connection) {
	t := time.Now()
	s := strings.Split(message, ";")
	if len(s) != 3 {
		fmt.Println("OnObjectUpdated() invalid message format:" + message)
		return
	}
	serverID, _, objectID := s[0], s[1], s[2]
	err := c.To("server"+serverID).Emit("objectupdate", objectID)
	if err != nil {
		fmt.Println(err, "failed to broacast object")
		return
	}
	fmt.Println(fmt.Sprintf("OnObjectUpdated() message:%v, time taken: %v", message, time.Since(t)))
}

// OnDisconnect clean up things when a client is disconnected
func OnDisconnect(c websocket.Connection) {
	c.Leave("server2")
	fmt.Println("OnDisconnect(): client disconnected!")

}
