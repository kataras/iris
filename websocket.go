package iris

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iris-contrib/logger"
	"github.com/iris-contrib/websocket"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/utils"
)

// ---------------------------------------------------------------------------------------------------------
// ---------------------------------------------------------------------------------------------------------
// --------------------------------Websocket implementation-------------------------------------------------
// Global functions in order to be able to use unlimitted number of websocket servers on each iris station--
// ---------------------------------------------------------------------------------------------------------

// NewWebsocketServer creates a websocket server and returns it
func NewWebsocketServer(c *config.Websocket) WebsocketServer {
	return newWebsocketServer(c)
}

// RegisterWebsocketServer registers the handlers for the websocket server
// it's a bridge between station and websocket server
func RegisterWebsocketServer(station FrameworkAPI, server WebsocketServer, logger *logger.Logger) {
	c := server.Config()
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
		station.StaticContent("/iris-ws.js", "application/json", websocketClientSource)(clientSideLookupName)
	}

}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------WebsocketServer implementation-----------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// WebsocketConnectionFunc is the callback which fires when a client/websocketConnection is connected to the websocketServer.
	// Receives one parameter which is the WebsocketConnection
	WebsocketConnectionFunc func(WebsocketConnection)
	// WebsocketRooms is just a map with key a string and  value slice of string
	WebsocketRooms map[string][]string

	// websocketRoomPayload is used as payload from the websocketConnection to the websocketServer
	websocketRoomPayload struct {
		roomName              string
		websocketConnectionID string
	}

	// payloads, websocketConnection -> websocketServer
	websocketMessagePayload struct {
		from string
		to   string
		data []byte
	}

	// WebsocketServer is the websocket server, listens on the config's port, the critical part is the event OnConnection
	WebsocketServer interface {
		Config() *config.Websocket
		Upgrade(ctx *Context) error
		OnConnection(cb WebsocketConnectionFunc)
	}

	websocketServer struct {
		config                *config.Websocket
		upgrader              websocket.Upgrader
		put                   chan *websocketConnection
		free                  chan *websocketConnection
		websocketConnections  map[string]*websocketConnection
		join                  chan websocketRoomPayload
		leave                 chan websocketRoomPayload
		rooms                 WebsocketRooms // by default a websocketConnection is joined to a room which has the websocketConnection id as its name
		mu                    sync.Mutex     // for rooms
		messages              chan websocketMessagePayload
		onConnectionListeners []WebsocketConnectionFunc
		//websocketConnectionPool        *sync.Pool // sadly I can't make this because the websocket websocketConnection is live until is closed.
	}
)

var _ WebsocketServer = &websocketServer{}

// websocketServer implementation

// newWebsocketServer creates a websocket websocketServer and returns it
func newWebsocketServer(c *config.Websocket) *websocketServer {
	s := &websocketServer{
		config:                c,
		put:                   make(chan *websocketConnection),
		free:                  make(chan *websocketConnection),
		websocketConnections:  make(map[string]*websocketConnection),
		join:                  make(chan websocketRoomPayload, 1), // buffered because join can be called immediately on websocketConnection connected
		leave:                 make(chan websocketRoomPayload),
		rooms:                 make(WebsocketRooms),
		messages:              make(chan websocketMessagePayload, 1), // buffered because messages can be sent/received immediately on websocketConnection connected
		onConnectionListeners: make([]WebsocketConnectionFunc, 0),
	}

	s.upgrader = websocket.Custom(s.handleWebsocketConnection, s.config.ReadBufferSize, s.config.WriteBufferSize, false)
	go s.serve() // start the websocketServer automatically
	return s
}

func (s *websocketServer) Config() *config.Websocket {
	return s.config
}

func (s *websocketServer) Upgrade(ctx *Context) error {
	return s.upgrader.Upgrade(ctx)
}

func (s *websocketServer) handleWebsocketConnection(websocketConn *websocket.Conn) {
	c := newWebsocketConnection(websocketConn, s)
	s.put <- c
	go c.writer()
	c.reader()
}

func (s *websocketServer) OnConnection(cb WebsocketConnectionFunc) {
	s.onConnectionListeners = append(s.onConnectionListeners, cb)
}

func (s *websocketServer) joinRoom(roomName string, connID string) {
	s.mu.Lock()
	if s.rooms[roomName] == nil {
		s.rooms[roomName] = make([]string, 0)
	}
	s.rooms[roomName] = append(s.rooms[roomName], connID)
	s.mu.Unlock()
}

func (s *websocketServer) leaveRoom(roomName string, connID string) {
	s.mu.Lock()
	if s.rooms[roomName] != nil {
		for i := range s.rooms[roomName] {
			if s.rooms[roomName][i] == connID {
				s.rooms[roomName][i] = s.rooms[roomName][len(s.rooms[roomName])-1]
				s.rooms[roomName] = s.rooms[roomName][:len(s.rooms[roomName])-1]
				break
			}
		}
		if len(s.rooms[roomName]) == 0 { // if room is empty then delete it
			delete(s.rooms, roomName)
		}
	}

	s.mu.Unlock()
}

