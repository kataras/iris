package config

import (
	"io"
	"os"

	"github.com/imdario/mergo"
)

// Default values for base Iris conf
const (
	DefaultDisablePathCorrection = false
	DefaultDisablePathEscape     = false
	DefaultCharset               = "UTF-8"
	DefaultLoggerPreffix         = "[IRIS] "
)

var (
	DefaultLoggerOut = os.Stdout
)

type (
	// Iris configs for the station
	Iris struct {

		// DisablePathCorrection corrects and redirects the requested path to the registed path
		// for example, if /home/ path is requested but no handler for this Route found,
		// then the Router checks if /home handler exists, if yes,
		// (permant)redirects the client to the correct path /home
		//
		// Default is false
		DisablePathCorrection bool

		// DisablePathEscape when is false then its escapes the path, the named parameters (if any).
		// Change to true it if you want something like this https://github.com/kataras/iris/issues/135 to work
		//
		// When do you need to Disable(true) it:
		// accepts parameters with slash '/'
		// Request: http://localhost:8080/details/Project%2FDelta
		// ctx.Param("project") returns the raw named parameter: Project%2FDelta
		// which you can escape it manually with net/url:
		// projectName, _ := url.QueryUnescape(c.Param("project").
		// Look here: https://github.com/kataras/iris/issues/135 for more
		//
		// Default is false
		DisablePathEscape bool

		// DisableBanner outputs the iris banner at startup
		//
		// Default is false
		DisableBanner bool

		// LoggerOut is the destination for output
		//
		// defaults to os.Stdout
		LoggerOut io.Writer
		// LoggerOut is the logger's prefix to write at beginning of each line
		//
		// Defaults to [IRIS]
		LoggerPreffix string

		// ProfilePath a the route path, set it to enable http pprof tool
		// Default is empty, if you set it to a $path, these routes will handled:
		// $path/cmdline
		// $path/profile
		// $path/symbol
		// $path/goroutine
		// $path/heap
		// $path/threadcreate
		// $path/pprof/block
		// for example if '/debug/pprof'
		// http://yourdomain:PORT/debug/pprof/
		// http://yourdomain:PORT/debug/pprof/cmdline
		// http://yourdomain:PORT/debug/pprof/profile
		// http://yourdomain:PORT/debug/pprof/symbol
		// http://yourdomain:PORT/debug/pprof/goroutine
		// http://yourdomain:PORT/debug/pprof/heap
		// http://yourdomain:PORT/debug/pprof/threadcreate
		// http://yourdomain:PORT/debug/pprof/pprof/block
		// it can be a subdomain also, for example, if 'debug.'
		// http://debug.yourdomain:PORT/
		// http://debug.yourdomain:PORT/cmdline
		// http://debug.yourdomain:PORT/profile
		// http://debug.yourdomain:PORT/symbol
		// http://debug.yourdomain:PORT/goroutine
		// http://debug.yourdomain:PORT/heap
		// http://debug.yourdomain:PORT/threadcreate
		// http://debug.yourdomain:PORT/pprof/block
		ProfilePath string
		// DisableTemplateEngines set to true to disable loading the default template engine (html/template) and disallow the use of iris.UseEngine
		// default is false
		DisableTemplateEngines bool
		// IsDevelopment iris will act like a developer, for example
		// If true then re-builds the templates on each request
		// default is false
		IsDevelopment bool

		// Charset character encoding for various rendering
		// used for templates and the rest of the responses
		// defaults to "UTF-8"
		Charset string

		// Gzip enables gzip compression on your Render actions, this includes any type of render, templates and pure/raw content
		// If you don't want to enable it globaly, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
		// defaults to false
		Gzip bool

		// Sessions contains the configs for sessions
		Sessions Sessions

		// Websocket contains the configs for Websocket's server integration
		Websocket *Websocket

		// Tester contains the configs for the test framework, so far we have only one because all test framework's configs are setted by the iris itself
		// You can find example on the https://github.com/kataras/iris/glob/master/context_test.go
		Tester Tester
	}
)

// Default returns the default configuration for the Iris staton
func Default() Iris {
	return Iris{
		DisablePathCorrection:  DefaultDisablePathCorrection,
		DisablePathEscape:      DefaultDisablePathEscape,
		DisableBanner:          false,
		LoggerOut:              DefaultLoggerOut,
		LoggerPreffix:          DefaultLoggerPreffix,
		DisableTemplateEngines: false,
		IsDevelopment:          false,
		Charset:                DefaultCharset,
		Gzip:                   false,
		ProfilePath:            "",
		Sessions:               DefaultSessions(),
		Websocket:              DefaultWebsocket(),
		Tester:                 DefaultTester(),
	}
}

// Merge merges the default with the given config and returns the result
// receives an array because the func caller is variadic
func (c Iris) Merge(cfg []Iris) (config Iris) {
	// I tried to make it more generic with interfaces for all configs, inside config.go but it fails,
	// so do it foreach configuration np they aint so much...

	if cfg != nil && len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}

// MergeSingle merges the default with the given config and returns the result
func (c Iris) MergeSingle(cfg Iris) (config Iris) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
