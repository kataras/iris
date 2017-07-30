package logger

import (
	"time"

	"github.com/kataras/iris/context"
)

// The SkipperFunc signature, used to serve the main request without logs.
// See `Configuration` too.
type SkipperFunc func(ctx context.Context) bool

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
	//
	// Defaults to true.
	Path bool

	// Columns will display the logs as a formatted columns-rows text (bool).
	// If custom `LogFunc` has been provided then this field is useless and users should
	// use the `Columinize` function of the logger to get the output result as columns.
	//
	// Defaults to false.
	Columns bool

	// MessageContextKey if not empty,
	// the middleware will try to fetch
	// the contents with `ctx.Values().Get(MessageContextKey)`
	// and if available then these contents will be
	// appended as part of the logs (with `%v`, in order to be able to set a struct too),
	// if Columns field was setted to true then
	// a new column will be added named 'Message'.
	//
	// Defaults to empty.
	MessageContextKey string

	// LogFunc is the writer which logs are written to,
	// if missing the logger middleware uses the app.Logger().Infof instead.
	// Note that message argument can be empty.
	LogFunc func(now time.Time, latency time.Duration, status, ip, method, path string, message interface{})
	// Skippers used to skip the logging i.e by `ctx.Path()` and serve
	// the next/main handler immediately.
	Skippers []SkipperFunc
	// the Skippers as one function in order to reduce the time needed to
	// combine them at serve time.
	skip SkipperFunc
}

// DefaultConfig returns a default config
// that have all boolean fields to true except `Columns`,
// all strings are empty,
// LogFunc and Skippers to nil as well.
func DefaultConfig() Config {
	return Config{
		Status:            true,
		IP:                true,
		Method:            true,
		Path:              true,
		Columns:           false,
		MessageContextKey: "",
		LogFunc:           nil,
		Skippers:          nil,
		skip:              nil,
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
	c.skip = func(ctx context.Context) bool {
		for _, s := range skippersLocked {
			if s(ctx) {
				return true
			}
		}
		return false
	}
}
