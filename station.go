// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package iris

import (
	"net/http/pprof"
	"os"
)

const (
	// DefaultProfilePath is the default path for the web pprof '/debug/pprof'
	DefaultProfilePath = "/debug/pprof"
)

type (
	// IStation is the interface which the Station should implements
	IStation interface {
		IRouter
		Plugin(IPlugin) error
		GetPluginContainer() IPluginContainer
		GetTemplates() *HTMLTemplates
		//yes we need that again if no .Listen called and you use other server, you have to call .Build() before
		OptimusPrime()
		HasOptimized() bool
		Logger() *Logger
		SetMaxRequestBodySize(int)
	}

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

		// PathCorrection corrects and redirects the requested path to the registed path
		// for example, if /home/ path is requested but no handler for this Route found,
		// then the Router checks if /home handler exists, if yes, redirects the client to the correct path /home
		// and VICE - VERSA if /home/ is registed but /home is requested then it redirects to /home/
		//
		// Default is true
		PathCorrection bool

		// Log turn it to false if you want to disable logger,
		// Iris prints/logs ONLY errors, so be careful when you disable it
		Log bool
	}

	// Station is the container of all, server, router, cache and the sync.Pool
	Station struct {
		IRouter
		Server          *Server
		templates       *HTMLTemplates
		options         StationOptions
		pluginContainer *PluginContainer
		//it's true when hosts(domain) and cors middleware has optimized or when Listen occured
		optimized      bool
		optimizedHosts bool
		optimizedCors  bool

		logger *Logger
	}
)

// check at the compile time if Station implements correct the IRouter interface
// which it comes from the *Router or MemoryRouter which again it comes from *Router
var _ IStation = &Station{}

// newStation creates and returns a station, is used only inside main file iris.go
func newStation(options StationOptions) *Station {
	// create the station
	s := &Station{options: options, pluginContainer: &PluginContainer{}}
	// create & set the router
	s.IRouter = NewRouter(s)

	// set the debug profiling handlers if enabled
	if options.Profile {
		debugPath := options.ProfilePath
		s.IRouter.Get(debugPath+"/", ToHandlerFunc(pprof.Index))
		s.IRouter.Get(debugPath+"/cmdline", ToHandlerFunc(pprof.Cmdline))
		s.IRouter.Get(debugPath+"/profile", ToHandlerFunc(pprof.Profile))
		s.IRouter.Get(debugPath+"/symbol", ToHandlerFunc(pprof.Symbol))

		s.IRouter.Get(debugPath+"/goroutine", ToHandlerFunc(pprof.Handler("goroutine")))
		s.IRouter.Get(debugPath+"/heap", ToHandlerFunc(pprof.Handler("heap")))
		s.IRouter.Get(debugPath+"/threadcreate", ToHandlerFunc(pprof.Handler("threadcreate")))
		s.IRouter.Get(debugPath+"/pprof/block", ToHandlerFunc(pprof.Handler("block")))
	}

	//set the logger
	s.logger = NewLogger(LoggerOutTerminal, "", 0)
	s.logger.SetEnable(options.Log)
	return s
}

// Plugin activates the plugins and if succeed then adds it to the activated plugins list
func (s *Station) Plugin(plugin IPlugin) error {
	return s.pluginContainer.Plugin(plugin)
}

// GetPluginContainer returns the pluginContainer
func (s Station) GetPluginContainer() IPluginContainer {
	return s.pluginContainer
}

// GetTemplates returns the *template.Template registed to this station, if any
func (s Station) GetTemplates() *HTMLTemplates {
	return s.templates
}

// Logger returns the station's logger
func (s *Station) Logger() *Logger {
	return s.logger
}

