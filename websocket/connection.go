package websocket

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kataras/iris/context"
)

type (
	connectionValue struct {
		key   []byte
		value interface{}
	}
	// ConnectionValues is the temporary connection's memory store
	ConnectionValues []connectionValue
)

// Set sets a value based on the key
func (r *ConnectionValues) Set(key string, value interface{}) {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if string(kv.key) == key {
			kv.value = value
			return
		}
	}

	c := cap(args)
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.key = append(kv.key[:0], key...)
		kv.value = value
		*r = args
		return
	}

	kv := connectionValue{}
	kv.key = append(kv.key[:0], key...)
	kv.value = value
	*r = append(args, kv)
}

// Get returns a value based on its key
func (r *ConnectionValues) Get(key string) interface{} {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if string(kv.key) == key {
			return kv.value
		}
	}
	return nil
}

// Reset clears the values
func (r *ConnectionValues) Reset() {
	*r = (*r)[:0]
}

// UnderlineConnection is the underline connection, nothing to think about,
// it's used internally mostly but can be used for extreme cases with other libraries.
type UnderlineConnection interface {
	// SetWriteDeadline sets the write deadline on the underlying network
	// connection. After a write has timed out, the websocket state is corrupt and
	// all future writes will return an error. A zero value for t means writes will
	// not time out.
	SetWriteDeadline(t time.Time) error
	// SetReadDeadline sets the read deadline on the underlying network connection.
	// After a read has timed out, the websocket connection state is corrupt and
	// all future reads will return an error. A zero value for t means reads will
	// not time out.
	SetReadDeadline(t time.Time) error
	// SetReadLimit sets the maximum size for a message read from the peer. If a
	// message exceeds the limit, the connection sends a close frame to the peer
	// and returns ErrReadLimit to the application.
	SetReadLimit(limit int64)
	// SetPongHandler sets the handler for pong messages received from the peer.
	// The appData argument to h is the PONG frame application data. The default
	// pong handler does nothing.
	SetPongHandler(h func(appData string) error)
	// SetPingHandler sets the handler for ping messages received from the peer.
	// The appData argument to h is the PING frame application data. The default
	// ping handler sends a pong to the peer.
	SetPingHandler(h func(appData string) error)
	// WriteControl writes a control message with the given deadline. The allowed
	// message types are CloseMessage, PingMessage and PongMessage.
	WriteControl(messageType int, data []byte, deadline time.Time) error
	// WriteMessage is a helper method for getting a writer using NextWriter,
	// writing the message and closing the writer.
	WriteMessage(messageType int, data []byte) error
	// ReadMessage is a helper method for getting a reader using NextReader and
	// reading from that reader to a buffer.
	ReadMessage() (messageType int, p []byte, err error)
	// NextWriter returns a writer for the next message to send. The writer's Close
	// method flushes the complete message to the network.
	//
	// There can be at most one open writer on a connection. NextWriter closes the
	// previous writer if the application has not already done so.
	NextWriter(messageType int) (io.WriteCloser, error)
	// Close closes the underlying network connection without sending or waiting for a close frame.
	Close() error
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------Connection implementation-----------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// DisconnectFunc is the callback which is fired when a client/connection closed
	DisconnectFunc func()
	// LeaveRoomFunc is the callback which is fired when a client/connection leaves from any room.
	// This is called automatically when client/connection disconnected
	// (because websocket server automatically leaves from all joined rooms)
	LeaveRoomFunc func(roomName string)
	// ErrorFunc is the callback which fires whenever an error occurs
	ErrorFunc (func(error))
	// NativeMessageFunc is the callback for native websocket messages, receives one []byte parameter which is the raw client's message
	NativeMessageFunc func([]byte)
	// MessageFunc is the second argument to the Emitter's Emit functions.
	// A callback which should receives one parameter of type string, int, bool or any valid JSON/Go struct
	MessageFunc interface{}
	// PingFunc is the callback which fires each ping
	PingFunc func()
	// PongFunc is the callback which fires on pong message received
	PongFunc func()
	// Connection is the front-end API that you will use to communicate with the client side
	Connection interface {
		// Emitter implements EmitMessage & Emit
		Emitter
		// Err is not nil if the upgrader failed to upgrade http to websocket connection.
		Err() error

		// ID returns the connection's identifier
		ID() string

		// Server returns the websocket server instance
		// which this connection is listening to.
		//
		// Its connection-relative operations are safe for use.
		Server() *Server

		// Write writes a raw websocket message with a specific type to the client
		// used by ping messages and any CloseMessage types.
		Write(websocketMessageType int, data []byte) error

		// Context returns the (upgraded) context.Context of this connection
		// avoid using it, you normally don't need it,
		// websocket has everything you need to authenticate the user BUT if it's necessary
		// then  you use it to receive user information, for example: from headers
		Context() context.Context

		// OnDisconnect registers a callback which is fired when this connection is closed by an error or manual
		OnDisconnect(DisconnectFunc)
		// OnError registers a callback which fires when this connection occurs an error
		OnError(ErrorFunc)
		// OnPing  registers a callback which fires on each ping
		OnPing(PingFunc)
		// OnPong  registers a callback which fires on pong message received
		OnPong(PongFunc)
		// FireOnError can be used to send a custom error message to the connection
		//
		// It does nothing more than firing the OnError listeners. It doesn't send anything to the client.
		FireOnError(err error)
		// To defines on what "room" (see Join) the server should send a message
		// returns an Emmiter(`EmitMessage` & `Emit`) to send messages.
		To(string) Emitter
		// OnMessage registers a callback which fires when native websocket message received
		OnMessage(NativeMessageFunc)
		// On registers a callback to a particular event which is fired when a message to this event is received
		On(string, MessageFunc)
		// Join registers this connection to a room, if it doesn't exist then it creates a new. One room can have one or more connections. One connection can be joined to many rooms. All connections are joined to a room specified by their `ID` automatically.
		Join(string)
		// IsJoined returns true when this connection is joined to the room, otherwise false.
		// It Takes the room name as its input parameter.
		IsJoined(roomName string) bool
		// Leave removes this connection entry from a room
		// Returns true if the connection has actually left from the particular room.
		Leave(string) bool
		// OnLeave registers a callback which fires when this connection left from any joined room.
		// This callback is called automatically on Disconnected client, because websocket server automatically
		// deletes the disconnected connection from any joined rooms.
		//
		// Note: the callback(s) called right before the server deletes the connection from the room
		// so the connection theoretical can still send messages to its room right before it is being disconnected.
		OnLeave(roomLeaveCb LeaveRoomFunc)
		// Wait starts the pinger and the messages reader,
		// it's named as "Wait" because it should be called LAST,
		// after the "On" events IF server's `Upgrade` is used,
		// otherise you don't have to call it because the `Handler()` does it automatically.
		Wait()
		// Disconnect disconnects the client, close the underline websocket conn and removes it from the conn list
		// returns the error, if any, from the underline connection
		Disconnect() error
		// SetValue sets a key-value pair on the connection's mem store.
		SetValue(key string, value interface{})
		// GetValue gets a value by its key from the connection's mem store.
		GetValue(key string) interface{}
		// GetValueArrString gets a value as []string by its key from the connection's mem store.
		GetValueArrString(key string) []string
		// GetValueString gets a value as string by its key from the connection's mem store.
		GetValueString(key string) string
		// GetValueInt gets a value as integer by its key from the connection's mem store.
		GetValueInt(key string) int
	}

	connection struct {
		err                      error
		underline                UnderlineConnection
		id                       string
		messageType              int
		disconnected             bool
		onDisconnectListeners    []DisconnectFunc
		onRoomLeaveListeners     []LeaveRoomFunc
		onErrorListeners         []ErrorFunc
		onPingListeners          []PingFunc
		onPongListeners          []PongFunc
		onNativeMessageListeners []NativeMessageFunc
		onEventListeners         map[string][]MessageFunc
		started                  bool
		// these were  maden for performance only
		self      Emitter // pre-defined emitter than sends message to its self client
		broadcast Emitter // pre-defined emitter that sends message to all except this
		all       Emitter // pre-defined emitter which sends message to all clients

		// access to the Context, use with caution, you can't use response writer as you imagine.
		ctx    context.Context
		values ConnectionValues
		server *Server
		// #119 , websocket writers are not protected by locks inside the gorilla's websocket code
		// so we must protect them otherwise we're getting concurrent connection error on multi writers in the same time.
		writerMu sync.Mutex
		// same exists for reader look here: https://godoc.org/github.com/gorilla/websocket#hdr-Control_Messages
		// but we only use one reader in one goroutine, so we are safe.
		// readerMu sync.Mutex
	}
)

