// Package logger provides request logging via middleware. See _examples/http_request/request-logger
package logger

import (
	"strconv"
	"time"

	"github.com/kataras/iris/context"
)

type requestLoggerMiddleware struct {
	config Config
}

// Serve serves the middleware
func (l *requestLoggerMiddleware) ServeHTTP(ctx context.Context) {
	//all except latency to string
	var status, ip, method, path string
	var latency time.Duration
	var startTime, endTime time.Time
	path = ctx.Path()
	method = ctx.Method()

	startTime = time.Now()

	ctx.Next()
	//no time.Since in order to format it well after
	endTime = time.Now()
	latency = endTime.Sub(startTime)

	if l.config.Status {
		status = strconv.Itoa(ctx.GetStatusCode())
	}

	if l.config.IP {
		ip = ctx.RemoteAddr()
	}

	if !l.config.Method {
		method = ""
	}

	if !l.config.Path {
		path = ""
	}

	//finally print the logs, no new line, the framework's logger is responsible how to render each log.
	ctx.Application().Logger().Infof("%v %4v %s %s %s", status, latency, ip, method, path)
}

// New creates and returns a new request logger middleware.
// Do not confuse it with the framework's Logger.
// This is for the http requests.
//
// Receives an optional configuation.
func New(cfg ...Config) context.Handler {
	c := DefaultConfiguration()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	l := &requestLoggerMiddleware{config: c}

	return l.ServeHTTP
}
