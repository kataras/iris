> This package is converted to work with Iris but it was originaly created by Gorila team, the original package is gorilla/websocket


You can find working examples [here](https://github.com/iris-contrib/examples), folders starts with websocket_ are these you are looking for.

## How to use

### From
```go


import (
	"github.com/gorilla/websocket"
	"net/http"
)
//...

var upgrader = websocket.Upgrader{} // use default options

// here is the http handler
func myChatHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)

	//here is the websocket connection actions
	// defer c.Close()
	// mt, message, err := c.ReadMessage()
	// c.WriteMessage(mt, message)
	//....

}

http.HandleFunc("/chat_back", myChatHandler)

```
### To

```go

import (
	"github.com/kataras/iris/websocket"
	"github.com/kataras/iris"
)

// here is the websocket connection handler with actions
func chat(c *websocket.Conn) {
	// defer c.Close()
	// mt, message, err := c.ReadMessage()
	// c.WriteMessage(mt, message)
}

var upgrader = websocket.New(chat) // use default options
//var upgrader = websocket.Custom(chat, 1024, 1024) // customized options, read and write buffer sizes (int). Default: 4096
// var upgrader = websocket.New(chat).DontCheckOrigin() // it's useful when you have the websocket server on a different machine

//here is the http handler
func myChatHandler(ctx *iris.Context) {
	err := upgrader.Upgrade(ctx)// returns only error, executes the handler you defined on the websocket.New before (the 'chat' function)
}

iris.Get("/chat_back", myChatHandler)

```

Difference,

1.  websocket logic is outside of the normal http handler, has it's own function

2.  return an```Upgrader``` using websocket.New/.Custom but also you can return using weboskcet.Upgrader{Receiver:chat}. Note thatwebsocket.Upgrader{} will be fail at runtime.





Click [here](https://github.com/gorilla/websocket) for more, if you find any bugs releated to this package post an [issue](https://github.com/gorilla/websocket).

Now you are looking the contents of the file [README.md](https://github.com/gorilla/websocket/blob/master/README.md), taken from Gorrila's repo.


-----------------


### Documentation

* [API Reference](http://godoc.org/github.com/gorilla/websocket)
* [Chat example](https://github.com/gorilla/websocket/tree/master/examples/chat)
* [Command example](https://github.com/gorilla/websocket/tree/master/examples/command)
* [Client and server example](https://github.com/gorilla/websocket/tree/master/examples/echo)
* [File watch example](https://github.com/gorilla/websocket/tree/master/examples/filewatch)

### Status

The Gorilla WebSocket package provides a complete and tested implementation of
the [WebSocket](http://www.rfc-editor.org/rfc/rfc6455.txt) protocol. The
package API is stable.

### Installation

    go get github.com/gorilla/websocket

### Protocol Compliance

The Gorilla WebSocket package passes the server tests in the [Autobahn Test
Suite](http://autobahn.ws/testsuite) using the application in the [examples/autobahn
subdirectory](https://github.com/gorilla/websocket/tree/master/examples/autobahn).

### Gorilla WebSocket compared with other packages

<table>
<tr>
<th></th>
<th><a href="http://godoc.org/github.com/gorilla/websocket">github.com/gorilla</a></th>
<th><a href="http://godoc.org/golang.org/x/net/websocket">golang.org/x/net</a></th>
</tr>
<tr>
<tr><td colspan="3"><a href="http://tools.ietf.org/html/rfc6455">RFC 6455</a> Features</td></tr>
<tr><td>Passes <a href="http://autobahn.ws/testsuite/">Autobahn Test Suite</a></td><td><a href="https://github.com/gorilla/websocket/tree/master/examples/autobahn">Yes</a></td><td>No</td></tr>
<tr><td>Receive <a href="https://tools.ietf.org/html/rfc6455#section-5.4">fragmented</a> message<td>Yes</td><td><a href="https://code.google.com/p/go/issues/detail?id=7632">No</a>, see note 1</td></tr>
<tr><td>Send <a href="https://tools.ietf.org/html/rfc6455#section-5.5.1">close</a> message</td><td><a href="http://godoc.org/github.com/gorilla/websocket#hdr-Control_Messages">Yes</a></td><td><a href="https://code.google.com/p/go/issues/detail?id=4588">No</a></td></tr>
<tr><td>Send <a href="https://tools.ietf.org/html/rfc6455#section-5.5.2">pings</a> and receive <a href="https://tools.ietf.org/html/rfc6455#section-5.5.3">pongs</a></td><td><a href="http://godoc.org/github.com/gorilla/websocket#hdr-Control_Messages">Yes</a></td><td>No</td></tr>
<tr><td>Get the <a href="https://tools.ietf.org/html/rfc6455#section-5.6">type</a> of a received data message</td><td>Yes</td><td>Yes, see note 2</td></tr>
<tr><td colspan="3">Other Features</tr></td>
<tr><td>Limit size of received message</td><td><a href="http://godoc.org/github.com/gorilla/websocket#Conn.SetReadLimit">Yes</a></td><td><a href="https://code.google.com/p/go/issues/detail?id=5082">No</a></td></tr>
<tr><td>Read message using io.Reader</td><td><a href="http://godoc.org/github.com/gorilla/websocket#Conn.NextReader">Yes</a></td><td>No, see note 3</td></tr>
<tr><td>Write message using io.WriteCloser</td><td><a href="http://godoc.org/github.com/gorilla/websocket#Conn.NextWriter">Yes</a></td><td>No, see note 3</td></tr>
</table>

Notes:

1. Large messages are fragmented in [Chrome's new WebSocket implementation](http://www.ietf.org/mail-archive/web/hybi/current/msg10503.html).
2. The application can get the type of a received data message by implementing
   a [Codec marshal](http://godoc.org/golang.org/x/net/websocket#Codec.Marshal)
   function.
3. The go.net io.Reader and io.Writer operate across WebSocket frame boundaries.
  Read returns when the input buffer is full or a frame boundary is
  encountered. Each call to Write sends a single frame message. The Gorilla
  io.Reader and io.WriteCloser operate on a single WebSocket message.

