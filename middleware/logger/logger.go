package logger

import (
	"strconv"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/logger"
)

// Options are the options of the logger middlweare
// contains 5 bools
// Latency, Status, IP, Method, Path
// if set to true then these will print
type Options struct {
	// Latency displays latency (bool)
	Latency bool
	// Status displays status code (bool)
	Status bool
	// IP displays request's remote address (bool)
	IP bool
	// Method displays the http method (bool)
	Method bool
	// Path displays the request path (bool)
	Path bool
}

// DefaultOptions returns an options which all properties are true
func DefaultOptions() Options {
	return Options{true, true, true, true, true}
}

type loggerMiddleware struct {
	*logger.Logger
	options Options
}

// Serve serves the middleware
func (l *loggerMiddleware) Serve(ctx *iris.Context) {
	//all except latency to string
	var date, status, ip, method, path string
	var latency time.Duration
	var startTime, endTime time.Time
	path = ctx.PathString()
	method = ctx.MethodString()

	if l.options.Latency {
		startTime = time.Now()
	}

	ctx.Next()
	if l.options.Latency {
		//no time.Since in order to format it well after
		endTime = time.Now()
		date = endTime.Format("01/02 - 15:04:05")
		latency = endTime.Sub(startTime)
	}

	if l.options.Status {
		status = strconv.Itoa(ctx.Response.StatusCode())
	}

	if l.options.IP {
		ip = ctx.RemoteAddr()
	}

	if !l.options.Method {
		method = ""
	}

	if !l.options.Path {
		path = ""
	}

	//finally print the logs
	if l.options.Latency {
		l.Otherf("%s %v %4v %s %s %s \n", date, status, latency, ip, method, path)
	} else {
		l.Otherf("%s %v %s %s %s \n", date, status, ip, method, path)
	}

}

// Default returns the logger middleware as Handler with the default settings
func New(theLogger *logger.Logger, options ...Options) iris.HandlerFunc {
	if theLogger == nil {
		theLogger = logger.New(config.DefaultLogger())
	}

	l := &loggerMiddleware{Logger: theLogger}

	if len(options) > 0 {
		l.options = options[0]
	} else {
		l.options = DefaultOptions()
	}

	return l.Serve
}
