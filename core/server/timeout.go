package server

import (
	"net/http"
	"time"
)

// TimeoutOptions is used to configure the server's timeouts.
//
// The right timeouts for an HTTP lifecycle decide how the server behaves under load and how quickly a stuck client is released. Some general guidelines:
//
// Read Headers Timeout: This should be relatively short, as header data is typically small and quick to transmit.
// A common timeout is between 1-5 seconds.
// Read Timeout: This depends on the expected size of the body content and headers.
//
//	For small to medium-sized bodies, 5-10 seconds might be sufficient.
//	For larger bodies, especially if you expect large file uploads, you might need to increase this to 30 seconds or more.
//
// Idle Connection Timeout: This is the time a connection stays open waiting for new requests.
// Common values range from 5 seconds to 120 seconds1. Apache’s default, for example, is 5 seconds.
// Server Write Timeout: This is the time your server will try to send a response.
// This can be similar to the read body timeout, but you might want to allow extra time in case of slow clients.
// A starting point could be 10 seconds.
//
// Its fields default to 0.
// Initialize a new timeout configuration through the `Timeout` package-level function.
type TimeoutOptions struct {
	// ShutdownOnInterruptTimeout is the maximum duration for the server to wait for
	// the server to shutdown on OS interrupt signal (e.g. CTRL+C).
	// A zero or negative value means there will be no timeout.
	//
	// Defaults to 15 seconds.
	ShutdownOnInterruptTimeout time.Duration `ini:"shutdown_on_interrupt_timeout" json:"shutdown_on_interrupt_timeout,omitzero" yaml:"ShutdownOnInterruptTimeout" toml:"ShutdownOnInterruptTimeout"`

	// ServerTimeout is the underline net/http server's read/write timeout.
	// if greater than zero, framework automatically calculates and sets the server's read headers,
	// idle connection and write timeouts. We don't automatically set the read timeout (which includes the read body timeout),
	// in order for handlers to decide by theirselves (e.g. big file upload and/or slow clients).
	// Follow up with the fields below to learn more about these timeouts.
	ServerTimeout time.Duration `ini:"server_timeout" json:"server_timeout,omitzero" yaml:"ServerTimeout" toml:"ServerTimeout"`

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body. A zero or negative value means
	// there will be no timeout.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration `ini:"read_timeout" json:"read_timeout,omitzero" yaml:"ReadTimeout" toml:"ReadTimeout"`

	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers. The connection's read deadline is reset
	// after reading the headers and the Handler can decide what
	// is considered too slow for the body. If ReadHeaderTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	ReadHeaderTimeout time.Duration `ini:"read_header_timeout" json:"read_header_timeout,omitzero" yaml:"ReadHeaderTimeout" toml:"ReadHeaderTimeout"`

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	// A zero or negative value means there will be no timeout.
	WriteTimeout time.Duration `ini:"write_timeout" json:"write_timeout,omitzero" yaml:"WriteTimeout" toml:"WriteTimeout"`

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration `ini:"idle_timeout" json:"idle_timeout,omitzero" yaml:"IdleTimeout" toml:"IdleTimeout"`

	// HandlerTimeout wraps the application's router with an http timeout handler
	// if the value is greater than zero.
	// It wraps the router (except when Downgrade is used) under the standard net/http.TimeoutHandler.
	//
	// The underline response writer supports the Pusher interface but does not support
	// the Hijacker or Flusher interfaces when Timeout handler is registered.
	//
	// Read more at: https://pkg.go.dev/net/http#TimeoutHandler.
	//
	// See the `HandlerTimeoutMessage` field to customize the displayed message too.
	HandlerTimeout time.Duration `ini:"handler_timeout" json:"handler_timeout,omitzero" yaml:"HandlerTimeout" toml:"HandlerTimeout"`
	// HandlerTimeoutMessage is the message (HTML) to be sent when the handler's timeout reached.
	HandlerTimeoutMessage string `ini:"handler_timeout_message" json:"handler_timeout_message,omitempty" yaml:"HandlerTimeoutMessage" toml:"HandlerTimeoutMessage"`
}

// DefaultTimeoutMessage is the default timeout message which is rendered
// on expired handlers when timeout handler is registered (see Timeout configuration field).
var DefaultTimeoutMessage = `<html><head><title>Timeout</title></head><body><h1>Timeout</h1>Looks like the server is taking too long to respond, this can be caused by either poor connectivity or an error with our servers. Please try again in a while.</body></html>`

// Timeout returns a new TimeoutOptions, as it works as a builder.
func Timeout() *TimeoutOptions {
	return &TimeoutOptions{
		ShutdownOnInterruptTimeout: 15 * time.Second,
		HandlerTimeoutMessage:      DefaultTimeoutMessage,
	}
}

// Get returns the TimeoutOptions itself if it's not nil, otherwise it returns the default TimeoutOptions.
func (c *TimeoutOptions) Get() *TimeoutOptions {
	if c == nil {
		return Timeout() // if it's nil, get defaults.
	}

	return c // otherwise return itself.
}

// WithShutdownOnInterruptTimeout sets the maximum duration for the server to wait for
// the server to shutdown on OS interrupt signal (e.g. CTRL+C).
// A zero or negative value means there will be no timeout.
//
// Defaults to 15 seconds.
func (c *TimeoutOptions) WithShutdownOnInterruptTimeout(timeout time.Duration) *TimeoutOptions {
	c.ShutdownOnInterruptTimeout = timeout
	return c
}

// WithServerTimeout sets the server's read/write timeout.
// Overrides the read header, idle connection and write timeouts if they are not set.
// The read (body) timeout is deliberately left alone, so handlers can decide by theirselves
// (e.g. big file upload and/or slow clients); use the `WithReadTimeout` method to set it.
func (c *TimeoutOptions) WithServerTimeout(timeout time.Duration) *TimeoutOptions {
	c.ServerTimeout = timeout
	return c
}

