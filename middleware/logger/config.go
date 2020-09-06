package logger

import (
	"time"

	"github.com/kataras/iris/v12/context"
)

// The SkipperFunc signature, used to serve the main request without logs.
// See `Configuration` too.
type SkipperFunc func(ctx *context.Context) bool

// Config contains the options for the logger middleware
// can be optionally be passed to the `New`.
type Config struct {
	// Status displays status code (bool).
	//
	// Defaults to true.
	Status bool
	// IP displays request's remote address (bool).
	//
	// Defaults to true.
	IP bool
	// Method displays the http method (bool).
	//
	// Defaults to true.
	Method bool
	// Path displays the request path (bool).
	// See `Query` and `PathAfterHandler` too.
	//
	// Defaults to true.
	Path bool
	// PathAfterHandler displays the request path
	// which may be set and modified
	// after the handler chain is executed.
	// See `Query` too.
	//
	// Defaults to false.
	PathAfterHandler bool
	// Query will append the URL Query to the Path.
	// Path should be true too.
	//
	// Defaults to false.
	Query bool
	// TraceRoute displays the debug
	// information about the current route executed.
	//
	// Defaults to false.
	TraceRoute bool

	// MessageContextKeys if not empty,
	// the middleware will try to fetch
	// the contents with `ctx.Values().Get(MessageContextKey)`
	// and if available then these contents will be
	// appended as part of the logs (with `%v`, in order to be able to set a struct too),
	//
	// Defaults to empty.
	MessageContextKeys []string

	// MessageHeaderKeys if not empty,
	// the middleware will try to fetch
	// the contents with `ctx.Values().Get(MessageHeaderKey)`
	// and if available then these contents will be
	// appended as part of the logs (with `%v`, in order to be able to set a struct too),
	//
	// Defaults to empty.
	MessageHeaderKeys []string

	// LogFunc is the writer which logs are written to,
	// if missing the logger middleware uses the app.Logger().Infof instead.
	// Note that message argument can be empty.
	LogFunc func(endTime time.Time, latency time.Duration, status, ip, method, path string, message interface{}, headerMessage interface{})
	// LogFuncCtx can be used instead of `LogFunc` if handlers need to customize the output based on
	// custom request-time information that the LogFunc isn't aware of.
	LogFuncCtx func(ctx *context.Context, latency time.Duration)
	// Skippers used to skip the logging i.e by `ctx.Path()` and serve
	// the next/main handler immediately.
	Skippers []SkipperFunc
	// the Skippers as one function in order to reduce the time needed to
	// combine them at serve time.
	skip SkipperFunc
}

// DefaultConfig returns a default config
// that have all boolean fields to true,
// all strings are empty,
// LogFunc and Skippers to nil as well.
func DefaultConfig() Config {
	return Config{
		Status:           true,
		IP:               true,
		Method:           true,
		Path:             true,
		PathAfterHandler: false,
		Query:            false,
		TraceRoute:       false,
		LogFunc:          nil,
		LogFuncCtx:       nil,
		Skippers:         nil,
		skip:             nil,
	}
}

// AddSkipper adds a skipper to the configuration.
func (c *Config) AddSkipper(sk SkipperFunc) {
	c.Skippers = append(c.Skippers, sk)
	c.buildSkipper()
}

func (c *Config) buildSkipper() {
	if len(c.Skippers) == 0 {
		return
	}
	skippersLocked := c.Skippers[0:]
	c.skip = func(ctx *context.Context) bool {
		for _, s := range skippersLocked {
			if s(ctx) {
				return true
			}
		}
		return false
	}
}
