package websocket

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
	e.conn.server.emitMessage(e.conn.id, e.to, nativeMessage)
	return nil
}

func (e *emitter) Emit(event string, data interface{}) error {
	message, err := e.conn.serializer.serialize(event, data)
	if err != nil {
		return err
	}
	e.EmitMessage(message)
	return nil
}
