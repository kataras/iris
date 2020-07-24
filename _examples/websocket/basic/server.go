package main

import (
	"log"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/websocket"

	// Used when "enableJWT" constant is true:
	"github.com/iris-contrib/middleware/jwt"
)

// values should match with the client sides as well.
const enableJWT = true
const namespace = "default"

// if namespace is empty then simply websocket.Events{...} can be used instead.
var serverEvents = websocket.Namespaces{
	namespace: websocket.Events{
		websocket.OnNamespaceConnected: func(nsConn *websocket.NSConn, msg websocket.Message) error {
			// with `websocket.GetContext` you can retrieve the Iris' `Context`.
			ctx := websocket.GetContext(nsConn.Conn)

			log.Printf("[%s] connected to namespace [%s] with IP [%s]",
				nsConn, msg.Namespace,
				ctx.RemoteAddr())
			return nil
		},
		websocket.OnNamespaceDisconnect: func(nsConn *websocket.NSConn, msg websocket.Message) error {
			log.Printf("[%s] disconnected from namespace [%s]", nsConn, msg.Namespace)
			return nil
		},
		"chat": func(nsConn *websocket.NSConn, msg websocket.Message) error {
			// room.String() returns -> NSConn.String() returns -> Conn.String() returns -> Conn.ID()
			log.Printf("[%s] sent: %s", nsConn, string(msg.Body))

			// Write message back to the client message owner with:
			// nsConn.Emit("chat", msg)
			// Write message to all except this client with:
			nsConn.Conn.Server().Broadcast(nsConn, msg)
			return nil
		},
	},
}

func main() {
	app := iris.New()
	websocketServer := websocket.New(
		websocket.DefaultGorillaUpgrader, /* DefaultGobwasUpgrader can be used too. */
		serverEvents)

	j := jwt.New(jwt.Config{
		// Extract by the "token" url,
		// so the client should dial with ws://localhost:8080/echo?token=$token
		Extractor: jwt.FromParameter("token"),

		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte("My Secret"), nil
		},

		// When set, the middleware verifies that tokens are signed
		// with the specific signing algorithm
		// If the signing method is not constant the
		// `Config.ValidationKeyGetter` callback field can be used
		// to implement additional checks
		// Important to avoid security issues described here:
		// https://auth0.com/blog/2015/03/31/critical-vulnerabilities-in-json-web-token-libraries/
		SigningMethod: jwt.SigningMethodHS256,
	})

	idGen := func(ctx iris.Context) string {
		if username := ctx.GetHeader("X-Username"); username != "" {
			return username
		}

		return websocket.DefaultIDGenerator(ctx)
	}

	// serves the endpoint of ws://localhost:8080/echo
	// with optional custom ID generator.
	websocketRoute := app.Get("/echo", websocket.Handler(websocketServer, idGen))

	if enableJWT {
		// Register the jwt middleware (on handshake):
		websocketRoute.Use(j.Serve)
		// OR
		//
		// Check for token through the jwt middleware
		// on websocket connection or on any event:
		/* websocketServer.OnConnect = func(c *websocket.Conn) error {
		ctx := websocket.GetContext(c)
		if err := j.CheckJWT(ctx); err != nil {
			// will send the above error on the client
			// and will not allow it to connect to the websocket server at all.
			return err
		}

		user := ctx.Values().Get("jwt").(*jwt.Token)
		// or just: user := j.Get(ctx)

		log.Printf("This is an authenticated request\n")
		log.Printf("Claim content:")
		log.Printf("%#+v\n", user.Claims)

		log.Printf("[%s] connected to the server", c.ID())

		return nil
		} */
	}

	// serves the browser-based websocket client.
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("./browser/index.html")
	})

	// serves the npm browser websocket client usage example.
	app.HandleDir("/browserify", iris.Dir("./browserify"))

	app.Listen(":8080")
}