// WithReadTimeout sets the server's read headers and body timeout.
func (c *TimeoutOptions) WithReadTimeout(timeout time.Duration) *TimeoutOptions {
	c.ReadTimeout = timeout
	return c
}

// WithReadHeaderTimeout sets the server's read headers timeout.
func (c *TimeoutOptions) WithReadHeaderTimeout(timeout time.Duration) *TimeoutOptions {
	c.ReadHeaderTimeout = timeout
	return c
}

// WithWriteTimeout sets the server's write timeout.
func (c *TimeoutOptions) WithWriteTimeout(timeout time.Duration) *TimeoutOptions {
	c.WriteTimeout = timeout
	return c
}

// WithIdleTimeout sets the server's idle connection timeout.
func (c *TimeoutOptions) WithIdleTimeout(timeout time.Duration) *TimeoutOptions {
	c.IdleTimeout = timeout
	return c
}

func (c *TimeoutOptions) ApplyToServer(srv *http.Server) {
	if c == nil {
		return
	}

	// initialize by calculation of the server timeout option,
	// if zero then all of these are zero and the developer can set them manually
	// through the rest of the fields (see below).
	readHeader, _, write, idle := CalculateTimeouts(c.ServerTimeout)
	if v := c.ReadHeaderTimeout; v > 0 {
		readHeader = v
	}

	// Do not set globlal read timeout, it includes body too, let it be customizable by handlers.
	// if v := c.ReadTimeout; v > 0 {
	// 	readTimeout = v
	// }

	if v := c.WriteTimeout; v > 0 {
		write = v
	}

	if v := c.IdleTimeout; v > 0 {
		idle = v
	}

	// now check if no other customization was made to the server and if not, set the timeouts.
	if srv.ReadHeaderTimeout == 0 {
		srv.ReadHeaderTimeout = readHeader
	}

	if srv.ReadTimeout == 0 {
		srv.ReadTimeout = c.ReadTimeout
	}

	if srv.WriteTimeout == 0 {
		srv.WriteTimeout = write
	}

	if srv.IdleTimeout == 0 {
		srv.IdleTimeout = idle
	}
}

func (c *TimeoutOptions) WithHandlerTimeout(timeout time.Duration, htmlBody ...string) *TimeoutOptions {
	c.HandlerTimeout = timeout
	if len(htmlBody) > 0 {
		c.HandlerTimeoutMessage = htmlBody[0]
	}
	return c
}

func (c *TimeoutOptions) TimeoutHandler(mainHandler http.Handler) http.Handler {
	if c == nil {
		return mainHandler
	}

	if c.HandlerTimeout <= 0 {
		return mainHandler
	}

	handlerToWrap := mainHandler

	return http.TimeoutHandler(handlerToWrap, c.HandlerTimeout, c.HandlerTimeoutMessage)
}

// CalculateTimeouts calculates the read and write timeouts based on the total lifecycle timeout.
//
// Define the total timeout for the response-request lifecycle. This is the maximum amount of time the entire process should take.
// Divide the total timeout into portions for the read, write, and processing time.
// A common approach is to allocate 50% of the time to reading, 25% to processing, and 25% to writing.
// Set a minimum threshold for both read and write timeouts to ensure they are not too short, which could cause premature timeout errors.
// Define the idle timeout as a percentage of the total lifecycle timeout or as a fixed value, depending on your requirements.
// Ensure that the idle timeout is not greater than the read and write timeouts, as it should be a period of inactivity within reasonable bounds.
//
// The total lifecycle timeout is divided into:
// 50% for read timeout
// 25% for processing time (not directly used but subtracted from the total to calculate write timeout)
// 25% for write timeout (calculated as the remainder after allocating read timeout and processing time)
// The read header timeout is set to 50% of the read timeout.
// The idle timeout is set to 75% of the read timeout.
// Minimum thresholds are applied to ensure that none of the timeouts fall below a certain value.
func CalculateTimeouts(totalLifecycleTimeout time.Duration) (readHeaderTimeout, readTimeout, writeTimeout, idleTimeout time.Duration) {
	// Ensure the total timeout is greater than 0.
	if totalLifecycleTimeout <= 0 {
		return 0, 0, 0, 0
	}

	// Define minimum thresholds.
	const (
		minReadHeaderTimeout = 2 * time.Second
		minReadTimeout       = 5 * time.Second
		minWriteTimeout      = 5 * time.Second
		minIdleTimeout       = 5 * time.Second
	)

	// Calculate half for read, a quarter for processing, and a quarter for write.
	processingTime := totalLifecycleTimeout / 4
	readTimeout = totalLifecycleTimeout / 2
	writeTimeout = totalLifecycleTimeout - readTimeout - processingTime

	// Calculate read header timeout as a percentage of the read timeout
	// Here we use 50% of the read timeout as the read header timeout.
	readHeaderTimeout = readTimeout / 2

	// Calculate idle timeout as a percentage of the read timeout
	// Here we use 75% of the read timeout as the idle timeout.
	idleTimeout = (readTimeout * 75) / 100

	// Apply minimum thresholds
	if readHeaderTimeout < minReadHeaderTimeout {
		readHeaderTimeout = minReadHeaderTimeout
	}
	if readTimeout < minReadTimeout {
		readTimeout = minReadTimeout
	}
	if writeTimeout < minWriteTimeout {
		writeTimeout = minWriteTimeout
	}
	if idleTimeout < minIdleTimeout || idleTimeout > readTimeout {
		idleTimeout = minIdleTimeout
	}

	return readHeaderTimeout, readTimeout, writeTimeout, idleTimeout
}
