package logger

import (
	"time"

	"github.com/kataras/iris/context"
)

// The SkipperFunc signature, used to serve the main request without logs.
// See `Configuration` too.
type SkipperFunc func(ctx context.Context) bool

// Config are the options of the logger middlweare
// contains 4 bools
// Status, IP, Method, Path
// if set to true then these will print
type Config struct {
	// Status displays status code (bool)
	//
	// Defaults to true
	Status bool
	// IP displays request's remote address (bool)
	//
	// Defaults to true
	IP bool
	// Method displays the http method (bool)
	//
	// Defaults to true
	Method bool
	// Path displays the request path (bool)
	//
	// Defaults to true
	Path bool
	// Columns will display the logs as well formatted columns (bool)
	// If custom `LogFunc` has been provided then this field is useless and users should
	// use the `Columinize` function of the logger to get the ouput result as columns.
	//
	// Defaults to true
	Columns bool
	// LogFunc is the writer which logs are written to,
	// if missing the logger middleware uses the app.Logger().Infof instead.
	LogFunc func(now time.Time, latency time.Duration, status, ip, method, path string)
	// Skippers used to skip the logging i.e by `ctx.Path()` and serve
	// the next/main handler immediately.
	Skippers []SkipperFunc
	// the Skippers as one function in order to reduce the time needed to
	// combine them at serve time.
	skip SkipperFunc
}

// DefaultConfiguration returns a default configuration
// that have all boolean fields to true,
// LogFunc to nil,
// and Skippers to nil.
func DefaultConfiguration() Config {
	return Config{true, true, true, true, true, nil, nil, nil}
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
