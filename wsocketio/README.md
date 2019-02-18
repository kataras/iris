# go-socket.io

[![GoDoc](http://godoc.org/github.com/googollee/go-socket.io?status.svg)](http://godoc.org/github.com/googollee/go-socket.io) [![Build Status](https://travis-ci.org/googollee/go-socket.io.svg)](https://travis-ci.org/googollee/go-socket.io)

go-socket.io is an implementation of [Socket.IO](http://socket.io) in Golang, which is a realtime application framework.

Currently this library supports 1.4 version of the Socket.IO client. It supports room and namespaces.

**Help wanted** This project is looking for contributors to help fix bugs and implement new features. Please check [Issue 192](https://github.com/googollee/go-socket.io/issues/192). All help is much appreciated.

* for compatibility with Socket.IO 0.9.x, please use branch 0.9.x *

## Install

Install the package with:

```bash
go get github.com/googollee/go-socket.io
```

Import it with:

```go
import "github.com/googollee/go-socket.io"
```

and use `socketio` as the package name inside the code.

## Example

Please check the example folder for details.

```go
package main

import (
	"log"
	"net/http"

	"github.com/googollee/go-socket.io"
)

func main() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("connection", func(so socketio.Socket) {
		log.Println("on connection")
		so.Join("chat")
		so.On("chat message", func(msg string) {
			log.Println("emit:", so.Emit("chat message", msg))
			server.BroadcastTo("chat", "chat message", msg)
		})
		so.On("disconnection", func() {
			log.Println("on disconnect")
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
```

## Acknowledgements in go-socket.io 1.X.X

[See documentation about acknowledgements](http://socket.io/docs/#sending-and-getting-data-(acknowledgements))

##### Sending ACK with data from SERVER to CLIENT

* Client-side

```javascript
 //using client-side socket.io-1.X.X.js
 socket.emit('some:event', JSON.stringify(someData), function(data){
       console.log('ACK from server wtih data: ', data));
 });
```

* Server-side

```go
// The return type may vary depending on whether you will return
// In golang implementation of Socket.IO don't used callbacks for acknowledgement,
// but used return value, which wrapped into ack package and returned to the client's callback in JavaScript
so.On("some:event", func(msg string) string {
	return msg //Sending ack with data in msg back to client, using "return statement"
})
```

##### Sending ACK with data from CLIENT to SERVER

* Client-side

```javascript
//using client-side socket.io-1.X.X.js
//last parameter of "on" handler is callback for sending ack to server with data or without data
socket.on('some:event', function (msg, sendAckCb) {
    //Sending ACK with data to server after receiving some:event from server
    sendAckCb(JSON.stringify(data)); // for example used serializing to JSON
}
```

* Server-side

```go
//You can use Emit or BroadcastTo with last parameter as callback for handling ack from client
//Sending packet to room "room_name" and event "some:event"
so.BroadcastTo("room_name", "some:event", dataForClient, func (so socketio.Socket, data string) {
	log.Println("Client ACK with data: ", data)
})

// Or

so.Emit("some:event", dataForClient, func (so socketio.Socket, data string) {
	log.Println("Client ACK with data: ", data)
})
```

## License

The 3-clause BSD License  - see LICENSE for more details
