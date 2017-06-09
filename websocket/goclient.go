// Copyright 2017 Joseph deBlaquiere. All rights reserved.
// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bytes"
	"crypto/tls"
	// "fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	gwebsocket "github.com/gorilla/websocket"
)

// Client presents a subset of iris.websocket.Connection interface to support
// client-initiated connections using the Iris websocket message protocol
type client struct {
	conn                     *gwebsocket.Conn
	config                   Config
	wAbort                   chan bool
	wchan                    chan []byte
	pchan                    chan []byte
	onDisconnectListeners    []DisconnectFunc
	onErrorListeners         []ErrorFunc
	onNativeMessageListeners []NativeMessageFunc
	onEventListeners         map[string][]MessageFunc
	connected                bool
	dMutex                   sync.Mutex
}

// ClientConnection defines proper subset of Connection interface which is
// satisfied by Client
type ClientConnection interface {
	// EmitMessage sends a native websocket message
	EmitMessage([]byte) error
	// Emit sends a message on a particular event
	Emit(string, interface{}) error

	// OnDisconnect registers a callback which fires when this connection is closed by an error or manual
	OnDisconnect(DisconnectFunc)

	// OnMessage registers a callback which fires when native websocket message received
	OnMessage(NativeMessageFunc)
	// On registers a callback to a particular event which fires when a message to this event received
	On(string, MessageFunc)
	// Disconnect disconnects the client, close the underline websocket conn and removes it from the conn list
	// returns the error, if any, from the underline connection
	Disconnect() error
}

// in order to ensure all read operations are within a single goroutine
// readPump processes incoming messages and dispatches them to messageReceived
func (c *client) readPump() {
	defer c.conn.Close()
	c.conn.SetReadLimit(c.config.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	c.conn.SetPongHandler(func(s string) error {
		// fmt.Printf("received PONG from %s\n", c.conn.UnderlyingConn().RemoteAddr().String())
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		return nil
	})
	c.conn.SetPingHandler(func(s string) error {
		// fmt.Printf("received PING (%s) from %s\n", s, c.conn.UnderlyingConn().RemoteAddr().String())
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		c.pchan <- []byte(s)
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// fmt.Println("disconnect @ ", time.Now().Format("2006-01-02 15:04:05.000000"))
			c.wAbort <- false
			return
		}
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))

		// fmt.Println("recv:", string(message), "@", time.Now().Format("2006-01-02 15:04:05.000000"))

		go c.messageReceived(message)
	}
}

func (c *client) fireDisconnect() {
	c.dMutex.Lock()
	defer c.dMutex.Unlock()
	if c.connected == false {
		return
	}
	// fmt.Println("fireDisconnect unique")
	for i := range c.onDisconnectListeners {
		c.onDisconnectListeners[i]()
	}
	c.connected = false
}