var _ Connection = &connection{}

// CloseMessage denotes a close control message. The optional message
// payload contains a numeric code and text. Use the FormatCloseMessage
// function to format a close message payload.
//
// Use the `Connection#Disconnect` instead.
const CloseMessage = websocket.CloseMessage

func newConnection(ctx context.Context, s *Server, underlineConn UnderlineConnection, id string) *connection {
	c := &connection{
		underline:                underlineConn,
		id:                       id,
		messageType:              websocket.TextMessage,
		onDisconnectListeners:    make([]DisconnectFunc, 0),
		onRoomLeaveListeners:     make([]LeaveRoomFunc, 0),
		onErrorListeners:         make([]ErrorFunc, 0),
		onNativeMessageListeners: make([]NativeMessageFunc, 0),
		onEventListeners:         make(map[string][]MessageFunc, 0),
		onPongListeners:          make([]PongFunc, 0),
		started:                  false,
		ctx:                      ctx,
		server:                   s,
	}

	if s.config.BinaryMessages {
		c.messageType = websocket.BinaryMessage
	}

	c.self = newEmitter(c, c.id)
	c.broadcast = newEmitter(c, Broadcast)
	c.all = newEmitter(c, All)

	return c
}

// Err is not nil if the upgrader failed to upgrade http to websocket connection.
func (c *connection) Err() error {
	return c.err
}