func (s *websocketServer) serve() {
	for {
		select {
		case c := <-s.put: // websocketConnection connected
			s.websocketConnections[c.id] = c
			// make and join a room with the websocketConnection's id
			s.rooms[c.id] = make([]string, 0)
			s.rooms[c.id] = []string{c.id}
			for i := range s.onConnectionListeners {
				s.onConnectionListeners[i](c)
			}
		case c := <-s.free: // websocketConnection closed
			if _, found := s.websocketConnections[c.id]; found {
				// leave from all rooms
				for roomName := range s.rooms {
					s.leaveRoom(roomName, c.id)
				}
				delete(s.websocketConnections, c.id)
				close(c.send)
				c.fireDisconnect()

			}
		case join := <-s.join:
			s.joinRoom(join.roomName, join.websocketConnectionID)
		case leave := <-s.leave:
			if _, found := s.websocketConnections[leave.websocketConnectionID]; found {
				s.leaveRoom(leave.roomName, leave.websocketConnectionID)
			}
		case msg := <-s.messages: // message received from the websocketConnection
			if msg.to != All && msg.to != NotMe && s.rooms[msg.to] != nil {
				// it suppose to send the message to a room
				for _, websocketConnectionIDInsideRoom := range s.rooms[msg.to] {
					if c, connected := s.websocketConnections[websocketConnectionIDInsideRoom]; connected {
						c.send <- msg.data //here we send it without need to continue below
					} else {
						// the websocketConnection is not connected but it's inside the room, we remove it on disconnect but for ANY CASE:
						s.leaveRoom(c.id, msg.to)
					}
				}

			} else { // it suppose to send the message to all opened websocketConnections or to all except the sender
				for connID, c := range s.websocketConnections {
					if msg.to != All { // if it's not suppose to send to all websocketConnections (including itself)
						if msg.to == NotMe && msg.from == connID { // if broadcast to other websocketConnections except this
							continue //here we do the opossite of previous block, just skip this websocketConnection when it's suppose to send the message to all websocketConnections except the sender
						}
					}
					select {
					case s.websocketConnections[connID].send <- msg.data: //send the message back to the websocketConnection in order to send it to the client
					default:
						close(c.send)
						delete(s.websocketConnections, connID)
						c.fireDisconnect()

					}

				}
			}

		}

	}
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------WebsocketEmmiter implementation----------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

const (
	// All is the string which the WebsocketEmmiter use to send a message to all
	All = ""
	// NotMe is the string which the WebsocketEmmiter use to send a message to all except this websocketConnection
	NotMe = ";iris;to;all;except;me;"
	// Broadcast is the string which the WebsocketEmmiter use to send a message to all except this websocketConnection, same as 'NotMe'
	Broadcast = NotMe
)

type (
	// WebsocketEmmiter is the message/or/event manager
	WebsocketEmmiter interface {
		// EmitMessage sends a native websocket message
		EmitMessage([]byte) error
		// Emit sends a message on a particular event
		Emit(string, interface{}) error
	}

	websocketEmmiter struct {
		conn *websocketConnection
		to   string
	}
)

var _ WebsocketEmmiter = &websocketEmmiter{}

func newWebsocketEmmiter(c *websocketConnection, to string) *websocketEmmiter {
	return &websocketEmmiter{conn: c, to: to}
}

func (e *websocketEmmiter) EmitMessage(nativeMessage []byte) error {
	mp := websocketMessagePayload{e.conn.id, e.to, nativeMessage}
	e.conn.websocketServer.messages <- mp
	return nil
}

func (e *websocketEmmiter) Emit(event string, data interface{}) error {
	message, err := websocketMessageSerialize(event, data)
	if err != nil {
		return err
	}
	e.EmitMessage([]byte(message))
	return nil
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------WebsocketWebsocketConnection implementation-------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// WebsocketDisconnectFunc is the callback which fires when a client/websocketConnection closed
	WebsocketDisconnectFunc func()
	// WebsocketErrorFunc is the callback which fires when an error happens
	WebsocketErrorFunc (func(string))
	// WebsocketNativeMessageFunc is the callback for native websocket messages, receives one []byte parameter which is the raw client's message
	WebsocketNativeMessageFunc func([]byte)
	// WebsocketMessageFunc is the second argument to the WebsocketEmmiter's Emit functions.
	// A callback which should receives one parameter of type string, int, bool or any valid JSON/Go struct
	WebsocketMessageFunc interface{}
	// WebsocketConnection is the client
	WebsocketConnection interface {
		// WebsocketEmmiter implements EmitMessage & Emit
		WebsocketEmmiter
		// ID returns the websocketConnection's identifier
		ID() string
		// OnDisconnect registers a callback which fires when this websocketConnection is closed by an error or manual
		OnDisconnect(WebsocketDisconnectFunc)
		// OnError registers a callback which fires when this websocketConnection occurs an error
		OnError(WebsocketErrorFunc)
		// EmitError can be used to send a custom error message to the websocketConnection
		//
		// It does nothing more than firing the OnError listeners. It doesn't sends anything to the client.
		EmitError(errorMessage string)
		// To defines where websocketServer should send a message
		// returns an emmiter to send messages
		To(string) WebsocketEmmiter
		// OnMessage registers a callback which fires when native websocket message received
		OnMessage(WebsocketNativeMessageFunc)
		// On registers a callback to a particular event which fires when a message to this event received
		On(string, WebsocketMessageFunc)
		// Join join a websocketConnection to a room, it doesn't check if websocketConnection is already there, so care
		Join(string)
		// Leave removes a websocketConnection from a room
		Leave(string)
		// Disconnect disconnects the client, close the underline websocket conn and removes it from the conn list
		// returns the error, if any, from the underline connection
		Disconnect() error
	}

	websocketConnection struct {
		underline                *websocket.Conn
		id                       string
		send                     chan []byte
		onDisconnectListeners    []WebsocketDisconnectFunc
		onErrorListeners         []WebsocketErrorFunc
		onNativeMessageListeners []WebsocketNativeMessageFunc
		onEventListeners         map[string][]WebsocketMessageFunc
		// these were  maden for performance only
		self      WebsocketEmmiter // pre-defined emmiter than sends message to its self client
		broadcast WebsocketEmmiter // pre-defined emmiter that sends message to all except this
		all       WebsocketEmmiter // pre-defined emmiter which sends message to all clients

		websocketServer *websocketServer
	}
)

var _ WebsocketConnection = &websocketConnection{}

func newWebsocketConnection(websocketConn *websocket.Conn, s *websocketServer) *websocketConnection {
	c := &websocketConnection{
		id:        utils.RandomString(64),
		underline: websocketConn,
		send:      make(chan []byte, 256),
		onDisconnectListeners:    make([]WebsocketDisconnectFunc, 0),
		onErrorListeners:         make([]WebsocketErrorFunc, 0),
		onNativeMessageListeners: make([]WebsocketNativeMessageFunc, 0),
		onEventListeners:         make(map[string][]WebsocketMessageFunc, 0),
		websocketServer:          s,
	}

	c.self = newWebsocketEmmiter(c, c.id)
	c.broadcast = newWebsocketEmmiter(c, NotMe)
	c.all = newWebsocketEmmiter(c, All)

	return c
}

func (c *websocketConnection) write(websocketMessageType int, data []byte) error {
	c.underline.SetWriteDeadline(time.Now().Add(c.websocketServer.config.WriteTimeout))
	return c.underline.WriteMessage(websocketMessageType, data)
}

func (c *websocketConnection) writer() {
	ticker := time.NewTicker(c.websocketServer.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Disconnect()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				defer func() {

					// FIX FOR: https://github.com/kataras/iris/issues/175
					// AS I TESTED ON TRIDENT ENGINE (INTERNET EXPLORER/SAFARI):
					// NAVIGATE TO SITE, CLOSE THE TAB, NOTHING HAPPENS
					// CLOSE THE WHOLE BROWSER, THEN THE c.conn is NOT NILL BUT ALL ITS FUNCTIONS PANICS, MEANS THAT IS THE STRUCT IS NOT NIL BUT THE WRITER/READER ARE NIL
					// THE ONLY SOLUTION IS TO RECOVER HERE AT ANY PANIC
					// THE FRAMETYPE = 8, c.closeSend = true
					// NOTE THAT THE CLIENT IS NOT DISCONNECTED UNTIL THE WHOLE WINDOW BROWSER  CLOSED, this is engine's bug.
					//
					if err := recover(); err != nil {
						ticker.Stop()
						c.Disconnect()
					}
				}()
				c.write(websocket.CloseMessage, []byte{})
				return
			}

			c.underline.SetWriteDeadline(time.Now().Add(c.websocketServer.config.WriteTimeout))
			res, err := c.underline.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			res.Write(msg)

			n := len(c.send)
			for i := 0; i < n; i++ {
				res.Write(<-c.send)
			}

			if err := res.Close(); err != nil {
				return
			}

			// if err := c.write(websocket.TextMessage, msg); err != nil {
			// 	return
			// }

		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *websocketConnection) reader() {
	defer func() {
		c.Disconnect()
	}()
	conn := c.underline

	conn.SetReadLimit(c.websocketServer.config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(c.websocketServer.config.PongTimeout))
	conn.SetPongHandler(func(s string) error {
		conn.SetReadDeadline(time.Now().Add(c.websocketServer.config.PongTimeout))
		return nil
	})

	for {
		if _, data, err := conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.EmitError(err.Error())
			}
			break
		} else {
			c.messageReceived(data)
		}

	}
}

