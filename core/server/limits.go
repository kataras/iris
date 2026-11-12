package server

import (
	"net/http"
)

// LimitsOptions is used to configure the underline net/http server's request limits.
//
// These matter more since Go 1.27: every TLS-enabled Iris server offers "h2" through
// its tls.Config NextProtos, and the HTTP/2 server now honors RFC 9218 client-driven
// response prioritization on every h2 connection unless DisableClientPriority is set.
// In the same release the server started rejecting, with a 400, requests that repeat
// a header name more times than MaxHeaderValueCount allows.
//
// Its fields default to 0 (or false), which keeps the standard library's defaults.
// See the `LimitsOptions.ApplyToServer` method and the `ServerOptions.Limits` field.
type LimitsOptions struct {
	// MaxHeaderBytes controls the maximum number of bytes the server will
	// read parsing the request header's keys and values, including the
	// request line. It does not limit the size of the request body.
	//
	// A zero value keeps the standard library's DefaultMaxHeaderBytes (1MB).
	MaxHeaderBytes int `ini:"max_header_bytes" json:"max_header_bytes,omitzero" yaml:"MaxHeaderBytes" toml:"MaxHeaderBytes"`

	// MaxHeaderValueCount controls the maximum number of header values
	// a request may contain for any single header name. Since Go 1.27 the
	// server responds with 400 Bad Request when a client exceeds it, which
	// guards against repeated-header floods; set this field only to loosen
	// or tighten that guard.
	//
	// A zero value keeps the standard library's DefaultMaxHeaderValueCount.
	MaxHeaderValueCount int `ini:"max_header_value_count" json:"max_header_value_count,omitzero" yaml:"MaxHeaderValueCount" toml:"MaxHeaderValueCount"`

	// DisableClientPriority opts out of RFC 9218 client-specified response
	// prioritization on HTTP/2 connections. Every TLS Iris server offers "h2",
	// so since Go 1.27 this scheduling is active by default; disable it when
	// clients must not influence the order responses are written in.
	DisableClientPriority bool `ini:"disable_client_priority" json:"disable_client_priority,omitzero" yaml:"DisableClientPriority" toml:"DisableClientPriority"`
}

// ApplyToServer sets the limits to the given standard http.Server instance,
// respecting any customization already made to the server:
// a field is written only when the option is set (non-zero or true)
// and the server's own field still holds its zero value.
func (c *LimitsOptions) ApplyToServer(srv *http.Server) {
	if c == nil {
		return
	}

	if v := c.MaxHeaderBytes; v > 0 && srv.MaxHeaderBytes == 0 {
		srv.MaxHeaderBytes = v
	}

	if v := c.MaxHeaderValueCount; v > 0 && srv.MaxHeaderValueCount == 0 {
		srv.MaxHeaderValueCount = v
	}

	if c.DisableClientPriority && !srv.DisableClientPriority {
		srv.DisableClientPriority = true
	}
}
