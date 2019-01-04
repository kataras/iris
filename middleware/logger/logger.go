// Package logger provides request logging via middleware. See _examples/http_request/request-logger
package logger

import (
	"fmt"
	"strconv"
	"time"

	"github.com/kataras/iris/context"

	"github.com/ryanuber/columnize"
)

type requestLoggerMiddleware struct {
	config Config
}

// New creates and returns a new request logger middleware.
// Do not confuse it with the framework's Logger.
// This is for the http requests.
//
// Receives an optional configuation.
func New(cfg ...Config) context.Handler {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.buildSkipper()
	l := &requestLoggerMiddleware{config: c}

	return l.ServeHTTP
}

// Serve serves the middleware
func (l *requestLoggerMiddleware) ServeHTTP(ctx context.Context) {
	// skip logs and serve the main request immediately
	if l.config.skip != nil {
		if l.config.skip(ctx) {
			ctx.Next()
			return
		}
	}

	//all except latency to string
	var status, ip, method, path string
	var latency time.Duration
	var startTime, endTime time.Time
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

	if l.config.Method {
		method = ctx.Method()
	}

	if l.config.Path {
		if l.config.Query {
			path = ctx.Request().URL.RequestURI()
		} else {
			path = ctx.Path()
		}
	}

	var message interface{}
	if ctxKeys := l.config.MessageContextKeys; len(ctxKeys) > 0 {
		for _, key := range ctxKeys {
			msg := ctx.Values().Get(key)
			if message == nil {
				message = msg
			} else {
				message = fmt.Sprintf(" %v %v", message, msg)
			}
		}
	}
	var headerMessage interface{}
	if headerKeys := l.config.MessageHeaderKeys; len(headerKeys) > 0 {
		for _, key := range headerKeys {
			msg := ctx.GetHeader(key)
			if headerMessage == nil {
				headerMessage = msg
			} else {
				headerMessage = fmt.Sprintf(" %v %v", headerMessage, msg)
			}
		}
	}

	// print the logs
	if logFunc := l.config.LogFunc; logFunc != nil {
		logFunc(endTime, latency, status, ip, method, path, message, headerMessage)
		return
	}

	if l.config.Columns {
		endTimeFormatted := endTime.Format("2006/01/02 - 15:04:05")
		output := Columnize(endTimeFormatted, latency, status, ip, method, path, message, headerMessage)
		ctx.Application().Logger().Printer.Output.Write([]byte(output))
		return
	}
	// no new line, the framework's logger is responsible how to render each log.
	line := fmt.Sprintf("%v %4v %s %s %s", status, latency, ip, method, path)
	if message != nil {
		line += fmt.Sprintf(" %v", message)
	}

	if headerMessage != nil {
		line += fmt.Sprintf(" %v", headerMessage)
	}
	ctx.Application().Logger().Info(line)
}

// Columnize formats the given arguments as columns and returns the formatted output,
// note that it appends a new line to the end.
func Columnize(nowFormatted string, latency time.Duration, status, ip, method, path string, message interface{}, headerMessage interface{}) string {

	titles := "Time | Status | Latency | IP | Method | Path"
	line := fmt.Sprintf("%s | %v | %4v | %s | %s | %s", nowFormatted, status, latency, ip, method, path)
	if message != nil {
		titles += " | Message"
		line += fmt.Sprintf(" | %v", message)
	}

	if headerMessage != nil {
		titles += " | HeaderMessage"
		line += fmt.Sprintf(" | %v", headerMessage)
	}

	outputC := []string{
		titles,
		line,
	}
	output := columnize.SimpleFormat(outputC) + "\n"
	return output
}
