package websocket

import (
	"github.com/imdario/mergo"
	"net/http"
	"time"
)

const (
	// DefaultWebsocketWriteTimeout 15 * time.Second
	DefaultWebsocketWriteTimeout = 15 * time.Second
	// DefaultWebsocketPongTimeout 60 * time.Second
	DefaultWebsocketPongTimeout = 60 * time.Second
	// DefaultWebsocketPingPeriod (DefaultPongTimeout * 9) / 10
	DefaultWebsocketPingPeriod = (DefaultWebsocketPongTimeout * 9) / 10
	// DefaultWebsocketMaxMessageSize 1024
	DefaultWebsocketMaxMessageSize = 1024
	// DefaultWebsocketReadBufferSize 4096
	DefaultWebsocketReadBufferSize = 4096
	// DefaultWebsocketWriterBufferSize 4096
	DefaultWebsocketWriterBufferSize = 4096
)

type (
	// OptionSetter sets a configuration field to the websocket config
	// used to help developers to write less and configure only what they really want and nothing else
	OptionSetter interface {
		// Set receives a pointer to the global Config type and does the job of filling it
		Set(c *Config)
	}
	// OptionSet implements the OptionSetter
	OptionSet func(c *Config)
)

// Set is the func which makes the OptionSet an OptionSetter, this is used mostly
func (o OptionSet) Set(c *Config) {
	o(c)
}

// Config the websocket server configuration
// all of these are optional.
type Config struct {
	Error       func(res http.ResponseWriter, req *http.Request, status int, reason error)
	CheckOrigin func(req *http.Request) bool
	// WriteTimeout time allowed to write a message to the connection.
	// Default value is 15 * time.Second
	WriteTimeout time.Duration
	// PongTimeout allowed to read the next pong message from the connection
	// Default value is 60 * time.Second
	PongTimeout time.Duration
	// PingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
	// Default value is (PongTimeout * 9) / 10
	PingPeriod time.Duration
	// MaxMessageSize max message size allowed from connection
	// Default value is 1024
	MaxMessageSize int64
	// BinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
	// compatible if you wanna use the Connection's EmitMessage to send a custom binary data to the client, like a native server-client communication.
	// defaults to false
	BinaryMessages bool
	// ReadBufferSize is the buffer size for the underline reader
	ReadBufferSize int
	// WriteBufferSize is the buffer size for the underline writer
	WriteBufferSize int
}

// Set is the func which makes the OptionSet an OptionSetter, this is used mostly
func (c Config) Set(main *Config) {
	mergo.MergeWithOverwrite(main, c)
}

// Error sets the error handler
func Error(val func(res http.ResponseWriter, req *http.Request, status int, reason error)) OptionSet {
	return func(c *Config) {
		c.Error = val
	}
}

// CheckOrigin sets a handler which will check if different origin(domains) are allowed to contact with
// the websocket server
func CheckOrigin(val func(req *http.Request) bool) OptionSet {
	return func(c *Config) {
		c.CheckOrigin = val
	}
}

// WriteTimeout time allowed to write a message to the connection.
// Default value is 15 * time.Second
func WriteTimeout(val time.Duration) OptionSet {
	return func(c *Config) {
		c.WriteTimeout = val
	}
}

// PongTimeout allowed to read the next pong message from the connection
// Default value is 60 * time.Second
func PongTimeout(val time.Duration) OptionSet {
	return func(c *Config) {
		c.PongTimeout = val
	}
}

// PingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
// Default value is (PongTimeout * 9) / 10
func PingPeriod(val time.Duration) OptionSet {
	return func(c *Config) {
		c.PingPeriod = val
	}
}

// MaxMessageSize max message size allowed from connection
// Default value is 1024
func MaxMessageSize(val int64) OptionSet {
	return func(c *Config) {
		c.MaxMessageSize = val
	}
}

// BinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
// compatible if you wanna use the Connection's EmitMessage to send a custom binary data to the client, like a native server-client communication.
// defaults to false
func BinaryMessages(val bool) OptionSet {
	return func(c *Config) {
		c.BinaryMessages = val
	}
}

// ReadBufferSize is the buffer size for the underline reader
func ReadBufferSize(val int) OptionSet {
	return func(c *Config) {
		c.ReadBufferSize = val
	}
}

// WriteBufferSize is the buffer size for the underline writer
func WriteBufferSize(val int) OptionSet {
	return func(c *Config) {
		c.WriteBufferSize = val
	}
}

// Validate validates the configuration
func (c Config) Validate() Config {
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = DefaultWebsocketWriteTimeout
	}
	if c.PongTimeout <= 0 {
		c.PongTimeout = DefaultWebsocketPongTimeout
	}
	if c.PingPeriod <= 0 {
		c.PingPeriod = DefaultWebsocketPingPeriod
	}
	if c.MaxMessageSize <= 0 {
		c.MaxMessageSize = DefaultWebsocketMaxMessageSize
	}
	if c.ReadBufferSize <= 0 {
		c.ReadBufferSize = DefaultWebsocketReadBufferSize
	}
	if c.WriteBufferSize <= 0 {
		c.WriteBufferSize = DefaultWebsocketWriterBufferSize
	}
	if c.Error == nil {
		c.Error = func(res http.ResponseWriter, req *http.Request, status int, reason error) {
			//http.Error(res, reason.Error(), status)
		}
	}
	if c.CheckOrigin == nil {
		c.CheckOrigin = func(req *http.Request) bool {
			// allow all connections by default
			return true
		}
	}

	return c
}