// Write writes a raw websocket message with a specific type to the client
// used by ping messages and any CloseMessage types.
func (c *connection) Write(websocketMessageType int, data []byte) error {
	// for any-case the app tries to write from different goroutines,
	// we must protect them because they're reporting that as bug...
	c.writerMu.Lock()
	if writeTimeout := c.server.config.WriteTimeout; writeTimeout > 0 {
		// set the write deadline based on the configuration
		c.underline.SetWriteDeadline(time.Now().Add(writeTimeout))
	}

	// .WriteMessage same as NextWriter and close (flush)
	err := c.underline.WriteMessage(websocketMessageType, data)
	c.writerMu.Unlock()
	if err != nil {
		// if failed then the connection is off, fire the disconnect
		c.Disconnect()
	}
	return err
}

// writeDefault is the same as write but the message type is the configured by c.messageType
// if BinaryMessages is enabled then it's raw []byte as you expected to work with protobufs
func (c *connection) writeDefault(data []byte) {
	c.Write(c.messageType, data)
}

const (
	// WriteWait is 1 second at the internal implementation,
	// same as here but this can be changed at the future*
	WriteWait = 1 * time.Second
)

func (c *connection) startPinger() {

	// this is the default internal handler, we just change the writeWait because of the actions we must do before
	// the server sends the ping-pong.

	pingHandler := func(message string) error {
		err := c.underline.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(WriteWait))
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}
		return err
	}

	c.underline.SetPingHandler(pingHandler)

	go func() {
		for {
			// using sleep avoids the ticker error that causes a memory leak
			time.Sleep(c.server.config.PingPeriod)
			if c.disconnected {
				// verifies if already disconected
				break
			}
			//fire all OnPing methods
			c.fireOnPing()
			// try to ping the client, if failed then it disconnects
			err := c.Write(websocket.PingMessage, []byte{})
			if err != nil {
				// must stop to exit the loop and finish the go routine
				break
			}
		}
	}()
}

func (c *connection) fireOnPing() {
	// fire the onPingListeners
	for i := range c.onPingListeners {
		c.onPingListeners[i]()
	}
}

func (c *connection) fireOnPong() {
	// fire the onPongListeners
	for i := range c.onPongListeners {
		c.onPongListeners[i]()
	}
}

func (c *connection) startReader() {
	conn := c.underline
	hasReadTimeout := c.server.config.ReadTimeout > 0

	conn.SetReadLimit(c.server.config.MaxMessageSize)
	conn.SetPongHandler(func(s string) error {
		if hasReadTimeout {
			conn.SetReadDeadline(time.Now().Add(c.server.config.ReadTimeout))
		}
		//fire all OnPong methods
		go c.fireOnPong()

		return nil
	})

	defer func() {
		c.Disconnect()
	}()

	for {
		if hasReadTimeout {
			// set the read deadline based on the configuration
			conn.SetReadDeadline(time.Now().Add(c.server.config.ReadTimeout))
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.FireOnError(err)
			}
			break
		} else {
			c.messageReceived(data)
		}

	}

}

