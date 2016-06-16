package config

import (
	"time"

	"github.com/imdario/mergo"
)

// Currently only these 5 values are used for real
const (
	// DefaultWriteTimeout 30 * time.Second
	DefaultWriteTimeout = 30 * time.Second
	// DefaultPongTimeout 120 * time.Second
	DefaultPongTimeout = 120 * time.Second
	// DefaultPingPeriod (DefaultPongTimeout * 9) / 10
	DefaultPingPeriod = (DefaultPongTimeout * 9) / 10
	// DefaultMaxMessageSize 2048
	DefaultMaxMessageSize = 2048
)

//

// Websocket the config contains options for 'websocket' package
type Websocket struct {
	// WriteTimeout time allowed to write a message to the connection.
	// Default value is 30 * time.Second
	WriteTimeout time.Duration
	// PongTimeout allowed to read the next pong message from the connection
	// Default value is 120 * time.Second
	PongTimeout time.Duration
	// PingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
	// Default value is (PongTimeout * 9) / 10
	PingPeriod time.Duration
	// MaxMessageSize max message size allowed from connection
	// Default value is 2048
	MaxMessageSize int64
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