// messageReceived checks the incoming message and fire the nativeMessage listeners or the event listeners (iris-ws custom message)
func (c *websocketConnection) messageReceived(data []byte) {

	if bytes.HasPrefix(data, websocketMessagePrefixBytes) {
		customData := string(data)
		//it's a custom iris-ws message
		receivedEvt := getWebsocketCustomEvent(customData)
		listeners := c.onEventListeners[receivedEvt]
		if listeners == nil { // if not listeners for this event exit from here
			return
		}
		customMessage, err := websocketMessageDeserialize(receivedEvt, customData)
		if customMessage == nil || err != nil {
			return
		}

		for i := range listeners {
			if fn, ok := listeners[i].(func()); ok { // its a simple func(){} callback
				fn()
			} else if fnString, ok := listeners[i].(func(string)); ok {

				if msgString, is := customMessage.(string); is {
					fnString(msgString)
				} else if msgInt, is := customMessage.(int); is {
					// here if websocketServer side waiting for string but client side sent an int, just convert this int to a string
					fnString(strconv.Itoa(msgInt))
				}

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

func (c *websocketConnection) ID() string {
	return c.id
}

func (c *websocketConnection) fireDisconnect() {
	for i := range c.onDisconnectListeners {
		c.onDisconnectListeners[i]()
	}
}

func (c *websocketConnection) OnDisconnect(cb WebsocketDisconnectFunc) {
	c.onDisconnectListeners = append(c.onDisconnectListeners, cb)
}

func (c *websocketConnection) OnError(cb WebsocketErrorFunc) {
	c.onErrorListeners = append(c.onErrorListeners, cb)
}

func (c *websocketConnection) EmitError(errorMessage string) {
	for _, cb := range c.onErrorListeners {
		cb(errorMessage)
	}
}

func (c *websocketConnection) To(to string) WebsocketEmmiter {
	if to == NotMe { // if send to all except me, then return the pre-defined emmiter, and so on
		return c.broadcast
	} else if to == All {
		return c.all
	} else if to == c.id {
		return c.self
	}
	// is an emmiter to another client/websocketConnection
	return newWebsocketEmmiter(c, to)
}

func (c *websocketConnection) EmitMessage(nativeMessage []byte) error {
	return c.self.EmitMessage(nativeMessage)
}

func (c *websocketConnection) Emit(event string, message interface{}) error {
	return c.self.Emit(event, message)
}

func (c *websocketConnection) OnMessage(cb WebsocketNativeMessageFunc) {
	c.onNativeMessageListeners = append(c.onNativeMessageListeners, cb)
}

func (c *websocketConnection) On(event string, cb WebsocketMessageFunc) {
	if c.onEventListeners[event] == nil {
		c.onEventListeners[event] = make([]WebsocketMessageFunc, 0)
	}

	c.onEventListeners[event] = append(c.onEventListeners[event], cb)
}

func (c *websocketConnection) Join(roomName string) {
	payload := websocketRoomPayload{roomName, c.id}
	c.websocketServer.join <- payload
}

func (c *websocketConnection) Leave(roomName string) {
	payload := websocketRoomPayload{roomName, c.id}
	c.websocketServer.leave <- payload
}

func (c *websocketConnection) Disconnect() error {
	c.websocketServer.free <- c // leaves from all rooms, fires the disconnect listeners and finally remove from conn list
	return c.underline.Close()
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------------websocket messages and de/serialization implementation--------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

/*
serializer, [de]websocketMessageSerialize the messages from the client to the websocketServer and from the websocketServer to the client
*/

// The same values are exists on client side also
const (
	websocketStringMessageType websocketMessageType = iota
	websocketIntMessageType
	websocketBoolMessageType
	websocketBytesMessageType
	websocketJSONMessageType
)

const (
	websocketMessagePrefix          = "iris-websocket-message:"
	websocketMessageSeparator       = ";"
	websocketMessagePrefixLen       = len(websocketMessagePrefix)
	websocketMessageSeparatorLen    = len(websocketMessageSeparator)
	websocketMessagePrefixAndSepIdx = websocketMessagePrefixLen + websocketMessageSeparatorLen - 1
	websocketMessagePrefixIdx       = websocketMessagePrefixLen - 1
	websocketMessageSeparatorIdx    = websocketMessageSeparatorLen - 1
)

var (
	websocketMessageSeparatorByte = websocketMessageSeparator[0]
	websocketMessageBuffer        = utils.NewBufferPool(256)
	websocketMessagePrefixBytes   = []byte(websocketMessagePrefix)
)

type (
	websocketMessageType uint8
)

func (m websocketMessageType) String() string {
	return strconv.Itoa(int(m))
}

func (m websocketMessageType) Name() string {
	if m == websocketStringMessageType {
		return "string"
	} else if m == websocketIntMessageType {
		return "int"
	} else if m == websocketBoolMessageType {
		return "bool"
	} else if m == websocketBytesMessageType {
		return "[]byte"
	} else if m == websocketJSONMessageType {
		return "json"
	}

	return "Invalid(" + m.String() + ")"

}

// websocketMessageSerialize serializes a custom websocket message from websocketServer to be delivered to the client
// returns the  string form of the message
// Supported data types are: string, int, bool, bytes and JSON.
func websocketMessageSerialize(event string, data interface{}) (string, error) {
	var msgType websocketMessageType
	var dataMessage string

	if s, ok := data.(string); ok {
		msgType = websocketStringMessageType
		dataMessage = s
	} else if i, ok := data.(int); ok {
		msgType = websocketIntMessageType
		dataMessage = strconv.Itoa(i)
	} else if b, ok := data.(bool); ok {
		msgType = websocketBoolMessageType
		dataMessage = strconv.FormatBool(b)
	} else if by, ok := data.([]byte); ok {
		msgType = websocketBytesMessageType
		dataMessage = string(by)
	} else {
		//we suppose is json
		res, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		msgType = websocketJSONMessageType
		dataMessage = string(res)
	}

	b := websocketMessageBuffer.Get()
	b.WriteString(websocketMessagePrefix)
	b.WriteString(event)
	b.WriteString(websocketMessageSeparator)
	b.WriteString(msgType.String())
	b.WriteString(websocketMessageSeparator)
	b.WriteString(dataMessage)
	dataMessage = b.String()
	websocketMessageBuffer.Put(b)

	return dataMessage, nil

}

// websocketMessageDeserialize deserializes a custom websocket message from the client
// ex: iris-websocket-message;chat;4;themarshaledstringfromajsonstruct will return 'hello' as string
// Supported data types are: string, int, bool, bytes and JSON.
func websocketMessageDeserialize(event string, websocketMessage string) (message interface{}, err error) {
	t, formaterr := strconv.Atoi(websocketMessage[websocketMessagePrefixAndSepIdx+len(event)+1 : websocketMessagePrefixAndSepIdx+len(event)+2]) // in order to iris-websocket-message;user;-> 4
	if formaterr != nil {
		return nil, formaterr
	}
	_type := websocketMessageType(t)
	_message := websocketMessage[websocketMessagePrefixAndSepIdx+len(event)+3:] // in order to iris-websocket-message;user;4; -> themarshaledstringfromajsonstruct

	if _type == websocketStringMessageType {
		message = string(_message)
	} else if _type == websocketIntMessageType {
		message, err = strconv.Atoi(_message)
	} else if _type == websocketBoolMessageType {
		message, err = strconv.ParseBool(_message)
	} else if _type == websocketBytesMessageType {
		message = []byte(_message)
	} else if _type == websocketJSONMessageType {
		err = json.Unmarshal([]byte(_message), message)
	} else {
		return nil, fmt.Errorf("Type %s is invalid for message: %s", _type.Name(), websocketMessage)
	}

	return
}

// getWebsocketCustomEvent return empty string when the websocketMessage is native message
func getWebsocketCustomEvent(websocketMessage string) string {
	if len(websocketMessage) < websocketMessagePrefixAndSepIdx {
		return ""
	}
	s := websocketMessage[websocketMessagePrefixAndSepIdx:]
	evt := s[:strings.IndexByte(s, websocketMessageSeparatorByte)]
	return evt
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------Client side websocket javascript source code ------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

var websocketClientSource = []byte(`var websocketStringMessageType = 0;
var websocketIntMessageType = 1;
var websocketBoolMessageType = 2;
// bytes is missing here for reasons I will explain somewhen
var websocketJSONMessageType = 4;
var websocketMessagePrefix = "iris-websocket-message:";
var websocketMessageSeparator = ";";
var websocketMessagePrefixLen = websocketMessagePrefix.length;
var websocketMessageSeparatorLen = websocketMessageSeparator.length;
var websocketMessagePrefixAndSepIdx = websocketMessagePrefixLen + websocketMessageSeparatorLen - 1;
var websocketMessagePrefixIdx = websocketMessagePrefixLen - 1;
var websocketMessageSeparatorIdx = websocketMessageSeparatorLen - 1;
var Ws = (function () {
    //
    function Ws(endpoint, protocols) {
        var _this = this;
        // events listeners
        this.connectListeners = [];
        this.disconnectListeners = [];
        this.nativeMessageListeners = [];
        this.messageListeners = {};
        if (!window["WebSocket"]) {
            return;
        }
        if (endpoint.indexOf("ws") == -1) {
            endpoint = "ws://" + endpoint;
        }
        if (protocols != null && protocols.length > 0) {
            this.conn = new WebSocket(endpoint, protocols);
        }
        else {
            this.conn = new WebSocket(endpoint);
        }
        this.conn.onopen = (function (evt) {
            _this.fireConnect();
            _this.isReady = true;
            return null;
        });
        this.conn.onclose = (function (evt) {
            _this.fireDisconnect();
            return null;
        });
        this.conn.onmessage = (function (evt) {
            _this.messageReceivedFromConn(evt);
        });
    }
    //utils
    Ws.prototype.isNumber = function (obj) {
        return !isNaN(obj - 0) && obj !== null && obj !== "" && obj !== false;
    };
    Ws.prototype.isString = function (obj) {
        return Object.prototype.toString.call(obj) == "[object String]";
    };
    Ws.prototype.isBoolean = function (obj) {
        return typeof obj === 'boolean' ||
            (typeof obj === 'object' && typeof obj.valueOf() === 'boolean');
    };
    Ws.prototype.isJSON = function (obj) {
        try {
            JSON.parse(obj);
        }
        catch (e) {
            return false;
        }
        return true;
    };
    //
    // messages
    Ws.prototype._msg = function (event, websocketMessageType, dataMessage) {
        return websocketMessagePrefix + event + websocketMessageSeparator + String(websocketMessageType) + websocketMessageSeparator + dataMessage;
    };
    Ws.prototype.encodeMessage = function (event, data) {
        var m = "";
        var t = 0;
        if (this.isNumber(data)) {
            t = websocketIntMessageType;
            m = data.toString();
        }
        else if (this.isBoolean(data)) {
            t = websocketBoolMessageType;
            m = data.toString();
        }
        else if (this.isString(data)) {
            t = websocketStringMessageType;
            m = data.toString();
        }
        else if (this.isJSON(data)) {
            //propably json-object
            t = websocketJSONMessageType;
            m = JSON.stringify(data);
        }
        else {
            console.log("Invalid");
        }
        return this._msg(event, t, m);
    };
    Ws.prototype.decodeMessage = function (event, websocketMessage) {
        //iris-websocket-message;user;4;themarshaledstringfromajsonstruct
        var skipLen = websocketMessagePrefixLen + websocketMessageSeparatorLen + event.length + 2;
        if (websocketMessage.length < skipLen + 1) {
            return null;
        }
        var websocketMessageType = parseInt(websocketMessage.charAt(skipLen - 2));
        var theMessage = websocketMessage.substring(skipLen, websocketMessage.length);
        if (websocketMessageType == websocketIntMessageType) {
            return parseInt(theMessage);
        }
        else if (websocketMessageType == websocketBoolMessageType) {
            return Boolean(theMessage);
        }
        else if (websocketMessageType == websocketStringMessageType) {
            return theMessage;
        }
        else if (websocketMessageType == websocketJSONMessageType) {
            return JSON.parse(theMessage);
        }
        else {
            return null; // invalid
        }
    };
    Ws.prototype.getWebsocketCustomEvent = function (websocketMessage) {
        if (websocketMessage.length < websocketMessagePrefixAndSepIdx) {
            return "";
        }
        var s = websocketMessage.substring(websocketMessagePrefixAndSepIdx, websocketMessage.length);
        var evt = s.substring(0, s.indexOf(websocketMessageSeparator));
        return evt;
    };
    Ws.prototype.getCustomMessage = function (event, websocketMessage) {
        var eventIdx = websocketMessage.indexOf(event + websocketMessageSeparator);
        var s = websocketMessage.substring(eventIdx + event.length + websocketMessageSeparator.length + 2, websocketMessage.length);
        return s;
    };
    //
    // Ws Events
    // messageReceivedFromConn this is the func which decides
    // if it's a native websocket message or a custom iris-ws message
    // if native message then calls the fireNativeMessage
    // else calls the fireMessage
    //
    // remember Iris gives you the freedom of native websocket messages if you don't want to use this client side at all.
    Ws.prototype.messageReceivedFromConn = function (evt) {
        //check if iris-ws message
        var message = evt.data;
        if (message.indexOf(websocketMessagePrefix) != -1) {
            var event_1 = this.getWebsocketCustomEvent(message);
            if (event_1 != "") {
                // it's a custom message
                this.fireMessage(event_1, this.getCustomMessage(event_1, message));
                return;
            }
        }
        // it's a native websocket message
        this.fireNativeMessage(message);
    };
    Ws.prototype.OnConnect = function (fn) {
        if (this.isReady) {
            fn();
        }
        this.connectListeners.push(fn);
    };
    Ws.prototype.fireConnect = function () {
        for (var i = 0; i < this.connectListeners.length; i++) {
            this.connectListeners[i]();
        }
    };
    Ws.prototype.OnDisconnect = function (fn) {
        this.disconnectListeners.push(fn);
    };
    Ws.prototype.fireDisconnect = function () {
        for (var i = 0; i < this.disconnectListeners.length; i++) {
            this.disconnectListeners[i]();
        }
    };
    Ws.prototype.OnMessage = function (cb) {
        this.nativeMessageListeners.push(cb);
    };
    Ws.prototype.fireNativeMessage = function (websocketMessage) {
        for (var i = 0; i < this.nativeMessageListeners.length; i++) {
            this.nativeMessageListeners[i](websocketMessage);
        }
    };
    Ws.prototype.On = function (event, cb) {
        if (this.messageListeners[event] == null || this.messageListeners[event] == undefined) {
            this.messageListeners[event] = [];
        }
        this.messageListeners[event].push(cb);
    };
    Ws.prototype.fireMessage = function (event, message) {
        for (var key in this.messageListeners) {
            if (this.messageListeners.hasOwnProperty(key)) {
                if (key == event) {
                    for (var i = 0; i < this.messageListeners[key].length; i++) {
                        this.messageListeners[key][i](message);
                    }
                }
            }
        }
    };
    //
    // Ws Actions
    Ws.prototype.Disconnect = function () {
        this.conn.close();
    };
    // EmitMessage sends a native websocket message
    Ws.prototype.EmitMessage = function (websocketMessage) {
        this.conn.send(websocketMessage);
    };
    // Emit sends an iris-custom websocket message
    Ws.prototype.Emit = function (event, data) {
        var messageStr = this.encodeMessage(event, data);
        this.EmitMessage(messageStr);
    };
    return Ws;
}());
`)

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------Client side websocket commented typescript source code --------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

/*
const websocketStringMessageType = 0;
const websocketIntMessageType = 1;
const websocketBoolMessageType = 2;
// bytes is missing here for reasons I will explain somewhen
const websocketJSONMessageType = 4;

const websocketMessagePrefix = "iris-websocket-message:";
const websocketMessageSeparator = ";";

const websocketMessagePrefixLen = websocketMessagePrefix.length;
var websocketMessageSeparatorLen = websocketMessageSeparator.length;
var websocketMessagePrefixAndSepIdx = websocketMessagePrefixLen + websocketMessageSeparatorLen - 1;
var websocketMessagePrefixIdx = websocketMessagePrefixLen - 1;
var websocketMessageSeparatorIdx = websocketMessageSeparatorLen - 1;

type onConnectFunc = () => void;
type onWebsocketDisconnectFunc = () => void;
type onWebsocketNativeMessageFunc = (websocketMessage: string) => void;
type onMessageFunc = (message: any) => void;

class Ws {
    private conn: WebSocket;
    private isReady: boolean;

    // events listeners

    private connectListeners: onConnectFunc[] = [];
    private disconnectListeners: onWebsocketDisconnectFunc[] = [];
    private nativeMessageListeners: onWebsocketNativeMessageFunc[] = [];
    private messageListeners: { [event: string]: onMessageFunc[] } = {};

    //

    constructor(endpoint: string, protocols?: string[]) {
        if (!window["WebSocket"]) {
            return;
        }

        if (endpoint.indexOf("ws") == -1) {
            endpoint = "ws://" + endpoint;
        }
        if (protocols != null && protocols.length > 0) {
            this.conn = new WebSocket(endpoint, protocols);
        } else {
            this.conn = new WebSocket(endpoint);
        }

        this.conn.onopen = ((evt: Event): any => {
            this.fireConnect();
            this.isReady = true;
            return null;
        });

        this.conn.onclose = ((evt: Event): any => {
            this.fireDisconnect();
            return null;
        });

        this.conn.onmessage = ((evt: MessageEvent) => {
            this.messageReceivedFromConn(evt);
        });
    }

    //utils

    private isNumber(obj: any): boolean {
        return !isNaN(obj - 0) && obj !== null && obj !== "" && obj !== false;
    }

    private isString(obj: any): boolean {
        return Object.prototype.toString.call(obj) == "[object String]";
    }

    private isBoolean(obj: any): boolean {
        return typeof obj === 'boolean' ||
            (typeof obj === 'object' && typeof obj.valueOf() === 'boolean');
    }

    private isJSON(obj: any): boolean {
        try {
            JSON.parse(obj);
        } catch (e) {
            return false;
        }
        return true;
    }

    //

    // messages
    private _msg(event: string, websocketMessageType: number, dataMessage: string): string {

        return websocketMessagePrefix + event + websocketMessageSeparator + String(websocketMessageType) + websocketMessageSeparator + dataMessage;
    }

    private encodeMessage(event: string, data: any): string {
        let m = "";
        let t = 0;
        if (this.isNumber(data)) {
            t = websocketIntMessageType;
            m = data.toString();
        } else if (this.isBoolean(data)) {
            t = websocketBoolMessageType;
            m = data.toString();
        } else if (this.isString(data)) {
            t = websocketStringMessageType;
            m = data.toString();
        } else if (this.isJSON(data)) {
            //propably json-object
            t = websocketJSONMessageType;
            m = JSON.stringify(data);
        } else {
            console.log("Invalid");
        }

        return this._msg(event, t, m);
    }

    private decodeMessage<T>(event: string, websocketMessage: string): T | any {
        //iris-websocket-message;user;4;themarshaledstringfromajsonstruct
        let skipLen = websocketMessagePrefixLen + websocketMessageSeparatorLen + event.length + 2;
        if (websocketMessage.length < skipLen + 1) {
            return null;
        }
        let websocketMessageType = parseInt(websocketMessage.charAt(skipLen - 2));
        let theMessage = websocketMessage.substring(skipLen, websocketMessage.length);
        if (websocketMessageType == websocketIntMessageType) {
            return parseInt(theMessage);
        } else if (websocketMessageType == websocketBoolMessageType) {
            return Boolean(theMessage);
        } else if (websocketMessageType == websocketStringMessageType) {
            return theMessage;
        } else if (websocketMessageType == websocketJSONMessageType) {
            return JSON.parse(theMessage);
        } else {
            return null; // invalid
        }
    }

    private getWebsocketCustomEvent(websocketMessage: string): string {
        if (websocketMessage.length < websocketMessagePrefixAndSepIdx) {
            return "";
        }
        let s = websocketMessage.substring(websocketMessagePrefixAndSepIdx, websocketMessage.length);
        let evt = s.substring(0, s.indexOf(websocketMessageSeparator));

        return evt;
    }

    private getCustomMessage(event: string, websocketMessage: string): string {
        let eventIdx = websocketMessage.indexOf(event + websocketMessageSeparator);
        let s = websocketMessage.substring(eventIdx + event.length + websocketMessageSeparator.length+2, websocketMessage.length);
        return s;
    }

    //

    // Ws Events

    // messageReceivedFromConn this is the func which decides
    // if it's a native websocket message or a custom iris-ws message
    // if native message then calls the fireNativeMessage
    // else calls the fireMessage
    //
    // remember Iris gives you the freedom of native websocket messages if you don't want to use this client side at all.
    private messageReceivedFromConn(evt: MessageEvent): void {
        //check if iris-ws message
        let message = <string>evt.data;
        if (message.indexOf(websocketMessagePrefix) != -1) {
            let event = this.getWebsocketCustomEvent(message);
            if (event != "") {
                // it's a custom message
                this.fireMessage(event, this.getCustomMessage(event, message));
                return;
            }
        }

        // it's a native websocket message
        this.fireNativeMessage(message);
    }

    OnConnect(fn: onConnectFunc): void {
        if (this.isReady) {
            fn();
        }
        this.connectListeners.push(fn);
    }

    fireConnect(): void {
        for (let i = 0; i < this.connectListeners.length; i++) {
            this.connectListeners[i]();
        }
    }

    OnDisconnect(fn: onWebsocketDisconnectFunc): void {
        this.disconnectListeners.push(fn);
    }

    fireDisconnect(): void {
        for (let i = 0; i < this.disconnectListeners.length; i++) {
            this.disconnectListeners[i]();
        }
    }

    OnMessage(cb: onWebsocketNativeMessageFunc): void {
        this.nativeMessageListeners.push(cb);
    }

    fireNativeMessage(websocketMessage: string): void {
        for (let i = 0; i < this.nativeMessageListeners.length; i++) {
            this.nativeMessageListeners[i](websocketMessage);
        }
    }

    On(event: string, cb: onMessageFunc): void {
        if (this.messageListeners[event] == null || this.messageListeners[event] == undefined) {
            this.messageListeners[event] = [];
        }
        this.messageListeners[event].push(cb);
    }

    fireMessage(event: string, message: any): void {
        for (let key in this.messageListeners) {
            if (this.messageListeners.hasOwnProperty(key)) {
                if (key == event) {
                    for (let i = 0; i < this.messageListeners[key].length; i++) {
                        this.messageListeners[key][i](message);
                    }
                }
            }
        }
    }


    //

    // Ws Actions

    Disconnect(): void {
        this.conn.close();
    }

    // EmitMessage sends a native websocket message
    EmitMessage(websocketMessage: string): void {
        this.conn.send(websocketMessage);
    }

    // Emit sends an iris-custom websocket message
    Emit(event: string, data: any): void {
        let messageStr = this.encodeMessage(event, data);
        this.EmitMessage(messageStr);
    }

    //

}

// node-modules export {Ws};
*/
