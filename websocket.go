package iris

import (
	"log"

	irisWebsocket "github.com/iris-contrib/websocket"
	"github.com/kataras/go-websocket"
	"github.com/kataras/iris/config"
)

// ---------------------------------------------------------------------------------------------------------
// ---------------------------------------------------------------------------------------------------------
// --------------------------------Websocket implementation-------------------------------------------------
// Global functions in order to be able to use unlimitted number of websocket servers on each iris station--
// ---------------------------------------------------------------------------------------------------------

// Note I keep this code only to no change the front-end API, we could only use the go-websocket and set our custom upgrader

// NewWebsocketServer creates a websocket server and returns it
func NewWebsocketServer(c *config.Websocket) *WebsocketServer {
	wsConfig := websocket.Config{
		WriteTimeout:    c.WriteTimeout,
		PongTimeout:     c.PongTimeout,
		PingPeriod:      c.PingPeriod,
		MaxMessageSize:  c.MaxMessageSize,
		BinaryMessages:  c.BinaryMessages,
		ReadBufferSize:  c.ReadBufferSize,
		WriteBufferSize: c.WriteBufferSize,
	}

	wsServer := websocket.New(wsConfig)

	upgrader := irisWebsocket.Custom(wsServer.HandleConnection, c.ReadBufferSize, c.WriteBufferSize, false)

	srv := &WebsocketServer{Server: wsServer, Config: c, upgrader: upgrader}

	return srv
}

// RegisterWebsocketServer registers the handlers for the websocket server
// it's a bridge between station and websocket server
func RegisterWebsocketServer(station FrameworkAPI, server *WebsocketServer, logger *log.Logger) {
	c := server.Config
	if c.Endpoint == "" {
		return
	}

	websocketHandler := func(ctx *Context) {

		if err := server.Upgrade(ctx); err != nil {
			if ctx.framework.Config.IsDevelopment {
				logger.Printf("Websocket error while trying to Upgrade the connection. Trace: %s", err.Error())
			}
			ctx.EmitError(StatusBadRequest)
		}
	}

	if c.Headers != nil && len(c.Headers) > 0 { // only for performance matter just re-create the websocketHandler if we have headers to set
		websocketHandler = func(ctx *Context) {
			for k, v := range c.Headers {
				ctx.SetHeader(k, v)
			}

			if err := server.Upgrade(ctx); err != nil {
				if ctx.framework.Config.IsDevelopment {
					logger.Printf("Websocket error while trying to Upgrade the connection. Trace: %s", err.Error())
				}
				ctx.EmitError(StatusBadRequest)
			}
		}
	}
	clientSideLookupName := "iris-websocket-client-side"
	station.Get(c.Endpoint, websocketHandler)
	// check if client side already exists
	if station.Lookup(clientSideLookupName) == nil {
		// serve the client side on domain:port/iris-ws.js
		station.StaticContent("/iris-ws.js", contentJavascript, websocket.ClientSource)(clientSideLookupName)
	}

	// run the ws server
	server.Serve()
}

// conversionals
const (
	// All is the string which the Emmiter use to send a message to all
	All = websocket.All
	// NotMe is the string which the Emmiter use to send a message to all except this websocket.Connection
	NotMe = websocket.NotMe
	// Broadcast is the string which the Emmiter use to send a message to all except this websocket.Connection, same as 'NotMe'
	Broadcast = websocket.Broadcast
)

type (
	// WebsocketServer is the iris websocket server, expose the websocket.Server
	// the below code is a wrapper and bridge between iris-contrib/websocket and kataras/go-websocket
	WebsocketServer struct {
		websocket.Server
		Config   *config.Websocket
		upgrader irisWebsocket.Upgrader
	}
)

// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
//
// The responseHeader is included in the response to the client's upgrade
// request. Use the responseHeader to specify cookies (Set-Cookie) and the
// application negotiated subprotocol (Sec-Websocket-Protocol).
//
// If the upgrade fails, then Upgrade replies to the client with an HTTP error
// response.
func (s *WebsocketServer) Upgrade(ctx *Context) error {
	return s.upgrader.Upgrade(ctx)
}

// WebsocketConnection is the front-end API that you will use to communicate with the client side
type WebsocketConnection interface {
	websocket.Connection
}

// OnConnection this is the main event you, as developer, will work with each of the websocket connections
func (s *WebsocketServer) OnConnection(connectionListener func(WebsocketConnection)) {
	s.Server.OnConnection(func(c websocket.Connection) {
		connectionListener(c)
	})
}
