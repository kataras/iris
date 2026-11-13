# WebSocket

WebSocket keeps a two-way connection open over TCP, so the server can push to the client instead of waiting to be asked. That is what chat, stock tickers, games and live dashboards are built on.

## What is WebSocket?

WebSocket is a communication protocol that provides full-duplex communication channels over a single TCP connection. It is designed to be implemented in web browsers and web servers, but it can be used by any client or server application. WebSocket is standardized by the IETF as RFC 6455.

For more information, you can refer to the [WebSocket Wikipedia page](https://en.wikipedia.org/wiki/WebSocket) and the [RFC 6455](https://tools.ietf.org/html/rfc6455).

## How to use the WebSocket Middleware?

To use the Iris WebSocket middleware, we need to do the following:

- Import the `github.com/kataras/iris/v14/middleware/websocket` package, which provides an Iris middleware bridge for neffos which is one of the feature-richest WebSocket implementations in Go.
- Import the `github.com/kataras/neffos` package to define the server events and the upgraders.
- Use the `neffos.New` function to create a new WebSocket server.
- Use the `websocket.New` function to create an `iris.Handler` and register it in our Iris app.

First, we need to install the Iris Web Framework and the neffos package:

```sh
go get github.com/kataras/iris/v14@latest
go get github.com/kataras/neffos@latest
```

Then, we need to import them in our `main.go` file:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/websocket"

    "github.com/kataras/neffos"
    "github.com/kataras/neffos/gorilla"
    "github.com/kataras/neffos/gobwas"
)
```

Next, we need to use the `neffos.New` function to create a new WebSocket server. The function takes a WebSocket upgrader and a map of namespaces and events. The namespaces and events define the structure of the WebSocket communication.

We can use the following code to create the WebSocket server:

```go
const namespace = "default"

var serverEvents = neffos.Namespaces{
    namespace: neffos.Events{
        neffos.OnNamespaceConnected: func(nsConn *neffos.NSConn, msg neffos.Message) error {
            ctx := websocket.GetContext(nsConn.Conn)
            log.Printf("[%s] connected to namespace [%s] with IP [%s]",
                nsConn, msg.Namespace, ctx.RemoteAddr())
            return nil
        },
        neffos.OnNamespaceDisconnect: func(nsConn *neffos.NSConn, msg neffos.Message) error {
            log.Printf("[%s] disconnected from namespace [%s]", nsConn, msg.Namespace)
            return nil
        },
        "chat": func(nsConn *neffos.NSConn, msg neffos.Message) error {
            log.Printf("[%s] sent: %s", nsConn, string(msg.Body))
            nsConn.BroadcastOthers(msg)
            return nil
        },
    },
}

websocketServer := neffos.New(
    gorilla.DefaultUpgrader, /* gobwas.DefaultUpgrader can be used too. */
    serverEvents)
```

Finally, we need to register the WebSocket handler in our Iris app. We can use the `app.Any` method to apply the handler on a specific route. For example:

```go
// Create an Iris app instance.
app := iris.New()

// Register the WebSocket handler.
app.Any("/ws", websocket.New(websocketServer, websocket.Options{}))

// Start the server.
app.Listen(":8080")
```

## Close Codes, Heartbeat and Message Size

A server mounted with `websocket.New` or `websocket.Upgrade` gets the optional socket features of neffos, as it does when neffos serves it without Iris:

- `Conn.Close`, `Conn.Terminate` and `Server.Close` send a close frame. The client reads 1000 after `Close`, the code and reason you gave after `Terminate` (`nsConn.Conn.Terminate(neffos.ClosePolicyViolation, "banned")` arrives as 1008 `banned`), and 1001 when the server closes.
- `WithTimeout.MaxMessageSize` caps the size of one incoming message. A bigger one closes the connection with 1009.
- `WithTimeout.PingInterval` runs a heartbeat: the server pings on that interval and closes a peer that sends no pong within it.

```go
websocketServer := neffos.New(gorilla.DefaultUpgrader, neffos.WithTimeout{
    WriteTimeout:   5 * time.Second,
    PingInterval:   30 * time.Second,
    MaxMessageSize: 64 << 10, // 64 KB for one message, envelope included.
    Namespaces:     serverEvents,
})
```

They cost something in three places:

- The size cap counts the neffos envelope as well as the body. With a cap of 64, event `chat` in namespace `default` takes a body of 45 bytes and refuses one of 46.
- The pong is read on the goroutine that runs the event callback, so a callback that runs longer than `PingInterval` can have its connection closed by the heartbeat. A 400ms callback under a 50ms interval was closed.
- `Conn.Close` waits for its close frame when the peer has stopped reading: up to `WriteTimeout`, or one second when that is not set.

`mvc.Websocket` builds its own server with no `WithTimeout`, so only the close frame applies there.

## Example

Here is a complete example of how to use WebSocket and MVC middlewares:

```go
package main

import (
    "fmt"
    "log"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/counter"
    "github.com/kataras/iris/v14/controller/fileserver"
    "github.com/kataras/iris/v14/controller/mvc"
    "github.com/kataras/iris/v14/middleware/websocket"

    "github.com/kataras/neffos"
    "github.com/kataras/neffos/gorilla"
    "github.com/kataras/golog"
)

func main() {
    app := iris.New()

    // Render the ./browser/index.html.
    app.Router("/", fileserver.New("./browser", fileserver.DefaultOptions))

    // Register a dependency which will be used by the WebSocket controller.
    app.RegisterDependency(app.Logger, newTestService)
    // Register the WebSocket server and controller(s).
    app.Router("/websocket", mvc.Websocket(newWebsocketController))

    // Start the server.
    app.Listen(":8080")
}

