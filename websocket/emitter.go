package websocket

const (
	// All is the string which the Emitter use to send a message to all
	All = ""
	// NotMe is the string which the Emitter use to send a message to all except this connection
	NotMe = ";iris;to;all;except;me;"
	// Broadcast is the string which the Emitter use to send a message to all except this connection, same as 'NotMe'
	Broadcast = NotMe
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

// emitter implementation

func newEmitter(c *connection, to string) *emitter {
	return &emitter{conn: c, to: to}
}

func (e *emitter) EmitMessage(nativeMessage []byte) error {
	mp := messagePayload{e.conn.id, e.to, nativeMessage}
	e.conn.server.messages <- mp
	return nil
}

func (e *emitter) Emit(event string, data interface{}) error {
	message, err := serialize(event, data)
	if err != nil {
		return err
	}
	e.EmitMessage([]byte(message))
	return nil
}

//
