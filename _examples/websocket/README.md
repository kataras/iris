# Websocket

[WebSocket](https://wikipedia.org/wiki/WebSocket) is a protocol that enables two-way persistent communication channels over TCP connections. It is used for applications such as chat, stock tickers, games, anywhere you want real-time functionality in a web application.

Iris websocket library is now merged with the [neffos real-time framework](https://github.com/kataras/neffos) and Iris-specific helpers and type aliases live on the [iris/websocket](https://github.com/kataras/iris/tree/main/websocket) subpackage. Learn neffos from its [wiki](https://github.com/kataras/neffos#learning-neffos).

Helpers and type aliases improves your code speed when writing a websocket module.
For example, instead of importing both `kataras/iris/websocket` - in order to use its `websocket.Handler` - and `github.com/kataras/neffos` - to create a new websocket server `neffos.New` - you can use the `websocket.New` instead, another example is the `neffos.Conn` which can be declared as `websocket.Conn`.

All neffos and its subpackage's types and package-level functions exist as type aliases on the `kataras/iris/websocket` package too, there are too many of those and there is no need to write each one of those here, some common types: 

- `github.com/kataras/neffos/#Conn`  -> `github.com/kataras/iris/websocket/#Conn`
- `github.com/kataras/neffos/gorilla/#DefaultUpgrader` ->  `github.com/kataras/iris/websocket/#DefaultGorillaUpgrader`
- `github.com/kataras/neffos/stackexchange/redis/#NewStackExchange` ->  `github.com/kataras/iris/websocket/#NewRedisStackExchange`