// messageReceived comes straight from iris/adapters/websocket/connection.go
// messageReceived checks the incoming message and fire the nativeMessage listeners or the event listeners (ws custom message)
func (c *client) messageReceived(data []byte) {

	if bytes.HasPrefix(data, websocketMessagePrefixBytes) {
		customData := string(data)
		//it's a custom ws message
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

// In order to ensure all write operations are within a single goroutine
// writePump handles write operations to the socket serially from channels
func (c *client) writePump() {
	pingtimer := time.NewTicker(c.config.PingPeriod)
	defer c.conn.Close()
	c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
	for {
		select {
		case wmsg := <-c.wchan:
			// fmt.Printf("WP: writing %s\n", string(wmsg))
			w, err := c.conn.NextWriter(gwebsocket.TextMessage)
			if err != nil {
				// fmt.Println("error getting NextWriter")
				c.fireDisconnect()
				return
			}
			w.Write(wmsg)

			if err := w.Close(); err != nil {
				// fmt.Println("error closing NextWriter")
				c.fireDisconnect()
				return
			}

		case pmsg := <-c.pchan:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			// fmt.Printf("sending PONG to %s\n", c.conn.UnderlyingConn().RemoteAddr().String())
			if err := c.conn.WriteControl(gwebsocket.PongMessage, pmsg, time.Now().Add(c.config.WriteTimeout)); err != nil {
				// fmt.Println("error sending PONG")
				c.fireDisconnect()
				return
			}

		// any write to wAbort aborts writePump
		case sendClose := <-c.wAbort:
			// fmt.Println("wAbort received")
			if sendClose {
				c.conn.WriteControl(gwebsocket.CloseMessage, gwebsocket.FormatCloseMessage(gwebsocket.CloseNormalClosure, ""),
					time.Now().Add(c.config.WriteTimeout))
			}
			c.fireDisconnect()
			return

		case <-pingtimer.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			// fmt.Printf("sending PING to %s\n", c.conn.UnderlyingConn().RemoteAddr().String())
			if err := c.conn.WriteControl(gwebsocket.PingMessage, []byte{}, time.Now().Add(c.config.WriteTimeout)); err != nil {
				// fmt.Println("error sending PING")
				c.fireDisconnect()
				return
			}
		}
	}
}

// EmitMessage sends a native (raw) message to the socket. The server will
// receive this message using the handler specified with OnMessage()
func (c *client) EmitMessage(nativeMessage []byte) error {
	c.wchan <- nativeMessage
	return nil
}

// Emit sends a message to a particular event queue. The server will receive
// these messages via the handler specified using On()
func (c *client) Emit(event string, data interface{}) error {
	message, err := websocketMessageSerialize(event, data)
	if err != nil {
		return err
	}
	// fmt.Printf("Message %s\n", string(message))
	c.wchan <- []byte(message)
	return nil
}

func (c *client) OnDisconnect(f DisconnectFunc) {
	c.onDisconnectListeners = append(c.onDisconnectListeners, f)
}

//func (c *client) OnError(f ErrorFunc) {
//
//}

// OnMessage designates a listener callback function for raw messages. If
// multiple callback functions are specified, all will be called for each
// message
func (c *client) OnMessage(f NativeMessageFunc) {
	c.onNativeMessageListeners = append(c.onNativeMessageListeners, f)
}

// On designates a listener callback for a specific event tag.  If multiple
// callback functions are specified, all will be called for each message
func (c *client) On(event string, f MessageFunc) {
	if c.onEventListeners[event] == nil {
		c.onEventListeners[event] = make([]MessageFunc, 0)
	}

	c.onEventListeners[event] = append(c.onEventListeners[event], f)
}

func (c *client) Disconnect() error {
	c.wAbort <- true
	return nil
}

// WSDialer here is a shameless wrapper around gorilla.websocket.Dialer
// which returns a wsclient.Client instead of the gorilla Connection on Dial()
type WSDialer struct {
	// NetDial specifies the dial function for creating TCP connections. If
	// NetDial is nil, net.Dial is used.
	NetDial func(network, addr string) (net.Conn, error)

	// Proxy specifies a function to return a proxy for a given
	// Request. If the function returns a non-nil error, the
	// request is aborted with the provided error.
	// If Proxy is nil or returns a nil *URL, no proxy is used.
	Proxy func(*http.Request) (*url.URL, error)

	// TLSClientConfig specifies the TLS configuration to use with tls.Client.
	// If nil, the default configuration is used.
	TLSClientConfig *tls.Config

	// HandshakeTimeout specifies the duration for the handshake to complete.
	HandshakeTimeout time.Duration

	// ReadBufferSize and WriteBufferSize specify I/O buffer sizes. If a buffer
	// size is zero, then a useful default size is used. The I/O buffer sizes
	// do not limit the size of the messages that can be sent or received.
	ReadBufferSize, WriteBufferSize int

	// Subprotocols specifies the client's requested subprotocols.
	Subprotocols []string

	// EnableCompression specifies if the client should attempt to negotiate
	// per message compression (RFC 7692). Setting this value to true does not
	// guarantee that compression will be supported. Currently only "no context
	// takeover" modes are supported.
	EnableCompression bool

	// Jar specifies the cookie jar.
	// If Jar is nil, cookies are not sent in requests and ignored
	// in responses.
	Jar http.CookieJar

	dialer *gwebsocket.Dialer
}

// Dial initiates a connection to a remote Iris server websocket listener
// using the gorilla websocket Dialer and returns a Client connection
// which can be used to emit and handle messages
func (wsd *WSDialer) Dial(urlStr string, requestHeader http.Header, config Config) (ClientConnection, *http.Response, error) {
	if wsd.dialer == nil {
		wsd.dialer = new(gwebsocket.Dialer)
	}
	wsd.dialer.NetDial = wsd.NetDial
	wsd.dialer.Proxy = wsd.Proxy
	wsd.dialer.TLSClientConfig = wsd.TLSClientConfig
	wsd.dialer.HandshakeTimeout = wsd.HandshakeTimeout
	wsd.dialer.ReadBufferSize = wsd.ReadBufferSize
	wsd.dialer.WriteBufferSize = wsd.WriteBufferSize
	wsd.dialer.Subprotocols = wsd.Subprotocols
	wsd.dialer.EnableCompression = wsd.EnableCompression
	wsd.dialer.Jar = wsd.Jar
	conn, response, err := wsd.dialer.Dial(urlStr, requestHeader)
	if err != nil {
		return nil, response, err
	}
	c := new(client)
	c.conn = conn
	c.config = config
	c.config.Validate()
	c.wAbort = make(chan bool)
	c.wchan = make(chan []byte)
	c.pchan = make(chan []byte)
	c.onEventListeners = make(map[string][]MessageFunc)
	c.config.Validate()
	c.connected = true

	go c.writePump()
	go c.readPump()

	return c, response, nil
}
