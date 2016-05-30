package websocket

import (
	"time"

	"bytes"

	"github.com/iris-contrib/websocket"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/utils"
)

type (
	// DisconnectFunc is the callback which fires when a client/connection closed
	DisconnectFunc func()
	// NativeMessageFunc is the callback for native websocket messages, receives one []byte parameter which is the raw client's message
	NativeMessageFunc func([]byte)
	// MessageFunc is the second argument to the Emitter's Emit functions.
	// A callback which should receives one parameter of type string, int, bool or any valid JSON/Go struct
	MessageFunc interface{}
	// Connection is the client
	Connection interface {
		// Emmiter implements EmitMessage & Emit
		Emmiter
		// ID returns the connection's identifier
		ID() string
		// OnDisconnect registers a callback which fires when this connection is closed by an error or manual
		OnDisconnect(DisconnectFunc)
		// To defines where server should send a message
		// returns an emmiter to send messages
		To(string) Emmiter
		// OnMessage registers a callback which fires when native websocket message received
		OnMessage(NativeMessageFunc)
		// On registers a callback to a particular event which fires when a message to this event received
		On(string, MessageFunc)
		// Join join a connection to a room, it doesn't check if connection is already there, so care
		Join(string)
		// Leave removes a connection from a room
		Leave(string)
	}

	connection struct {
		underline                *websocket.Conn
		id                       string
		send                     chan []byte
		onDisconnectListeners    []DisconnectFunc
		onNativeMessageListeners []NativeMessageFunc
		onEventListeners         map[string][]MessageFunc
		// these were  maden for performance only
		self      Emmiter // pre-defined emmiter than sends message to its self client
		broadcast Emmiter // pre-defined emmiter that sends message to all except this
		all       Emmiter // pre-defined emmiter which sends message to all clients

		server *server
	}
)

var _ Connection = &connection{}

// connection implementation

func newConnection(websocketConn *websocket.Conn, s *server) *connection {
	c := &connection{
		id:        utils.RandomString(64),
		underline: websocketConn,
		send:      make(chan []byte, 256),
		onDisconnectListeners:    make([]DisconnectFunc, 0),
		onNativeMessageListeners: make([]NativeMessageFunc, 0),
		onEventListeners:         make(map[string][]MessageFunc, 0),
		server:                   s,
	}

	c.self = newEmmiter(c, c.id)
	c.broadcast = newEmmiter(c, NotMe)
	c.all = newEmmiter(c, All)

	return c
}

func (c *connection) write(messageType int, data []byte) error {
	c.underline.SetWriteDeadline(time.Now().Add(config.DefaultWriteTimeout))
	return c.underline.WriteMessage(messageType, data)
}

func (c *connection) writer() {
	ticker := time.NewTicker(config.DefaultPingPeriod)
	defer func() {
		ticker.Stop()
		c.underline.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.write(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *connection) reader() {
	defer func() {
		c.server.free <- c
		c.underline.Close()
	}()
	conn := c.underline

	conn.SetReadLimit(config.DefaultMaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(config.DefaultPongTimeout))
	conn.SetPongHandler(func(s string) error {
		conn.SetReadDeadline(time.Now().Add(config.DefaultPongTimeout))
		return nil
	})

	for {
		if _, data, err := conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				println(err.Error())
			}
			break
		} else {
			c.messageReceived(data)
		}

	}
}

// messageReceived checks the incoming message and fire the nativeMessage listeners or the event listeners (iris-ws custom message)
func (c *connection) messageReceived(data []byte) {

	if bytes.HasPrefix(data, prefixBytes) {
		customData := string(data)
		//it's a custom iris-ws message
		receivedEvt := getCustomEvent(customData)
		listeners := c.onEventListeners[receivedEvt]
		if listeners == nil { // if not listeners for this event exit from here
			return
		}
		customMessage, err := deserialize(receivedEvt, customData)
		if customMessage == nil || err != nil {
			return
		}

		for i := range listeners {
			if fn, ok := listeners[i].(func()); ok { // its a simple func(){} callback
				fn()
			} else if fnString, ok := listeners[i].(func(string)); ok {
				fnString(customMessage.(string))
			} else if fnInt, ok := listeners[i].(func(int)); ok {
				fnInt(customMessage.(int))
			} else if fnBool, ok := listeners[i].(func(bool)); ok {
				fnBool(customMessage.(bool))
			} else if fnBytes, ok := listeners[i].(func([]byte)); ok {
				fnBytes(customMessage.([]byte))
			} else {
				listeners[i].(func(interface{}))(customMessage)
			}

		}
	} else {
		// it's native websocket message
		for i := range c.onNativeMessageListeners {
			c.onNativeMessageListeners[i](data)
		}
	}

}

func (c *connection) ID() string {
	return c.id
}

func (c *connection) fireDisconnect() {
	for i := range c.onDisconnectListeners {
		c.onDisconnectListeners[i]()
	}
}

func (c *connection) OnDisconnect(cb DisconnectFunc) {
	c.onDisconnectListeners = append(c.onDisconnectListeners, cb)
}

func (c *connection) To(to string) Emmiter {
	if to == NotMe { // if send to all except me, then return the pre-defined emmiter, and so on
		return c.broadcast
	} else if to == All {
		return c.all
	} else if to == c.id {
		return c.self
	}
	// is an emmiter to another client/connection
	return newEmmiter(c, to)
}

func (c *connection) EmitMessage(nativeMessage []byte) error {
	return c.self.EmitMessage(nativeMessage)
}

func (c *connection) Emit(event string, message interface{}) error {
	return c.self.Emit(event, message)
}

func (c *connection) OnMessage(cb NativeMessageFunc) {
	c.onNativeMessageListeners = append(c.onNativeMessageListeners, cb)
}

func (c *connection) On(event string, cb MessageFunc) {
	if c.onEventListeners[event] == nil {
		c.onEventListeners[event] = make([]MessageFunc, 0)
	}

	c.onEventListeners[event] = append(c.onEventListeners[event], cb)
}

func (c *connection) Join(roomName string) {
	payload := roomPayload{roomName, c.id}
	c.server.join <- payload
}

func (c *connection) Leave(roomName string) {
	payload := roomPayload{roomName, c.id}
	c.server.leave <- payload
}

//
