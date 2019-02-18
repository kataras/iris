package websocket

import (
	"bytes"
	stdContext "context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/context"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// Operation codes defined by specification.
// See https://tools.ietf.org/html/rfc6455#section-5.2
const (
	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage ws.OpCode = ws.OpText
	// BinaryMessage denotes a binary data message.
	BinaryMessage ws.OpCode = ws.OpBinary
	// CloseMessage denotes a close control message.
	CloseMessage ws.OpCode = ws.OpClose

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage ws.OpCode = ws.OpPing
	// PongMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage ws.OpCode = ws.OpPong
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
	// Connection is the front-end API that you will use to communicate with the client side,
	// it is the server-side connection.
	Connection interface {
		ClientConnection
		// Err is not nil if the upgrader failed to upgrade http to websocket connection.
		Err() error
		// ID returns the connection's identifier
		ID() string
		// Server returns the websocket server instance
		// which this connection is listening to.
		//
		// Its connection-relative operations are safe for use.
		Server() *Server
		// Context returns the (upgraded) context.Context of this connection
		// avoid using it, you normally don't need it,
		// websocket has everything you need to authenticate the user BUT if it's necessary
		// then  you use it to receive user information, for example: from headers
		Context() context.Context
		// To defines on what "room" (see Join) the server should send a message
		// returns an Emmiter(`EmitMessage` & `Emit`) to send messages.
		To(string) Emitter
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

	// ClientConnection is the client-side connection interface. Server shares some of its methods but the underline actions differs.
	ClientConnection interface {
		Emitter
		// Write writes a raw websocket message with a specific type to the client
		// used by ping messages and any CloseMessage types.
		Write(websocketMessageType ws.OpCode, data []byte) error
		// OnMessage registers a callback which fires when native websocket message received
		OnMessage(NativeMessageFunc)
		// On registers a callback to a particular event which is fired when a message to this event is received
		On(string, MessageFunc)
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
		// OnDisconnect registers a callback which is fired when this connection is closed by an error or manual
		OnDisconnect(DisconnectFunc)
		// Disconnect disconnects the client, close the underline websocket conn and removes it from the conn list
		// returns the error, if any, from the underline connection
		Disconnect() error
	}

	connection struct {
		err                error
		underline          net.Conn
		config             ConnectionConfig
		defaultMessageType ws.OpCode
		serializer         *messageSerializer
		id                 string

		onErrorListeners         []ErrorFunc
		onPingListeners          []PingFunc
		onPongListeners          []PongFunc
		onNativeMessageListeners []NativeMessageFunc
		onEventListeners         map[string][]MessageFunc
		onRoomLeaveListeners     []LeaveRoomFunc
		onDisconnectListeners    []DisconnectFunc
		disconnected             uint32

		started bool
		// these were  maden for performance only
		self      Emitter // pre-defined emitter than sends message to its self client
		broadcast Emitter // pre-defined emitter that sends message to all except this
		all       Emitter // pre-defined emitter which sends message to all clients

		// access to the Context, use with caution, you can't use response writer as you imagine.
		ctx    context.Context
		values ConnectionValues
		server *Server

		writer *wsutil.Writer

		// #119 , websocket writers are not protected by locks inside the gorilla's websocket code
		// so we must protect them otherwise we're getting concurrent connection error on multi writers in the same time.
		writerMu sync.Mutex
		// same exists for reader look here: https://godoc.org/github.com/gorilla/websocket#hdr-Control_Messages
		// but we only use one reader in one goroutine, so we are safe.
		// readerMu sync.Mutex
	}
)

var _ Connection = &connection{}

// WrapConnection wraps the underline websocket connection into a new iris websocket connection.
// The caller should call the `connection#Wait` (which blocks) to enable its read and write functionality.
func WrapConnection(conn net.Conn, cfg ConnectionConfig) Connection {
	return newConnection(conn, cfg)
}

func newConnection(conn net.Conn, cfg ConnectionConfig) *connection {
	cfg = cfg.Validate()
	c := &connection{
		underline:                conn,
		config:                   cfg,
		serializer:               newMessageSerializer(cfg.EvtMessagePrefix),
		defaultMessageType:       TextMessage,
		onErrorListeners:         make([]ErrorFunc, 0),
		onPingListeners:          make([]PingFunc, 0),
		onPongListeners:          make([]PongFunc, 0),
		onNativeMessageListeners: make([]NativeMessageFunc, 0),
		onEventListeners:         make(map[string][]MessageFunc, 0),
		onDisconnectListeners:    make([]DisconnectFunc, 0),
		disconnected:             0,
	}

	if cfg.BinaryMessages {
		c.defaultMessageType = BinaryMessage
	}

	// c.writer = wsutil.NewWriter(conn, c.getState(), c.defaultMessageType)

	return c
}

func newServerConnection(ctx context.Context, s *Server, conn net.Conn, id string) *connection {
	c := newConnection(conn, ConnectionConfig{
		EvtMessagePrefix:  s.config.EvtMessagePrefix,
		WriteTimeout:      s.config.WriteTimeout,
		ReadTimeout:       s.config.ReadTimeout,
		PingPeriod:        s.config.PingPeriod,
		BinaryMessages:    s.config.BinaryMessages,
		ReadBufferSize:    s.config.ReadBufferSize,
		WriteBufferSize:   s.config.WriteBufferSize,
		EnableCompression: s.config.EnableCompression,
	})

	c.id = id
	c.server = s
	c.ctx = ctx
	c.onRoomLeaveListeners = make([]LeaveRoomFunc, 0)
	c.started = false

	c.self = newEmitter(c, c.id)
	c.broadcast = newEmitter(c, Broadcast)
	c.all = newEmitter(c, All)

	return c
}

// Err is not nil if the upgrader failed to upgrade http to websocket connection.
func (c *connection) Err() error {
	return c.err
}

// IsClient returns true if that connection is from client.
func (c *connection) getState() ws.State {
	if c.server != nil {
		// server-side.
		return ws.StateServerSide
	}

	// else return client-side.
	return ws.StateClientSide
}

// Write writes a raw websocket message with a specific type to the client
// used by ping messages and any CloseMessage types.
func (c *connection) Write(websocketMessageType ws.OpCode, data []byte) (err error) {
	// for any-case the app tries to write from different goroutines,
	// we must protect them because they're reporting that as bug...
	c.writerMu.Lock()
	defer c.writerMu.Unlock()
	if writeTimeout := c.config.WriteTimeout; writeTimeout > 0 {
		// set the write deadline based on the configuration
		c.underline.SetWriteDeadline(time.Now().Add(writeTimeout))
	}

	// 2.
	// if websocketMessageType != c.defaultMessageType {
	// 	err = wsutil.WriteMessage(c.underline, c.getState(), websocketMessageType, data)
	// } else {
	// 	_, err = c.writer.Write(data)
	// 	c.writer.Flush()
	// }

	err = wsutil.WriteMessage(c.underline, c.getState(), websocketMessageType, data)

	if err != nil {
		// if failed then the connection is off, fire the disconnect
		c.Disconnect()
	}
	return err
}

// writeDefault is the same as write but the message type is the configured by c.messageType
// if BinaryMessages is enabled then it's raw []byte as you expected to work with protobufs
func (c *connection) writeDefault(data []byte) error {
	return c.Write(c.defaultMessageType, data)
}

func (c *connection) startPinger() {
	if c.config.PingPeriod > 0 {
		go func() {
			for {
				time.Sleep(c.config.PingPeriod)
				if c == nil || atomic.LoadUint32(&c.disconnected) > 0 {
					// verifies if already disconected.
					return
				}

				// try to ping the client, if failed then it disconnects.
				err := c.Write(PingMessage, []byte{})
				if err != nil && !c.isErrClosed(err) {
					c.FireOnError(err)
					// must stop to exit the loop and exit from the routine.
					return
				}

				//fire all OnPing methods
				c.fireOnPing()

			}
		}()
	}
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

func (c *connection) isErrClosed(err error) bool {
	if err == nil {
		return false
	}

	_, is := err.(wsutil.ClosedError)
	if is {
		return true
	}

	if opErr, is := err.(*net.OpError); is {
		if opErr.Err == io.EOF {
			return false
		}

		if atomic.LoadUint32(&c.disconnected) == 0 {
			c.Disconnect()
		}

		return true
	}

	return err != io.EOF
}

func (c *connection) startReader() {
	defer c.Disconnect()

	hasReadTimeout := c.config.ReadTimeout > 0

	controlHandler := wsutil.ControlFrameHandler(c.underline, c.getState())
	rd := wsutil.Reader{
		Source:          c.underline,
		State:           c.getState(),
		CheckUTF8:       false,
		SkipHeaderCheck: false,
		OnIntermediate:  controlHandler,
	}

	for {
		if hasReadTimeout {
			// set the read deadline based on the configuration
			c.underline.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		}

		hdr, err := rd.NextFrame()
		if err != nil {
			return
		}
		if hdr.OpCode.IsControl() {
			if err := controlHandler(hdr, &rd); err != nil {
				return
			}
			continue
		}

		if hdr.OpCode&TextMessage == 0 && hdr.OpCode&BinaryMessage == 0 {
			if err := rd.Discard(); err != nil {
				return
			}
			continue
		}

		data, err := ioutil.ReadAll(&rd)
		if err != nil {
			return
		}

		c.messageReceived(data)

		// 4.
		// var buf bytes.Buffer
		// data, code, err := wsutil.ReadData(struct {
		// 	io.Reader
		// 	io.Writer
		// }{c.underline, &buf}, c.getState())
		// if err != nil {
		// 	if _, closed := err.(*net.OpError); closed && code == 0 {
		// 		c.Disconnect()
		// 		return
		// 	} else if _, closed = err.(wsutil.ClosedError); closed {
		// 		c.Disconnect()
		// 		return
		// 		// > 1200 conns but I don't know why yet:
		// 	} else if err == ws.ErrProtocolOpCodeReserved || err == ws.ErrProtocolNonZeroRsv {
		// 		c.Disconnect()
		// 		return
		// 	} else if err == io.EOF || err == io.ErrUnexpectedEOF {
		// 		c.Disconnect()
		// 		return
		// 	}

		// 	c.FireOnError(err)
		// }

		// c.messageReceived(data)

		// 2.
		// header, err := reader.NextFrame()
		// if err != nil {
		// 	println("next frame err: " + err.Error())
		// 	return
		// }

		// if header.OpCode == ws.OpClose { // io.EOF.
		// 	return
		// }
		// payload := make([]byte, header.Length)
		// _, err = io.ReadFull(reader, payload)
		// if err != nil {
		// 	return
		// }

		// if header.Masked {
		// 	ws.Cipher(payload, header.Mask, 0)
		// }

		// c.messageReceived(payload)

		// data, code, err := wsutil.ReadData(c.underline, c.getState())
		// // if code == CloseMessage || c.isErrClosed(err) {
		// // 	c.Disconnect()
		// // 	return
		// // }

		// if err != nil {
		// 	if _, closed := err.(*net.OpError); closed && code == 0 {
		// 		c.Disconnect()
		// 		return
		// 	} else if _, closed = err.(wsutil.ClosedError); closed {
		// 		c.Disconnect()
		// 		return
		// 		// > 1200 conns but I don't know why yet:
		// 	} else if err == ws.ErrProtocolOpCodeReserved || err == ws.ErrProtocolNonZeroRsv {
		// 		c.Disconnect()
		// 		return
		// 	} else if err == io.EOF || err == io.ErrUnexpectedEOF {
		// 		c.Disconnect()
		// 		return
		// 	}

		// 	c.FireOnError(err)
		// }

		// c.messageReceived(data)
	}

}

// messageReceived checks the incoming message and fire the nativeMessage listeners or the event listeners (ws custom message)
func (c *connection) messageReceived(data []byte) {

	if bytes.HasPrefix(data, c.config.EvtMessagePrefix) {
		//it's a custom ws message
		receivedEvt := c.serializer.getWebsocketCustomEvent(data)
		listeners, ok := c.onEventListeners[string(receivedEvt)]
		if !ok || len(listeners) == 0 {
			return // if not listeners for this event exit from here
		}

		customMessage, err := c.serializer.deserialize(receivedEvt, data)
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
	if c.server != nil {
		return c.self.EmitMessage(nativeMessage)
	}
	return c.writeDefault(nativeMessage)
}

func (c *connection) Emit(event string, message interface{}) error {
	if c.server != nil {
		return c.self.Emit(event, message)
	}

	b, err := c.serializer.serialize(event, message)
	if err != nil {
		return err
	}

	return c.EmitMessage(b)
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
	if c == nil || !atomic.CompareAndSwapUint32(&c.disconnected, 0, 1) {
		return ErrAlreadyDisconnected
	}

	if c.server != nil {
		return c.server.Disconnect(c.ID())
	}

	err := c.Write(CloseMessage, nil)

	if err == nil {
		c.fireDisconnect()
	}

	c.underline.Close()

	return err
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

// ConnectionConfig is the base configuration for both server and client connections.
// Clients must use `ConnectionConfig` in order to `Dial`, server's connection configuration is set by the `Config` structure.
type ConnectionConfig struct {
	// EvtMessagePrefix is the prefix of the underline websocket events that are being established under the hoods.
	// This prefix is visible only to the javascript side (code) and it has nothing to do
	// with the message that the end-user receives.
	// Do not change it unless it is absolutely necessary.
	//
	// If empty then defaults to []byte("iris-websocket-message:").
	// Should match with the server's EvtMessagePrefix.
	EvtMessagePrefix []byte
	// WriteTimeout time allowed to write a message to the connection.
	// 0 means no timeout.
	// Default value is 0
	WriteTimeout time.Duration
	// ReadTimeout time allowed to read a message from the connection.
	// 0 means no timeout.
	// Default value is 0
	ReadTimeout time.Duration
	// PingPeriod send ping messages to the connection repeatedly after this period.
	// The value should be close to the ReadTimeout to avoid issues.
	// Default value is 0
	PingPeriod time.Duration
	// BinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
	// compatible if you wanna use the Connection's EmitMessage to send a custom binary data to the client, like a native server-client communication.
	// Default value is false
	BinaryMessages bool
	// ReadBufferSize is the buffer size for the connection reader.
	// Default value is 4096
	ReadBufferSize int
	// WriteBufferSize is the buffer size for the connection writer.
	// Default value is 4096
	WriteBufferSize int
	// EnableCompression specify if the server should attempt to negotiate per
	// message compression (RFC 7692). Setting this value to true does not
	// guarantee that compression will be supported. Currently only "no context
	// takeover" modes are supported.
	//
	// Defaults to false and it should be remain as it is, unless special requirements.
	EnableCompression bool
}

// Validate validates the connection configuration.
func (c ConnectionConfig) Validate() ConnectionConfig {
	if len(c.EvtMessagePrefix) == 0 {
		c.EvtMessagePrefix = []byte(DefaultEvtMessageKey)
	}

	// 0 means no timeout.
	if c.WriteTimeout < 0 {
		c.WriteTimeout = DefaultWebsocketWriteTimeout
	}

	if c.ReadTimeout < 0 {
		c.ReadTimeout = DefaultWebsocketReadTimeout
	}

	if c.PingPeriod <= 0 {
		c.PingPeriod = DefaultWebsocketPingPeriod
	}

	if c.ReadBufferSize <= 0 {
		c.ReadBufferSize = DefaultWebsocketReadBufferSize
	}

	if c.WriteBufferSize <= 0 {
		c.WriteBufferSize = DefaultWebsocketWriterBufferSize
	}

	return c
}

// ErrBadHandshake is returned when the server response to opening handshake is
// invalid.
var ErrBadHandshake = ws.ErrHandshakeBadConnection

// Dial creates a new client connection.
//
// The context will be used in the request and in the Dialer.
//
// If the WebSocket handshake fails, `ErrHandshakeBadConnection` is returned.
//
// The "url" input parameter is the url to connect to the server, it should be
// the ws:// (or wss:// if secure) + the host + the endpoint of the
// open socket of the server, i.e ws://localhost:8080/my_websocket_endpoint.
//
// Custom dialers can be used by wrapping the iris websocket connection via `websocket.WrapConnection`.
func Dial(ctx stdContext.Context, url string, cfg ConnectionConfig) (ClientConnection, error) {
	c, err := dial(ctx, url, cfg)
	if err != nil {
		time.Sleep(1 * time.Second)
		c, err = dial(ctx, url, cfg)
	}

	return c, err
}

func dial(ctx stdContext.Context, url string, cfg ConnectionConfig) (ClientConnection, error) {
	if ctx == nil {
		ctx = stdContext.Background()
	}

	if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		url = "ws://" + url
	}

	conn, _, _, err := ws.DefaultDialer.Dial(ctx, url)
	if err != nil {
		return nil, err
	}

	clientConn := WrapConnection(conn, cfg)
	go clientConn.Wait()

	return clientConn, nil
}
