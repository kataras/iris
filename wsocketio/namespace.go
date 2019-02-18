package wsocketio

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/googollee/go-socket.io/parser"
)

type namespaceHandler struct {
	onConnect    func(c Conn) error
	onDisconnect func(c Conn, msg string)
	onError      func(err error)
	events       map[string]*funcHandler
}

func newHandler() *namespaceHandler {
	return &namespaceHandler{
		events: make(map[string]*funcHandler),
	}
}

func (h *namespaceHandler) OnConnect(f func(Conn) error) {
	h.onConnect = f
}

func (h *namespaceHandler) OnDisconnect(f func(Conn, string)) {
	h.onDisconnect = f
}

func (h *namespaceHandler) OnError(f func(error)) {
	h.onError = f
}

func (h *namespaceHandler) OnEvent(event string, f interface{}) {
	h.events[event] = newEventFunc(f)
}

func (h *namespaceHandler) getTypes(header parser.Header, event string) []reflect.Type {
	switch header.Type {
	case parser.Error:
		fallthrough
	case parser.Disconnect:
		return []reflect.Type{reflect.TypeOf("")}
	case parser.Event:
		namespaceHandler := h.events[event]
		if namespaceHandler == nil {
			return nil
		}
		return namespaceHandler.argTypes
	}
	return nil
}

func (h *namespaceHandler) dispatch(c Conn, header parser.Header, event string, args []reflect.Value) ([]reflect.Value, error) {
	switch header.Type {
	case parser.Connect:
		var err error
		if h.onConnect != nil {
			err = h.onConnect(c)
		}
		return nil, err
	case parser.Disconnect:
		msg := ""
		if len(args) > 0 {
			msg = args[0].Interface().(string)
		}
		if h.onDisconnect != nil {
			h.onDisconnect(c, msg)
		}
		return nil, nil
	case parser.Error:
		msg := ""
		if len(args) > 0 {
			msg = args[0].Interface().(string)
		}
		if h.onError != nil {
			h.onError(errors.New(msg))
		}
	case parser.Event:
		namespaceHandler := h.events[event]
		if namespaceHandler == nil {
			return nil, nil
		}
		return namespaceHandler.Call(append([]reflect.Value{reflect.ValueOf(c)}, args...))
	}
	return nil, errors.New("invalid packet type")
}

type namespaceConn struct {
	*conn
	namespace string
	acks      sync.Map
	context   interface{}
}

func newNamespaceConn(conn *conn, namespace string) *namespaceConn {
	return &namespaceConn{
		conn:      conn,
		namespace: namespace,
		acks:      sync.Map{},
	}
}

func (c *namespaceConn) SetContext(v interface{}) {
	c.context = v
}

func (c *namespaceConn) Context() interface{} {
	return c.context
}

func (c *namespaceConn) Namespace() string {
	return c.namespace
}

func (c *namespaceConn) Emit(event string, v ...interface{}) {
	header := parser.Header{
		Type: parser.Event,
	}
	if c.namespace != "/" {
		header.Namespace = c.namespace
	}

	if l := len(v); l > 0 {
		last := v[l-1]
		lastV := reflect.TypeOf(last)
		if lastV.Kind() == reflect.Func {
			f := newAckFunc(last)
			header.ID = c.conn.nextID()
			header.NeedAck = true
			c.acks.Store(header.ID, f)
			v = v[:l-1]
		}
	}

	args := make([]reflect.Value, len(v)+1)
	args[0] = reflect.ValueOf(event)
	for i := 1; i < len(args); i++ {
		args[i] = reflect.ValueOf(v[i-1])
	}
	c.conn.write(header, args)
}

func (c *namespaceConn) dispatch(header parser.Header) {
	if header.Type != parser.Ack {
		return
	}

	rawFunc, ok := c.acks.Load(header.ID)
	if ok {
		f, ok := rawFunc.(*funcHandler)
		if !ok {
			c.conn.onError(c.namespace, fmt.Errorf("incorrect data stored for header %d", header.ID))
			return
		}
		c.acks.Delete(header.ID)
		args, err := c.conn.parseArgs(f.argTypes)
		if err != nil {
			c.conn.onError(c.namespace, err)
			return
		}
		if _, err := f.Call(args); err != nil {
			c.conn.onError(c.namespace, err)
			return
		}
	}
	return
}
