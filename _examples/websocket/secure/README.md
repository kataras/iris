# Secure Websockets

1. Run your server through `https://` (`iris.Run(iris.TLS)` or `iris.Run(iris.AutoTLS)` or a custom `iris.Listener(...)`)
2. Nothing changes inside the whole app, including the websocket side
3. The clients must dial the websocket server endpoint (i.e `/echo`) via `wss://` prefix (instead of the non-secure `ws://`), for example `wss://example.com/echo`
4. Ready to GO.
