package logger

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/kataras/iris.v6"
)

type loggerMiddleware struct {
	config Config
}

// Serve serves the middleware
func (l *loggerMiddleware) Serve(ctx *iris.Context) {
	//all except latency to string
	var date, status, ip, method, path string
	var latency time.Duration
	var startTime, endTime time.Time
	path = ctx.Path()
	method = ctx.Method()

	startTime = time.Now()

	ctx.Next()
	//no time.Since in order to format it well after
	endTime = time.Now()
	date = endTime.Format("01/02 - 15:04:05")
	latency = endTime.Sub(startTime)

	if l.config.Status {
		status = strconv.Itoa(ctx.ResponseWriter.StatusCode())
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

	//finally print the logs
	ctx.Log(iris.DevMode, fmt.Sprintf("%s %v %4v %s %s %s \n", date, status, latency, ip, method, path))
}

// New returns the logger middleware
// receives optional configs(logger.Config)
func New(cfg ...Config) iris.HandlerFunc {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	l := &loggerMiddleware{config: c}

	return l.Serve
}
