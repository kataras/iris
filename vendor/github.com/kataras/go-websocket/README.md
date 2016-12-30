<a href="https://travis-ci.org/kataras/go-websocket"><img src="https://img.shields.io/travis/kataras/go-websocket.svg?style=flat-square" alt="Build Status"></a>
<a href="https://github.com/kataras/go-websocket/blob/master/LICENSE"><img src="https://img.shields.io/badge/%20license-MIT%20%20License%20-E91E63.svg?style=flat-square" alt="License"></a>
<a href="https://github.com/kataras/go-websocket/releases"><img src="https://img.shields.io/badge/%20release%20-%20v0.0.3-blue.svg?style=flat-square" alt="Releases"></a>
<a href="#docs"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Read me docs"></a>
<a href="https://kataras.rocket.chat/channel/go-websocket"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Build Status"></a>
<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
<a href="#"><img src="https://img.shields.io/badge/platform-Any--OS-yellow.svg?style=flat-square" alt="Platforms"></a>


The package go-websocket provides an easy way to setup a rich Websocket server and client side.

It's already tested on production & used on [Iris](https://github.com/kataras/iris) and [Q](https://github.com/kataras/q) web framework.

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.7.

```bash
$ go get -u github.com/kataras/go-websocket
```


Examples
------------

To view working examples please navigate to the [./examples](https://github.com/kataras/go-websocket/tree/master/examples) folder.

Docs
------------
**WebSocket is a protocol providing full-duplex communication channels over a single TCP connection**.   
The WebSocket protocol was standardized by the IETF as RFC 6455 in 2011, and the WebSocket API in Web IDL is being standardized by the W3C.

WebSocket is designed to be implemented in web browsers and web servers, but it can be used by any client or server application.
The WebSocket protocol is an independent TCP-based protocol. Its only relationship to HTTP is that its handshake is interpreted by HTTP servers as an Upgrade request.
The WebSocket protocol makes more interaction between a browser and a website possible, **facilitating real-time data transfer from and to the server**.

[Read more about Websockets on Wikipedia](https://en.wikipedia.org/wiki/WebSocket).

-----

**Configuration**

```go
// Config the websocket server configuration
type Config struct {
	Error       func(res http.ResponseWriter, req *http.Request, status int, reason error)
	CheckOrigin func(req *http.Request) bool
	// WriteTimeout time allowed to write a message to the connection.
	// Default value is 15 * time.Second
	WriteTimeout time.Duration
	// PongTimeout allowed to read the next pong message from the connection
	// Default value is 60 * time.Second
	PongTimeout time.Duration
	// PingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
	// Default value is (PongTimeout * 9) / 10
	PingPeriod time.Duration
	// MaxMessageSize max message size allowed from connection
	// Default value is 1024
	MaxMessageSize int64
	// BinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
	// compatible if you wanna use the Connection's EmitMessage to send a custom binary data to the client, like a native server-client communication.
	// defaults to false
	BinaryMessages bool
	// ReadBufferSize is the buffer size for the underline reader
	ReadBufferSize int
	// WriteBufferSize is the buffer size for the underline writer
	WriteBufferSize int
}

```

**OUTLINE**

```go
// ws := websocket.New(websocket.Config...{})
// ws.websocket.OnConnection(func(c websocket.Connection){})
// or package-default
websocket.OnConnection(func(c websocket.Connection){})
```

Connection's methods
```go

// Receive from the client
On("anyCustomEvent", func(message string) {})
On("anyCustomEvent", func(message int){})
On("anyCustomEvent", func(message bool){})
On("anyCustomEvent", func(message anyCustomType){})
On("anyCustomEvent", func(){})

// Receive a native websocket message from the client
// compatible without need of import the iris-ws.js to the .html
OnMessage(func(message []byte){})

// Send to the client
Emit("anyCustomEvent", string)
Emit("anyCustomEvent", int)
Emit("anyCustomEvent", bool)
Emit("anyCustomEvent", anyCustomType)

// Send via native websocket way, compatible without need of import the go-websocket.js to the .html
EmitMessage([]byte("anyMessage"))

// Send to specific client(s)
To("otherConnectionId").Emit/EmitMessage...
To("anyCustomRoom").Emit/EmitMessage...

// Send to all opened connections/clients
To(websocket.All).Emit/EmitMessage...

// Send to all opened connections/clients EXCEPT this client(c)
To(websocket.NotMe).Emit/EmitMessage...

// Rooms, group of connections/clients
Join("anyCustomRoom")
Leave("anyCustomRoom")


// Fired when the connection is closed
OnDisconnect(func(){})

// Force-disconnect the client from the server-side
Disconnect() error
```


FAQ
------------

- Q: Did this package works only with net/http ?
- A: No, this package can work with [Iris](https://github.com/kataras/iris) & [fasthttp](https://github.com/valyala/fasthttp) too, look [here for more](https://github.com/kataras/iris/blob/master/websocket.go).

Explore [these questions](https://github.com/kataras/go-websocket/issues?go-websocket=label%3Aquestion) or navigate to the [community chat][Chat].

Versioning
------------

Current: **v0.0.3**


People
------------
The author of go-websocket is [@kataras](https://github.com/kataras).

If you're **willing to donate**, feel free to send **any** amount through paypal

[![](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted)


Contributing
------------
If you are interested in contributing to the go-websocket project, please make a PR.

License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/kataras/go-websocket.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/go-websocket
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/go-websocket/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v0.0.3-blue.svg?style=flat-square
[Release]: https://github.com/kataras/go-websocket/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/go-websocket
[ChatMain]: https://kataras.rocket.chat/channel/go-websocket
[ChatAlternative]: https://gitter.im/kataras/go-websocket
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/go-websocket
[Documentation Widget]: https://img.shields.io/badge/documentation-reference-5272B4.svg?style=flat-square
[Documentation]: https://www.gitbook.com/book/kataras/go-websocket/details
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square
