package config

import (
	"github.com/imdario/mergo"
)

// Default values for base Iris conf
const (
	DefaultDisablePathCorrection = false
	DefaultDisablePathEscape     = false
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

		// MaxRequestBodySize Maximum request body size.
		//
		// The server rejects requests with bodies exceeding this limit.
		//
		// By default request body size is -1, unlimited.
		MaxRequestBodySize int64

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

		// Logger the configuration for the logger
		// Iris logs ONLY SEMANTIC errors and the banner if enabled
		Logger Logger

		// Sessions contains the configs for sessions
		Sessions Sessions

		// Render contains the configs for template and rest configuration
		Render Render

		// Websocket contains the configs for Websocket's server integration
		Websocket *Websocket

		// Mail contains the configs for the mail sender service
		Mail Mail

		// OAuth the configs for the gothic oauth/oauth2 authentication for third-party websites
		// See https://github.com/iris-contrib/gothic/blob/master/example/main.go
		OAuth OAuth

		// Server contains the configs for the http server
		// Server configs are the only one which are setted inside base Iris package (from Listen, ListenTLS, ListenUNIX) NO from users
		//
		// this field is useful only when you need to READ which is the server's address, certfile & keyfile or unix's mode.
		//
		Server Server
	}

	// Render struct keeps organise all configuration about rendering, templates and rest currently.
	Render struct {
		// Template the configs for template
		Template Template
		// Rest configs for rendering.
		//
		// these options inside this config don't have any relation with the TemplateEngine
		// from github.com/kataras/iris/rest
		Rest Rest
	}
)

// DefaultRender returns default configuration for templates and rest rendering
func DefaultRender() Render {
	return Render{
		// set the default template config both not nil and default Engine to Standar
		Template: DefaultTemplate(),
		// set the default configs for rest
		Rest: DefaultRest(),
	}
}

// Default returns the default configuration for the Iris staton
func Default() Iris {
	return Iris{
		DisablePathCorrection: DefaultDisablePathCorrection,
		DisablePathEscape:     DefaultDisablePathEscape,
		DisableBanner:         false,
		MaxRequestBodySize:    -1,
		ProfilePath:           "",
		Logger:                DefaultLogger(),
		Sessions:              DefaultSessions(),
		Render:                DefaultRender(),
		Websocket:             DefaultWebsocket(),
		Mail:                  DefaultMail(),
		OAuth:                 DefaultOAuth(),
		Server:                DefaultServer(),
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

/* maybe some day
// FromFile returns the configuration for Iris station
//
// receives one parameter
// pathIni(string) the file path of the configuration-ini style
//
// returns an error if something bad happens
func FromFile(pathIni string) (c Iris, err error) {
	c = Iris{}
	err = ini.MapTo(&c, pathIni)

	return
}
*/
