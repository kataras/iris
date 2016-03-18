package iris

import (
	"html/template"
	"net/http/pprof"
	"os"
	"sync"
	"time"
)

const (
	// DefaultProfilePath is the default path for the web pprof '/debug/pprof'
	DefaultProfilePath = "/debug/pprof"
)

type (
	// StationOptions is the struct which contains all Iris' settings/options
	StationOptions struct {
		// Profile set to true to enable web pprof (debug profiling)
		// Default is false, enabling makes available these 7 routes:
		// /debug/pprof/cmdline
		// /debug/pprof/profile
		// /debug/pprof/symbol
		// /debug/pprof/goroutine
		// /debug/pprof/heap
		// /debug/pprof/threadcreate
		// /debug/pprof/pprof/block
		Profile bool

		// ProfilePath change it if you want other url path than the default
		// Default is /debug/pprof , which means yourhost.com/debug/pprof
		ProfilePath string

		// Cache for Router, change it to false if you don't want to use the cache mechanism that Iris provides for your routes
		Cache bool
		// CacheMaxItems max number of total cached routes, 500 = +~20000 bytes = ~0.019073MB
		// Every time the cache timer reach this number it will reset/clean itself
		// Default is 0
		// If <=0 then cache cleans all of items (bag)
		// Auto cache clean is happening after 5 minutes the last request serve, you can change this number by 'ResetDuration' property
		// Note that MaxItems doesn't means that the items never reach this lengh, only on timer tick this number is checked/consider.
		CacheMaxItems int
		// CacheResetDuration change this time.value to determinate how much duration after last request serving the cache must be reseted/cleaned
		// Default is 5 * time.Minute , Minimum is 30 seconds
		//
		// If CacheMaxItems <= 0 then it clears the whole cache bag at this duration.
		CacheResetDuration time.Duration

		// PathCorrection corrects and redirects the requested path to the registed path
		// for example, if /home/ path is requested but no handler for this Route found,
		// then the Router checks if /home handler exists, if yes, redirects the client to the correct path /home
		// and VISA - VERSA if /home/ is registed but /home is requested then it redirects to /home/
		//
		// Default is true
		PathCorrection bool
	}

	// Station is the container of all, server, router, cache and the sync.Pool
	Station struct {
		IRouter
		server          *Server
		templates       *template.Template
		pool            sync.Pool
		options         StationOptions
		pluginContainer *PluginContainer
	}
)

// check at the compile time if Station implements correct the IRouter interface
// which it comes from the *Router or MemoryRouter which again it comes from *Router
var _ IRouter = &Station{}

// newStation creates and returns a station, is used only inside main file iris.go
func newStation(options StationOptions) *Station {
	// create the station
	s := &Station{options: options, pluginContainer: &PluginContainer{}}
	// create the router
	var r IRouter
	//for now, we can't directly use NewRouter and after NewMemoryRouter, types are not the same.
	if options.Cache {
		r = NewMemoryRouter(NewRouter(s), options.CacheMaxItems, options.CacheResetDuration)
	} else {
		r = NewRouter(s)
	}

	// set the debug profiling handlers if enabled
	if options.Profile {
		debugPath := options.ProfilePath
		r.Get(debugPath+"/", ToHandlerFunc(pprof.Index))
		r.Get(debugPath+"/cmdline", ToHandlerFunc(pprof.Cmdline))
		r.Get(debugPath+"/profile", ToHandlerFunc(pprof.Profile))
		r.Get(debugPath+"/symbol", ToHandlerFunc(pprof.Symbol))

		r.Get(debugPath+"/goroutine", ToHandlerFunc(pprof.Handler("goroutine")))
		r.Get(debugPath+"/heap", ToHandlerFunc(pprof.Handler("heap")))
		r.Get(debugPath+"/threadcreate", ToHandlerFunc(pprof.Handler("threadcreate")))
		r.Get(debugPath+"/pprof/block", ToHandlerFunc(pprof.Handler("block")))
	}

	// set the router
	s.IRouter = r

	// set the server with the server handler
	s.server = &Server{handler: s}

	s.pool = sync.Pool{New: func() interface{} {
		return s.makeContext()
	}}

	return s
}

// Plugin activates the plugins and if succeed then adds it to the activated plugins list
func (s *Station) Plugin(plugin IPlugin) error {
	return s.pluginContainer.Plugin(plugin)
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Station) Listen(fullHostOrPort ...string) error {
	s.pluginContainer.doPreListen(s)
	err := s.server.listen(fullHostOrPort...)
	s.pluginContainer.doPostListen(s, err)

	return err
}

// ListenTLS Starts a httpS/http2 server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Station) ListenTLS(fullAddress string, certFile, keyFile string) error {
	err := s.server.listenTLS(fullAddress, certFile, keyFile)
	s.pluginContainer.doPostListen(s, err)

	return err
}

// Close is used to close the tcp listener from the server
func (s *Station) Close() {
	s.pluginContainer.doPreClose(s)
	s.server.closeServer()
}

func (s *Station) makeContext() *Context {
	return &Context{station: s, Params: make([]PathParameter, 6)}
}

// Templates sets the templates glob path for the web app
func (s *Station) Templates(pathGlob string) {
	var err error
	//s.htmlTemplates = template.Must(template.ParseGlob(pathGlob))
	s.templates, err = template.ParseGlob(pathGlob)

	if err != nil {
		//if err then try to load the same path but with the current directory prefix
		// and if not success again then just panic with the first error
		pwd, cerr := os.Getwd()
		if cerr != nil {
			panic(err.Error())

		}
		s.templates, cerr = template.ParseGlob(pwd + pathGlob)
		if cerr != nil {
			panic(err.Error())
		}
	}

}