type testService struct {
    logger *golog.Logger
}

func newTestService(logger *golog.Logger) *testService {
    return &testService{
        logger: logger,
    }
}

func (s *testService) Log(format string, args ...any) {
    s.logger.Infof(format, args...)
}

type websocketController struct {
    service *testService
    counter *counter.Counter[int]
}

func newWebsocketController(service *testService) *websocketController {
    return &websocketController{
        service: service,
        counter: counter.NewCounter[int](),
    }
}

func (c *websocketController) Namespace() string {
    return "default"
}

func (c *websocketController) OnNamespaceConnected(conn *neffos.NSConn, msg neffos.Message) error {
    // visits++
    newCount := c.counter.Increment()
    c.service.Log("Connected [%d]", newCount)

    // This will call the "OnVisit" event on all clients, including the current one,
    // with the 'newCount' variable.
    conn.Broadcast(neffos.Message{
        Namespace: msg.Namespace,
        Event:     "OnVisit",
        Body:      []byte(fmt.Sprintf("%d", newCount)),
    })

    return nil
}

func (c *websocketController) OnNamespaceDisconnect(conn *neffos.NSConn, msg neffos.Message) error {
    // visits--
    newCount := c.counter.Decrement()
    c.service.Log("Disconnected (%d)", newCount)
    // This will call the "OnVisit" event on all clients, except the current one,
    conn.Broadcast(neffos.Message{
        Namespace: msg.Namespace,
        Event:     "OnVisit",
        Body:      []byte(fmt.Sprintf("%d", newCount)),
    })

    return nil
}

func (c *websocketController) OnChat(conn *neffos.NSConn, msg neffos.Message) error {
    c.service.Log("[ID: %s] broadcast to other clients the message [%s]",
        conn, string(msg.Body))

    // Usage for websocket.GetContext to get the Iris http context.
    ctx := websocket.GetContext(conn.Conn)
    // And something to test it...
    ctx.Application().Logger().Infof("[IP: %s] [ID: %s] broadcast to other clients the message [%s]",
        ctx.RemoteAddr(), conn, string(msg.Body))

    conn.BroadcastOthers(msg)
    return nil
}
```

## Client-side Code

Here is the client-side code (`index.html`) to connect to the WebSocket server and handle events:

```html
<html>

<head>
    <title>Online visitors MVC example</title>
    <style>
        body {
            margin: 0;
            font-family: -apple-system, "San Francisco", "Helvetica Neue", "Noto", "Roboto", "Calibri Light", sans-serif;
            color: #212121;
            font-size: 1.0em;
            line-height: 1.6;
        }

        .container {
            max-width: 750px;
            margin: auto;
            padding: 15px;
        }

        #online_visitors {
            font-weight: bold;
            font-size: 18px;
        }
    </style>
</head>

<body>
    <div class="container">
        <span id="online_visitors">1 online visitor</span>
    </div>

    <!-- the message's input -->
    <input id="input" type="text" />

    <!-- when clicked then a websocket event will be sent to the server, at this example we registered the 'chat' -->
    <button id="sendBtn" disabled>Send</button>

    <!-- the messages will be shown here -->
    <pre id="output"></pre>
    <!-- import the iris client-side library for browser from a CDN or locally.
     However, `neffos.(min.)js` is a NPM package too so alternatively,
     you can use it as dependency on your package.json and all nodejs-npm tooling become available:
     see the "browserify" example for more-->
    <script src="https://cdn.jsdelivr.net/npm/neffos.js@latest/dist/neffos-es5.js"></script>
    <script type="text/javascript">
        const wsURL = "ws://localhost:8080/websocket"
        var outputTxt = document.getElementById("output");
        function addMessage(msg) {
            outputTxt.innerHTML += msg + "\n";
        }

        async function runExample() {
            try {
                const conn = await neffos.dial(wsURL, {
                    default: { // "default" namespace.
                        _OnNamespaceConnected: function (nsConn, msg) {
                            if (nsConn.conn.wasReconnected()) {
                                addMessage("re-connected after " + nsConn.conn.reconnectTries.toString() + " trie(s)");
                            }

                            let inputTxt = document.getElementById("input");
                            let sendBtn = document.getElementById("sendBtn");

                            sendBtn.disabled = false;
                            sendBtn.onclick = function () {
                                const input = inputTxt.value;
                                inputTxt.value = "";
                                nsConn.emit("OnChat", input);
                                addMessage("Me: " + input);
                            };

                            addMessage("connected to namespace: " + msg.Namespace);
                        },
                        _OnNamespaceDisconnect: function (nsConn, msg) {
                            addMessage("disconnected from namespace: " + msg.Namespace);
                        },
                        OnChat: function (nsConn, msg) { // "OnChat" event.
                            console.log(msg);

                            addMessage(msg.Body);
                        },
                        OnVisit: function (nsConn, msg) {
                            const newCount = Number(msg.Body); // or parseInt.
                            console.log("visit websocket event with newCount of: ", newCount);

                            var text = "1 online visitor";
                            if (newCount > 1) {
                                text = newCount + " online visitors";
                            }
                            document.getElementById("online_visitors").innerHTML = text;
                        },
                    }
                });

                conn.connect("default");
            } catch (err) {
                console.log(err)
            }
        }

        runExample();
    </script>

</body>

</html>
```
