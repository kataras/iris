package iris

import (
	"net/http"
	"sync"

	"github.com/kataras/go-websocket"
)

// conversionals
const (
	// All is the string which the Emitter use to send a message to all
	All = websocket.All
	// NotMe is the string which the Emitter use to send a message to all except this websocket.Connection
	NotMe = websocket.NotMe
	// Broadcast is the string which the Emitter use to send a message to all except this websocket.Connection, same as 'NotMe'
	Broadcast = websocket.Broadcast
)

// Note I keep this code only to no change the front-end API, we could only use the go-websocket and set our custom upgrader

type (
	// WebsocketServer is the iris websocket server, expose the websocket.Server
	// the below code is a wrapper and bridge between iris-contrib/websocket and kataras/go-websocket
	WebsocketServer struct {
		websocket.Server
		station *Framework
		once    sync.Once
		// Config:
		// if endpoint is not empty then this configuration is used instead of the station's
		// useful when the user/dev wants more than one websocket server inside one iris instance.
		Config WebsocketConfiguration
	}
)

// NewWebsocketServer returns a new empty unitialized websocket server
// it runs on first OnConnection
func NewWebsocketServer(station *Framework) *WebsocketServer {
	return &WebsocketServer{station: station, Server: websocket.New(), Config: station.Config.Websocket}
}

// NewWebsocketServer creates the client side source route and the route path Endpoint with the correct Handler
// receives the websocket configuration and  the iris station
// and returns the websocket server which can be attached to more than one iris station (if needed)
func (ws *WebsocketServer) init() {

	if ws.Config.Endpoint == "" {
		ws.Config = ws.station.Config.Websocket
	}

	c := ws.Config

	if c.Endpoint == "" {
		return
	}

	if c.CheckOrigin == nil {
		c.CheckOrigin = DefaultWebsocketCheckOrigin
	}

	if c.Error == nil {
		c.Error = DefaultWebsocketError
	}
	// set the underline websocket server's configuration
	ws.Server.Set(websocket.Config{
		WriteTimeout:    c.WriteTimeout,
		PongTimeout:     c.PongTimeout,
		PingPeriod:      c.PingPeriod,
		MaxMessageSize:  c.MaxMessageSize,
		BinaryMessages:  c.BinaryMessages,
		ReadBufferSize:  c.ReadBufferSize,
		WriteBufferSize: c.WriteBufferSize,
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			ws.station.Context.Run(w, r, func(ctx *Context) {
				c.Error(ctx, status, reason)
			})
		},
		CheckOrigin: c.CheckOrigin,
		IDGenerator: c.IDGenerator,
	})

	// set the routing for client-side source (javascript) (optional)
	clientSideLookupName := "iris-websocket-client-side"
	ws.station.Get(c.Endpoint, ToHandler(ws.Server.Handler()))
	// check if client side already exists
	if ws.station.Routes().Lookup(clientSideLookupName) == nil {
		// serve the client side on domain:port/iris-ws.js
		ws.station.StaticContent("/iris-ws.js", contentJavascript, websocket.ClientSource).ChangeName(clientSideLookupName)
	}
}

// WebsocketConnection is the front-end API that you will use to communicate with the client side
type WebsocketConnection interface {
	websocket.Connection
}

// OnConnection this is the main event you, as developer, will work with each of the websocket connections
func (ws *WebsocketServer) OnConnection(connectionListener func(WebsocketConnection)) {
	ws.once.Do(ws.init)

	ws.Server.OnConnection(func(c websocket.Connection) {
		connectionListener(c)
	})
}
