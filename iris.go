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

// Package iris v2.0.0

// Note: When 'Station', we mean the Iris type.
package iris

import (
	"os"

	"sync"

	"github.com/kataras/iris/logger"
	"github.com/kataras/iris/render"
	"github.com/kataras/iris/server"
)

type (
	// IrisConfig options for iris before server listen
	// MaxRequestBodySize is the only options that can be changed after server listen - using SetMaxRequestBodySize(int)
	// Render can be changed after declaration but before server's listen - using SetRenderConfig(RenderConfig)
	IrisConfig struct {
		// MaxRequestBodySize Maximum request body size.
		//
		// The server rejects requests with bodies exceeding this limit.
		//
		// By default request body size is unlimited.
		MaxRequestBodySize int
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

		// Render specify configs for rendering
		Render *RenderConfig
	}

	// Iris is the container of all, server, router, cache and the sync.Pool
	Iris struct {
		*router
		Server  *server.Server
		Plugins *PluginContainer
		render  *render.Render
		//we want options exported, Options but Options is an http method also, so we make a big change here
		// and rename the iris.IrisOptions to simple 'iris.IrisConfig' - no iris.Config because of the default func Config()
		Config *IrisConfig
		Logger *logger.Logger
	}
)

// New creates and returns a new iris Iris. If config is empty then default config is used
//
// Receives an optional iris.IrisConfig as parameter
// If empty then iris.DefaultConfig() are used
func New(configs ...*IrisConfig) *Iris {
	config := DefaultConfig()
	if configs != nil && len(configs) > 0 {
		config = configs[0]
	}

	if config.ProfilePath == "" {
		config.ProfilePath = DefaultProfilePath
	}

	// create the Iris
	s := &Iris{Config: config, Plugins: &PluginContainer{}}

	// create & set the router
	s.router = newRouter(s)

	// set the Logger
	s.Logger = logger.New()
	s.Logger.SetEnable(config.Log)

	return s
}

// SetMaxRequestBodySize Maximum request body size.
//
// The server rejects requests with bodies exceeding this limit.
//
// By default request body size is unlimited.
func (s *Iris) SetMaxRequestBodySize(size int) {
	s.Config.MaxRequestBodySize = size
}

// SetRenderConfig sets the Config.Render, can be setted before server's listen, not after.
func (s *Iris) SetRenderConfig(renderCfg *RenderConfig) {
	s.Config.Render = renderCfg
}

// newContextPool returns a new context pool, internal method used in tree and router
func (s *Iris) newContextPool() sync.Pool {
	return sync.Pool{New: func() interface{} {
		return &Context{station: s}
	}}
}

// openServer is internal method, open the server with specific options passed by the Listen and ListenTLS
func (s *Iris) openServer(opt server.Config) (err error) {
	s.router.optimize()

	s.Server = server.New(opt)
	s.Server.SetHandler(s.router.ServeRequest)

	if s.Config.MaxRequestBodySize > 0 {
		s.Server.MaxRequestBodySize = s.Config.MaxRequestBodySize
	}

	s.Plugins.DoPreListen(s)

	if err = s.Server.OpenServer(); err == nil {
		// set the render(er) now
		s.render = newRender(s.Config.Render)

		s.Plugins.DoPostListen(s)
		ch := make(chan os.Signal)
		<-ch
		s.Close()
	}
	return
}

// Listen starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It returns an error you are responsible how to handle this
// ex: log.Fatal(iris.Listen(":8080"))
func (s *Iris) Listen(addr string) error {
	opt := server.Config{ListeningAddr: addr}
	return s.openServer(opt)
}

// ListenTLS Starts a https server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It returns an error you are responsible how to handle this
// ex: log.Fatal(iris.ListenTLS(":8080","yourfile.cert","yourfile.key"))
func (s *Iris) ListenTLS(addr string, certFile, keyFile string) error {
	opt := server.Config{ListeningAddr: addr, CertFile: certFile, KeyFile: keyFile}
	return s.openServer(opt)
}

// Close is used to close the tcp listener from the server
func (s *Iris) Close() error {
	s.Plugins.DoPreClose(s)
	return s.Server.CloseServer()
}
