package iris

import (
	"html/template"
	"net/http/pprof"
	"os"
	"sync"
	"time"
)

const (
	DefaultProfilePath = "/debug/pprof"
)

type (
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

		// Cache change it to false if you don't want to use the cache mechanism that Iris provides for your routes
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
	}

	// Station is the container of all, server, router, cache and the sync.Pool
	Station struct {
		IRouter
		server        *Server
		htmlTemplates *template.Template
		pool          sync.Pool
		options       StationOptions
	}
)

// check at the compile time if Station implements correct the IRouter interface
// which it comes from the *Router or MemoryRouter which again it comes from *Router
var _ IRouter = &Station{}

// newStation creates and returns a station, is used only inside main file iris.go
func newStation(options StationOptions) *Station {
	// create the station
	s := &Station{options: options}
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
		r.HandleFunc(debugPath+"/", HandlerFunc(pprof.Index), HTTPMethods.GET)
		r.HandleFunc(debugPath+"/cmdline", HandlerFunc(pprof.Cmdline), HTTPMethods.GET)
		r.HandleFunc(debugPath+"/profile", HandlerFunc(pprof.Profile), HTTPMethods.GET)
		r.HandleFunc(debugPath+"/symbol", HandlerFunc(pprof.Symbol), HTTPMethods.GET)

		r.HandleFunc(debugPath+"/goroutine", HandlerFunc(pprof.Handler("goroutine")), HTTPMethods.GET)
		r.HandleFunc(debugPath+"/heap", HandlerFunc(pprof.Handler("heap")), HTTPMethods.GET)
		r.HandleFunc(debugPath+"/threadcreate", HandlerFunc(pprof.Handler("threadcreate")), HTTPMethods.GET)
		r.HandleFunc(debugPath+"/pprof/block", HandlerFunc(pprof.Handler("block")), HTTPMethods.GET)
	}

	// set the router
	s.IRouter = r

	// set the server whith the server handler
	s.server = &Server{handler: s}

	s.pool = sync.Pool{New: func() interface{} {
		return s.makeContext()
	}}

	return s
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Station) Listen(fullHostOrPort interface{}) error {
	return s.server.listen(fullHostOrPort)
}

// ListenTLS Starts a httpS/http2 server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Station) ListenTLS(fullHostOrPort interface{}, certFile, keyFile string) error {
	return s.server.listenTLS(fullHostOrPort, certFile, keyFile)
}

// Close is used to close the tcp listener from the server
func (s *Station) Close() {
	s.server.closeServer()
}

func (s *Station) makeContext() *Context {
	return &Context{Params: make([]PathParameter, 6), Renderer: &Renderer{responseWriter: nil, templates: s.htmlTemplates}}
}

// Templates sets the templates glob path for the web app
func (s *Station) Templates(pathGlob string) {
	var err error
	//s.htmlTemplates = template.Must(template.ParseGlob(pathGlob))
	s.htmlTemplates, err = template.ParseGlob(pathGlob)

	if err != nil {
		//if err then try to load the same path but with the current directory prefix
		// and if not success again then just panic with the first error
		pwd, cerr := os.Getwd()
		if cerr != nil {
			panic(err.Error())

		}
		s.htmlTemplates, cerr = template.ParseGlob(pwd + pathGlob)
		if cerr != nil {
			panic(err.Error())
		}
	}

}