// messageReceived checks the incoming message and fire the nativeMessage listeners or the event listeners (ws custom message)
func (c *connection) messageReceived(data []byte) {

	if bytes.HasPrefix(data, c.server.config.EvtMessagePrefix) {
		//it's a custom ws message
		receivedEvt := c.server.messageSerializer.getWebsocketCustomEvent(data)
		listeners, ok := c.onEventListeners[string(receivedEvt)]
		if !ok || len(listeners) == 0 {
			return // if not listeners for this event exit from here
		}

		customMessage, err := c.server.messageSerializer.deserialize(receivedEvt, data)
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
					// here if server side waiting for string but client side sent an int, just convert this int to a string
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

func (c *connection) ID() string {
	return c.id
}

func (c *connection) Server() *Server {
	return c.server
}

func (c *connection) Context() context.Context {
	return c.ctx
}

func (c *connection) Values() ConnectionValues {
	return c.values
}

func (c *connection) fireDisconnect() {
	for i := range c.onDisconnectListeners {
		c.onDisconnectListeners[i]()
	}
}

func (c *connection) OnDisconnect(cb DisconnectFunc) {
	c.onDisconnectListeners = append(c.onDisconnectListeners, cb)
}

func (c *connection) OnError(cb ErrorFunc) {
	c.onErrorListeners = append(c.onErrorListeners, cb)
}

func (c *connection) OnPing(cb PingFunc) {
	c.onPingListeners = append(c.onPingListeners, cb)
}

func (c *connection) OnPong(cb PongFunc) {
	c.onPongListeners = append(c.onPongListeners, cb)
}

func (c *connection) FireOnError(err error) {
	for _, cb := range c.onErrorListeners {
		cb(err)
	}
}

func (c *connection) To(to string) Emitter {
	if to == Broadcast { // if send to all except me, then return the pre-defined emitter, and so on
		return c.broadcast
	} else if to == All {
		return c.all
	} else if to == c.id {
		return c.self
	}

	// is an emitter to another client/connection
	return newEmitter(c, to)
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
	c.server.Join(roomName, c.id)
}

func (c *connection) IsJoined(roomName string) bool {
	return c.server.IsJoined(roomName, c.id)
}

func (c *connection) Leave(roomName string) bool {
	return c.server.Leave(roomName, c.id)
}

func (c *connection) OnLeave(roomLeaveCb LeaveRoomFunc) {
	c.onRoomLeaveListeners = append(c.onRoomLeaveListeners, roomLeaveCb)
	// note: the callbacks are called from the server on the '.leave' and '.LeaveAll' funcs.
}

func (c *connection) fireOnLeave(roomName string) {
	// check if connection is already closed
	if c == nil {
		return
	}
	// fire the onRoomLeaveListeners
	for i := range c.onRoomLeaveListeners {
		c.onRoomLeaveListeners[i](roomName)
	}
}

// Wait starts the pinger and the messages reader,
// it's named as "Wait" because it should be called LAST,
// after the "On" events IF server's `Upgrade` is used,
// otherise you don't have to call it because the `Handler()` does it automatically.
func (c *connection) Wait() {
	if c.started {
		return
	}
	c.started = true
	// start the ping
	c.startPinger()

	// start the messages reader
	c.startReader()
}

// ErrAlreadyDisconnected can be reported on the `Connection#Disconnect` function whenever the caller tries to close the
// connection when it is already closed by the client or the caller previously.
var ErrAlreadyDisconnected = errors.New("already disconnected")

func (c *connection) Disconnect() error {
	if c == nil || c.disconnected {
		return ErrAlreadyDisconnected
	}
	return c.server.Disconnect(c.ID())
}

// mem per-conn store

func (c *connection) SetValue(key string, value interface{}) {
	c.values.Set(key, value)
}

func (c *connection) GetValue(key string) interface{} {
	return c.values.Get(key)
}

func (c *connection) GetValueArrString(key string) []string {
	if v := c.values.Get(key); v != nil {
		if arrString, ok := v.([]string); ok {
			return arrString
		}
	}
	return nil
}

func (c *connection) GetValueString(key string) string {
	if v := c.values.Get(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c *connection) GetValueInt(key string) int {
	if v := c.values.Get(key); v != nil {
		if i, ok := v.(int); ok {
			return i
		} else if s, ok := v.(string); ok {
			if iv, err := strconv.Atoi(s); err == nil {
				return iv
			}
		}
	}
	return 0
}
