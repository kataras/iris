package ws1m

const (
	// All is the string which the Emitter use to send a message to all.
	All = ""
	// Broadcast is the string which the Emitter use to send a message to all except this connection.
	Broadcast = ";to;all;except;me;"
)

type (
	// Emitter is the message/or/event manager
	Emitter interface {
		// EmitMessage sends a native websocket message
		EmitMessage([]byte) error
		// Emit sends a message on a particular event
		Emit(string, interface{}) error
	}

	emitter struct {
		conn *connection
		to   string
	}
)

var _ Emitter = &emitter{}

func newEmitter(c *connection, to string) *emitter {
	return &emitter{conn: c, to: to}
}

func (e *emitter) EmitMessage(nativeMessage []byte) error {
	//e.conn.mSerializerMu.Lock()
	e.conn.server.emitMessage(e.conn.id, e.to, nativeMessage)
	//-->Writedefault ->Write
	//	e.conn.mSerializerMu.Unlock()
	return nil
}

func (e *emitter) Emit(event string, data interface{}) error {
	//message, err := e.conn.server.messageSerializer.serialize(event, data)
	message, err := e.serialize_k(event, data)
	if err != nil {
		return err
	}
	e.EmitMessage(message)
	return nil
}

func (e *emitter) serialize_k(event string, data interface{}) ([]byte, error) {
	//e.conn.mSerializerMu.Lock()
	message, err := e.conn.server.messageSerializer.serialize(event, data)
	//e.conn.mSerializerMu.Unlock()
	return message, err
}
