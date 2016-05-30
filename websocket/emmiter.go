package websocket

const (
	// All is the string which the Emmiter use to send a message to all
	All = ""
	// NotMe is the string which the Emmiter use to send a message to all except this connection
	NotMe = ";iris;to;all;except;me;"
	// Broadcast is the string which the Emmiter use to send a message to all except this connection, same as 'NotMe'
	Broadcast = NotMe
)

type (
	// Emmiter is the message/or/event manager
	Emmiter interface {
		// EmitMessage sends a native websocket message
		EmitMessage([]byte) error
		// Emit sends a message on a particular event
		Emit(string, interface{}) error
	}

	emmiter struct {
		conn *connection
		to   string
	}
)

var _ Emmiter = &emmiter{}

// emmiter implementation

func newEmmiter(c *connection, to string) *emmiter {
	return &emmiter{conn: c, to: to}
}

func (e *emmiter) EmitMessage(nativeMessage []byte) error {
	mp := messagePayload{e.conn.id, e.to, nativeMessage}
	e.conn.server.messages <- mp
	return nil
}

func (e *emmiter) Emit(event string, data interface{}) error {
	message, err := serialize(event, data)
	if err != nil {
		return err
	}
	e.EmitMessage([]byte(message))
	return nil
}

//
