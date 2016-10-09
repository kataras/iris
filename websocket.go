package iris

import (
	irisWebsocket "github.com/iris-contrib/websocket"
	"github.com/kataras/go-websocket"
)

// conversionals
const (
	// All is the string which the Emmiter use to send a message to all
	All = websocket.All
	// NotMe is the string which the Emmiter use to send a message to all except this websocket.Connection
	NotMe = websocket.NotMe
	// Broadcast is the string which the Emmiter use to send a message to all except this websocket.Connection, same as 'NotMe'
	Broadcast = websocket.Broadcast
)

// Note I keep this code only to no change the front-end API, we could only use the go-websocket and set our custom upgrader

type (
	// WebsocketServer is the iris websocket server, expose the websocket.Server
	// the below code is a wrapper and bridge between iris-contrib/websocket and kataras/go-websocket
	WebsocketServer struct {
		websocket.Server
		upgrader irisWebsocket.Upgrader

		// the only fields we need at runtime here for iris-specific error and check origin funcs
		// they comes from WebsocketConfiguration

		// Error specifies the function for generating HTTP error responses.
		Error func(ctx *Context, status int, reason string)
		// CheckOrigin returns true if the request Origin header is acceptable. If
		// CheckOrigin is nil, the host in the Origin header must not be set or
		// must match the host of the request.
		CheckOrigin func(ctx *Context) bool
	}
)

// NewWebsocketServer returns an empty WebsocketServer, nothing special here.
func NewWebsocketServer() *WebsocketServer {
	return &WebsocketServer{}
}

// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
//
// The responseHeader is included in the response to the client's upgrade
// request. Use the responseHeader to specify cookies (Set-Cookie) and the
// application negotiated subprotocol (Sec-Websocket-Protocol).
//
// If the upgrade fails, then Upgrade replies to the client with an HTTP error
// response.
func (s *WebsocketServer) Upgrade(ctx *Context) error {
	return s.upgrader.Upgrade(ctx.RequestCtx)
}

// Handler is the iris Handler to upgrade the request
// used inside RegisterRoutes
func (s *WebsocketServer) Handler(ctx *Context) {
	// first, check origin
	if !s.CheckOrigin(ctx) {
		s.Error(ctx, StatusForbidden, "websocket: origin not allowed")
		return
	}

	// all other errors comes from the underline iris-contrib/websocket
	if err := s.Upgrade(ctx); err != nil {
		if ctx.framework.Config.IsDevelopment {
			ctx.Log("Websocket error while trying to Upgrade the connection. Trace: %s", err.Error())
		}

		statusErrCode := StatusBadRequest
		if herr, isHandshake := err.(irisWebsocket.HandshakeError); isHandshake {
			statusErrCode = herr.Status()
		}
		// if not handshake error just fire the custom(if any) StatusBadRequest
		// with the websocket's error message in the ctx.Get("WsError")
		DefaultWebsocketError(ctx, statusErrCode, err.Error())

	}
}

// RegisterTo creates the client side source route and the route path Endpoint with the correct Handler
// receives the websocket configuration and  the iris station
func (s *WebsocketServer) RegisterTo(station *Framework, c WebsocketConfiguration) {

	// Note: s.Server should be initialize on the first OnConnection, which is called before this func  when Default websocket server.
	// When not: when calling this function before OnConnection, when we have more than one websocket server running
	if s.Server == nil {
		s.Server = websocket.New()
	}
	// is just a conversional type for kataras/go-websocket.Connection
	s.upgrader = irisWebsocket.Custom(s.Server.HandleConnection, c.ReadBufferSize, c.WriteBufferSize, c.Headers)

	// set the routing for client-side source (javascript) (optional)
	clientSideLookupName := "iris-websocket-client-side"
	station.Get(c.Endpoint, s.Handler)
	// check if client side already exists
	if station.Lookup(clientSideLookupName) == nil {
		// serve the client side on domain:port/iris-ws.js
		station.StaticContent("/iris-ws.js", contentJavascript, websocket.ClientSource)(clientSideLookupName)
	}

	s.Server.Set(websocket.Config{
		WriteTimeout:    c.WriteTimeout,
		PongTimeout:     c.PongTimeout,
		PingPeriod:      c.PingPeriod,
		MaxMessageSize:  c.MaxMessageSize,
		BinaryMessages:  c.BinaryMessages,
		ReadBufferSize:  c.ReadBufferSize,
		WriteBufferSize: c.WriteBufferSize,
	})

	s.Error = c.Error
	s.CheckOrigin = c.CheckOrigin

	if s.Error == nil {
		s.Error = DefaultWebsocketError
	}

	if s.CheckOrigin == nil {
		s.CheckOrigin = DefaultWebsocketCheckOrigin
	}

	// run the ws server
	s.Server.Serve()

}

// WebsocketConnection is the front-end API that you will use to communicate with the client side
type WebsocketConnection interface {
	websocket.Connection
}

// OnConnection this is the main event you, as developer, will work with each of the websocket connections
func (s *WebsocketServer) OnConnection(connectionListener func(WebsocketConnection)) {
	if s.Server == nil {
		// for default webserver this is the time when the websocket server will be init
		// let's initialize here the ws server, the user/dev is free to change its config before this step.
		s.Server = websocket.New() // we need that in order to use the Iris' WebsocketConnnection, which
		// config is empty here because are setted on the RegisterTo
		// websocket's configuration is optional on New because it doesn't really used before the websocket.Serve
	}
	s.Server.OnConnection(func(c websocket.Connection) {
		connectionListener(c)
	})
}
