package websocket

import (
	"bytes"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// Dial opens a new client connection to a WebSocket.
func Dial(url, evtMessagePrefix string) (ws *ClientConn, err error) {
	if !strings.HasPrefix(url, "ws://") {
		url = "ws://" + url
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	return NewClientConn(conn, evtMessagePrefix), nil
}

type ClientConn struct {
	underline   UnderlineConnection // TODO make it using gorilla's one, because the 'startReader' will not know when to stop otherwise, we have a fixed length currently...
	messageType int
	serializer  *messageSerializer

	onErrorListeners         []ErrorFunc
	onDisconnectListeners    []DisconnectFunc
	onNativeMessageListeners []NativeMessageFunc
	onEventListeners         map[string][]MessageFunc

	writerMu sync.Mutex

	disconnected uint32
}

func NewClientConn(conn UnderlineConnection, evtMessagePrefix string) *ClientConn {
	if evtMessagePrefix == "" {
		evtMessagePrefix = DefaultEvtMessageKey
	}

	c := &ClientConn{
		underline:  conn,
		serializer: newMessageSerializer([]byte(evtMessagePrefix)),

		onErrorListeners:         make([]ErrorFunc, 0),
		onDisconnectListeners:    make([]DisconnectFunc, 0),
		onNativeMessageListeners: make([]NativeMessageFunc, 0),
		onEventListeners:         make(map[string][]MessageFunc, 0),
	}

	c.SetBinaryMessages(false)

	go c.startReader()

	return c
}

func (c *ClientConn) SetBinaryMessages(binaryMessages bool) {
	if binaryMessages {
		c.messageType = websocket.BinaryMessage
	} else {
		c.messageType = websocket.TextMessage
	}
}

func (c *ClientConn) startReader() {
	defer c.Disconnect()

	for {
		_, data, err := c.underline.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.FireOnError(err)
			}

			break
		} else {
			c.messageReceived(data)
		}
	}
}

func (c *ClientConn) messageReceived(data []byte) error {
	if bytes.HasPrefix(data, c.serializer.prefix) {
		// is a custom iris message.
		receivedEvt := c.serializer.getWebsocketCustomEvent(data)
		listeners, ok := c.onEventListeners[string(receivedEvt)]
		if !ok || len(listeners) == 0 {
			return nil // if not listeners for this event exit from here
		}

		customMessage, err := c.serializer.deserialize(receivedEvt, data)
		if customMessage == nil || err != nil {
			return err
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

	return nil
}

func (c *ClientConn) OnMessage(cb NativeMessageFunc) {
	c.onNativeMessageListeners = append(c.onNativeMessageListeners, cb)
}

func (c *ClientConn) On(event string, cb MessageFunc) {
	if c.onEventListeners[event] == nil {
		c.onEventListeners[event] = make([]MessageFunc, 0)
	}

	c.onEventListeners[event] = append(c.onEventListeners[event], cb)
}

func (c *ClientConn) OnError(cb ErrorFunc) {
	c.onErrorListeners = append(c.onErrorListeners, cb)
}

func (c *ClientConn) FireOnError(err error) {
	for _, cb := range c.onErrorListeners {
		cb(err)
	}
}

func (c *ClientConn) OnDisconnect(cb DisconnectFunc) {
	c.onDisconnectListeners = append(c.onDisconnectListeners, cb)
}

func (c *ClientConn) Disconnect() error {
	if c == nil || !atomic.CompareAndSwapUint32(&c.disconnected, 0, 1) {
		return ErrAlreadyDisconnected
	}

	err := c.underline.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		err = c.underline.Close()
	}

	if err == nil {
		for i := range c.onDisconnectListeners {
			c.onDisconnectListeners[i]()
		}
	}

	return err
}

func (c *ClientConn) EmitMessage(nativeMessage []byte) error {
	return c.writeDefault(nativeMessage)
}

func (c *ClientConn) Emit(event string, data interface{}) error {
	b, err := c.serializer.serialize(event, data)
	if err != nil {
		return err
	}

	return c.EmitMessage(b)
}

// Write writes a raw websocket message with a specific type to the client
// used by ping messages and any CloseMessage types.
func (c *ClientConn) Write(websocketMessageType int, data []byte) error {
	// for any-case the app tries to write from different goroutines,
	// we must protect them because they're reporting that as bug...
	c.writerMu.Lock()
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
func (c *ClientConn) writeDefault(data []byte) error {
	return c.Write(c.messageType, data)
}
