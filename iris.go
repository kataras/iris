// Package iris v3.0.0-beta.1
//
// Note: When 'Station', we mean the Iris type.
package iris

import (
	"os"

	"sync"

	"strconv"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/logger"
	"github.com/kataras/iris/mail"
	"github.com/kataras/iris/render/rest"
	"github.com/kataras/iris/render/template"
	"github.com/kataras/iris/server"
	"github.com/kataras/iris/sessions"
	//  memory loads the memory session provider
	_ "github.com/kataras/iris/sessions/providers/memory"
	// _ redis loads the redis session provider
	_ "github.com/kataras/iris/sessions/providers/redis"
	"github.com/kataras/iris/utils"
	"github.com/kataras/iris/websocket"
	"github.com/klauspost/compress/gzip"
)

const (
	// Version of the iris
	Version = "v3.0.0-beta.1"
	banner  = `                        _____      _
                       |_   _|    (_)
                         | |  ____ _  ___
                         | | | __|| |/ __|
                        _| |_| |  | |\__ \
                       |_____|_|  |_||___/ ` + Version + `
                                                 				 `
)

/* for conversion */

var (
	// HTMLEngine conversion for config.HTMLEngine
	HTMLEngine = config.HTMLEngine
	// PongoEngine conversion for config.PongoEngine
	PongoEngine = config.PongoEngine
	// MarkdownEngine conversion for config.MarkdownEngine
	MarkdownEngine = config.MarkdownEngine
	// JadeEngine conversion for config.JadeEngine
	JadeEngine = config.JadeEngine
	// AmberEngine conversion for config.AmberEngine
	AmberEngine = config.AmberEngine

	// DefaultEngine conversion for config.DefaultEngine
	DefaultEngine = config.DefaultEngine
	// NoEngine conversion for config.NoEngine
	NoEngine = config.NoEngine
	//

	NoLayout = config.NoLayout
)

/* */

var stationsRunning = 0

type (

	// Iris is the container of all, server, router, cache and the sync.Pool
	Iris struct {
		*router
		config          *config.Iris
		server          *server.Server
		plugins         *PluginContainer
		rest            *rest.Render
		templates       *template.Template
		sessionManager  *sessions.Manager
		websocketServer websocket.Server
		mailService     mail.Service
		logger          *logger.Logger
		gzipWriterPool  sync.Pool // this pool is used everywhere needed in the iris for example inside party-> StaticSimple
	}
)

// New creates and returns a new iris station.
//
// Receives an optional config.Iris as parameter
// If empty then config.Default() is used instead
func New(cfg ...config.Iris) *Iris {

	c := config.Default().Merge(cfg)

	// create the Iris
	s := &Iris{config: &c, plugins: &PluginContainer{}}
	// create & set the router
	s.router = newRouter(s)

	// set the Logger
	s.logger = logger.New()

	// set the gzip writer pool
	s.gzipWriterPool = sync.Pool{New: func() interface{} { return &gzip.Writer{} }}
	return s
}

// newContextPool returns a new context pool, internal method used in tree and router
func (s *Iris) newContextPool() sync.Pool {
	return sync.Pool{New: func() interface{} {
		return &Context{station: s}
	}}
}

func (s *Iris) initTemplates() {
	if s.templates == nil { // because if .Templates() called before server's listen, s.templates != nil when PreListen
		//  init the templates

		// set the custom iris-direct-integration functions, layout and no-layout  if  HTMLEngine is used
		if s.config.Render.Template.Engine == config.HTMLEngine {
			funcs := map[string]interface{}{
				"url": func(routeName string, args ...interface{}) (string, error) {
					r := s.RouteByName(routeName)
					// check if not found
					if r.GetPath() == "" {
						return "", ErrRenderRouteNotFound.Format(routeName)
					}

					if result, ok := r.Parse(args...); ok {
						return result, nil
					}
					return "", nil
				},
			}
			// these should be already a non-nil map but if .New(cfg) it's not, is mergo's bug, temporary:
			if s.config.Render.Template.HTMLTemplate.LayoutFuncs == nil {
				s.config.Render.Template.HTMLTemplate.LayoutFuncs = make(map[string]interface{}, 1)
			}

			if s.config.Render.Template.HTMLTemplate.Funcs == nil {
				s.config.Render.Template.HTMLTemplate.Funcs = make(map[string]interface{}, 1)
			}
			//

			for k, v := range funcs {
				// we don't want to override the user's LayoutFuncs, user should be able to override anything.
				if s.config.Render.Template.HTMLTemplate.LayoutFuncs[k] == nil {
					s.config.Render.Template.HTMLTemplate.LayoutFuncs[k] = v
				}

				if s.config.Render.Template.HTMLTemplate.Funcs[k] == nil {
					s.config.Render.Template.HTMLTemplate.Funcs[k] = v
				}

			}

		}

		s.templates = template.New(s.config.Render.Template)

	}

}

func (s *Iris) initWebsocketServer() {
	if s.websocketServer == nil {
		// enable websocket if config.Websocket.Endpoint != ""
		if s.config.Websocket.Endpoint != "" {
			s.websocketServer = websocket.New(s, s.config.Websocket)
		}
	}
}

func (s *Iris) initMailService() {
	if s.mailService == nil {
		// enable mail sender  service if configs are valid
		if s.config.Mail.Host != "" && s.config.Mail.Username != "" && s.config.Mail.Password != "" {
			s.mailService = mail.New(s.config.Mail)
		}
	}
}