// OptimusPrime make the best last optimizations to make iris the faster framework out there
// This function is called automatically on .Listen, and all Router's Handle functions
func (s *Station) OptimusPrime() {
	if s.optimized {
		return
	}

	if !s.optimizedCors {
		//check if any route has cors setted to true
		routerHasCors := func() (has bool) {
			/*gLen := len(s.IRouter.getGarden())
			for i := 0; i < gLen; i++ {
				if s.IRouter.getGarden()[i].cors {
					return true
				}
			}*/
			s.IRouter.getGarden().visitAll(func(i int, tree *tree) {
				if tree.cors {
					has = true
				}
			})
			return
		}()

		if routerHasCors {
			s.IRouter.setMethodMatch(CorsMethodMatch)
			s.optimizedCors = true
		}

	}
	if !s.optimizedHosts {
		// check if any route has subdomains
		routerHasHosts := func() (has bool) {
			/*gLen := len(s.IRouter.getGarden())
			for i := 0; i < gLen; i++ {
				if s.IRouter.getGarden()[i].hosts {
					return true
				}
			}*/
			s.IRouter.getGarden().visitAll(func(i int, tree *tree) {
				if tree.hosts {
					has = true
				}
			})
			return
		}()

		// For performance only,in order to not check at runtime for hosts and subdomains, I think it's better to do this:
		if routerHasHosts {
			switch s.IRouter.getType() {
			case Normal:
				s.IRouter = NewRouterDomain(s.IRouter.(*Router))
				s.optimizedHosts = true
				break
			}
			// just this no new router
		}

	}

	if s.optimizedCors && s.optimizedHosts {
		s.optimized = true
	}

}

// HasOptimized returns if the station has optimized ( OptimusPrime run once)
func (s *Station) HasOptimized() bool {
	return s.optimized
}

// Listen starts the standalone http server
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Station) Listen(fullHostOrPort ...string) (err error) {
	s.OptimusPrime()
	opt := ServerOptions{ListeningAddr: ParseAddr(fullHostOrPort...)}
	server := NewServer(opt)
	server.SetHandler(s.IRouter.ServeRequest)
	s.Server = server
	s.pluginContainer.DoPreListen(s)
	err = server.Listen()
	if err == nil {
		s.pluginContainer.DoPostListen(s)
		ch := make(chan os.Signal)
		<-ch
		s.Close()
	}

	return
}

// ListenTLS Starts a httpS/http2 server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the fullHostOrPort parameter which as the form of
// host:port or just port
func (s *Station) ListenTLS(fullAddress string, certFile, keyFile string) error {
	s.OptimusPrime()
	opt := ServerOptions{ListeningAddr: ParseAddr(fullAddress), CertFile: certFile, KeyFile: keyFile}
	server := NewServer(opt)
	server.SetHandler(s.IRouter.ServeRequest)
	s.Server = server
	s.pluginContainer.DoPreListen(s)
	err := server.ListenTLS()
	if err == nil {
		s.pluginContainer.DoPostListen(s)
		ch := make(chan os.Signal)
		<-ch
		s.Close()
	}

	return err
}

// Serve is used instead of the iris.Listen
// eg  http.ListenAndServe(":80",iris.Serve()) if you don't want to use iris.Listen(":80")
//func (s *Station) Serve() http.Handler {
//	s.OptimusPrime()
//	s.pluginContainer.DoPreListen(s)
//	return s.IRouter
//}

// Close is used to close the tcp listener from the server
func (s *Station) Close() {
	s.pluginContainer.DoPreClose(s)
	s.Server.CloseServer()

}

// Templates loads HTML templates
// receives one parameter
//
// pathGlob the local directory, it can be a pattern (string)
func (s *Station) Templates(pathGlob string) {
	if s.templates == nil {
		s.templates = NewHTMLTemplates(s.logger)
	}

	s.templates.Load(pathGlob)

}

//SetMaxRequestBodySize sets the maximum request body size.
//
// The server rejects requests with bodies exceeding this limit.
//
// By default request body size is unlimited.
func (s *Station) SetMaxRequestBodySize(size int) {
	s.Server.MaxRequestBodySize = size
}
