package config

import (
	"time"

	"github.com/imdario/mergo"
)

const (
	// DefaultWriteTimeout 15 * time.Second
	DefaultWriteTimeout = 15 * time.Second
	// DefaultPongTimeout 60 * time.Second
	DefaultPongTimeout = 60 * time.Second
	// DefaultPingPeriod (DefaultPongTimeout * 9) / 10
	DefaultPingPeriod = (DefaultPongTimeout * 9) / 10
	// DefaultMaxMessageSize 1024
	DefaultMaxMessageSize = 1024
)

//

// Websocket the config contains options for the ../websocket.go
type Websocket struct {
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
	// see https://github.com/kataras/iris/issues/387#issuecomment-243006022 for more
	// defaults to false
	BinaryMessages bool
	// Endpoint is the path which the websocket server will listen for clients/connections
	// Default value is empty string, if you don't set it the Websocket server is disabled.
	Endpoint string
	// Headers  the response headers before upgrader
	// Default is empty
	Headers map[string]string
	// ReadBufferSize is the buffer size for the underline reader
	ReadBufferSize int
	// WriteBufferSize is the buffer size for the underline writer
	WriteBufferSize int
}

// DefaultWebsocket returns the default config for iris-ws websocket package
func DefaultWebsocket() *Websocket {
	return &Websocket{
		WriteTimeout:    DefaultWriteTimeout,
		PongTimeout:     DefaultPongTimeout,
		PingPeriod:      DefaultPingPeriod,
		MaxMessageSize:  DefaultMaxMessageSize,
		BinaryMessages:  false,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		Headers:         make(map[string]string, 0),
		Endpoint:        "",
	}
}

// Merge merges the default with the given config and returns the result
func (c *Websocket) Merge(cfg []*Websocket) (config *Websocket) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(config, c)
	} else {
		_default := c
		config = _default
	}

	return
}

// MergeSingle merges the default with the given config and returns the result
func (c *Websocket) MergeSingle(cfg *Websocket) (config *Websocket) {

	config = cfg
	mergo.Merge(config, c)

	return
}