func (s *Iris) printBanner() {
	c := color.New(color.FgHiBlue).Add(color.Bold)
	printTicker := utils.NewTicker()
	// for ANY case, we don't want to panic on print banner if anything goes bad
	defer func() {
		if r := recover(); r != nil {
			printTicker.Stop()
		}
	}()

	var i uint64 = 0

	printTicker.OnTick(func() {
		if len(banner) <= int(atomic.LoadUint64(&i)) {
			atomic.StoreUint64(&i, 0)
			printTicker.Stop()

			c.Add(color.FgGreen)
			stationsRunning++
			c.Println()

			if s.server != nil && s.server.IsListening() {
				if stationsRunning > 1 {
					c.Println("Server[" + strconv.Itoa(stationsRunning) + "]")

				}
				c.Printf("%s: Running at %s\n", time.Now().Format(config.TimeFormat), s.server.Config.ListeningAddr)

			}
			c.DisableColor()
			return
		}
		c.Printf("%c", banner[i])
		atomic.AddUint64(&i, 1)

	})

	printTicker.Start(time.Duration(433) * time.Nanosecond)

}

// PreListen call router's optimize, sets the server's handler and notice the plugins
// capital because we need it sometimes, for example inside the graceful
// receives the config.Server
// returns the station's Server (*server.Server)
// it's a non-blocking func
func (s *Iris) PreListen(opt config.Server) *server.Server {
	// run the printBanner with nice animation until PreListen and PostListen finish
	if !s.config.DisableBanner {
		go s.printBanner()
	}

	// set the logger's state
	s.logger.SetEnable(!s.config.DisableLog)
	// router preparation, runs only once even if called more than one time.
	if !s.router.optimized {
		s.router.optimize()

		s.server = server.New(opt)
		s.server.SetHandler(s.router.ServeRequest)

		if s.config.MaxRequestBodySize > 0 {
			s.server.MaxRequestBodySize = int(s.config.MaxRequestBodySize)
		}
	}

	s.plugins.DoPreListen(s)

	return s.server
}

// PostListen sets the rest render, template engine, sessions and notice the plugins
// capital because we need it sometimes, for example inside the graceful
// it's a non-blocking func
func (s *Iris) PostListen() {
	//if not error opening the server, then:

	// prepare the route actions, these actions needs real server's access because that it's after server's listen
	s.router.optimizeLookups()

	//set the  rest (for Data, Text, JSON, JSONP, XML)
	s.rest = rest.New(s.config.Render.Rest)
	// set the templates
	s.initTemplates()
	// set the session manager if we have a provider
	if s.config.Sessions.Provider != "" {
		s.sessionManager = sessions.New(s.config.Sessions)
	}

	// set the websocket
	s.initWebsocketServer()

	s.plugins.DoPostListen(s)
}

// listen is internal method, open the server with specific options passed by the Listen and ListenTLS
// it's a blocking func
func (s *Iris) listen(opt config.Server) (err error) {
	s.PreListen(opt)

	if err = s.server.OpenServer(); err == nil {
		s.PostListen()

		ch := make(chan os.Signal)
		<-ch
		s.Close()
	}
	return
}

// ListenWithErr starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It returns an error you are responsible how to handle this
// if you need a func to panic on error use the Listen
// ex: log.Fatal(iris.ListenWithErr(":8080"))
func (s *Iris) ListenWithErr(addr string) error {
	opt := config.Server{ListeningAddr: addr}
	return s.listen(opt)
}

// Listen starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It panics on error if you need a func to return an error use the ListenWithErr
// ex: iris.Listen(":8080")
func (s *Iris) Listen(addr string) {
	if err := s.ListenWithErr(addr); err != nil {
		panic(err)
	}
}

// ListenTLSWithErr Starts a https server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It returns an error you are responsible how to handle this
// if you need a func to panic on error use the ListenTLS
// ex: log.Fatal(iris.ListenTLSWithErr(":8080","yourfile.cert","yourfile.key"))
func (s *Iris) ListenTLSWithErr(addr string, certFile, keyFile string) error {
	opt := config.Server{ListeningAddr: addr, CertFile: certFile, KeyFile: keyFile}
	return s.listen(opt)
}

// ListenTLS Starts a https server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port or just port
//
// It panics on error if you need a func to return an error use the ListenTLSWithErr
// ex: iris.ListenTLS(":8080","yourfile.cert","yourfile.key")
func (s *Iris) ListenTLS(addr string, certFile, keyFile string) {
	if err := s.ListenTLSWithErr(addr, certFile, keyFile); err != nil {
		panic(err)
	}
}

// CloseWithErr is used to close the tcp listener from the server, returns an error
func (s *Iris) CloseWithErr() error {
	s.plugins.DoPreClose(s)
	return s.server.CloseServer()
}

//Close terminates the server and panic if error occurs
func (s *Iris) Close() {
	if err := s.CloseWithErr(); err != nil {
		panic(err)
	}
}

// Server returns the server
func (s *Iris) Server() *server.Server {
	return s.server
}

// Plugins returns the plugin container
func (s *Iris) Plugins() *PluginContainer {
	return s.plugins
}

// Config returns the configs
func (s *Iris) Config() *config.Iris {
	return s.config
}

// Logger returns the logger
func (s *Iris) Logger() *logger.Logger {
	return s.logger
}

// Rest returns the rest render
func (s *Iris) Rest() *rest.Render {
	return s.rest
}

// Templates returns the template render
func (s *Iris) Templates() *template.Template {
	s.initTemplates() // for any case the user called .Templates() before server's listen
	return s.templates
}

// Websocket returns the websocket server
func (s *Iris) Websocket() websocket.Server {
	s.initWebsocketServer() // for any case the user called .Websocket() before server's listen
	return s.websocketServer
}

// Mail returns the mail sender service
func (s *Iris) Mail() mail.Service {
	s.initMailService()
	return s.mailService
}
