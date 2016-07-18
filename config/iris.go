package config

import (
	"github.com/imdario/mergo"
)

// Default values for base Iris conf
const (
	DefaultDisablePathCorrection = false
	DefaultDisablePathEscape     = false
	DefaultCharset               = "UTF-8"
)

type (
	// Iris configs for the station
	// All fields can be changed before server's listen except the DisablePathCorrection field
	//
	// MaxRequestBodySize is the only options that can be changed after server listen -
	// using Config.MaxRequestBodySize = ...
	// Render's rest config can be changed after declaration but before server's listen -
	// using Config.Render.Rest...
	// Render's Template config can be changed after declaration but before server's listen -
	// using Config.Render.Template...
	// Sessions config can be changed after declaration but before server's listen -
	// using Config.Sessions...
	// and so on...
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

		// Sessions contains the configs for sessions
		Sessions Sessions

		// Websocket contains the configs for Websocket's server integration
		Websocket *Websocket

		// Tester contains the configs for the test framework, so far we have only one because all test framework's configs are setted by the iris itself
		Tester Tester
	}
)

// Default returns the default configuration for the Iris staton
func Default() Iris {
	return Iris{
		DisablePathCorrection:  DefaultDisablePathCorrection,
		DisablePathEscape:      DefaultDisablePathEscape,
		DisableBanner:          false,
		DisableTemplateEngines: false,
		IsDevelopment:          false,
		Charset:                DefaultCharset,
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
