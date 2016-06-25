package iris

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kataras/iris/config"
	"github.com/kataras/iris/logger"
	"github.com/kataras/iris/websocket"

	"github.com/kataras/iris/render/rest"
	"github.com/kataras/iris/render/template"
	"github.com/kataras/iris/sessions"
	///NOTE: register the session providers, but the s.Config.Sessions.Provider will be used only, if this empty then sessions are disabled.
	_ "github.com/kataras/iris/sessions/providers/memory"
	_ "github.com/kataras/iris/sessions/providers/redis"
)

// Default entry, use it with iris.$anyPublicFunc
var (
	Default    *Framework
	Config     *config.Iris
	Logger     *logger.Logger
	Plugins    PluginContainer
	Websocket  websocket.Server
	HTTPServer *Server
)

func init() {
	Default = New()
	Config = Default.Config
	Logger = Default.Logger
	Plugins = Default.Plugins
	Websocket = Default.Websocket
	HTTPServer = Default.HTTPServer
}

const (
	/* conversional */

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
	// NoLayout to disable layout for a particular template file
	// conversion for config.NoLayout
	NoLayout = config.NoLayout

	/* end conversional */
)

// Framework is our God |\| Google.Search('Greek mythology Iris')
//
// Implements the FrameworkAPI
type Framework struct {
	*muxAPI
	rest      *rest.Render
	templates *template.Template
	sessions  *sessions.Manager
	// fields which are useful to the user/dev
	HTTPServer *Server
	Config     *config.Iris
	Logger     *logger.Logger
	Plugins    PluginContainer
	Websocket  websocket.Server
}

// New creates and returns a new Iris station aka Framework.
//
// Receives an optional config.Iris as parameter
// If empty then config.Default() is used instead
func New(cfg ...config.Iris) *Framework {
	c := config.Default().Merge(cfg)

	// we always use 's' no 'f' because 's' is easier for me to remember because of 'station'
	// some things never change :)
	s := &Framework{Config: &c}
	{
		///NOTE: set all with s.Config pointer
		// set the Logger
		s.Logger = logger.New(s.Config.Logger)
		// set the plugin container
		s.Plugins = &pluginContainer{logger: s.Logger}
		// set the websocket server
		s.Websocket = websocket.NewServer(s.Config.Websocket)
		// set the servemux, which will provide us the public API also, with its context pool
		mux := newServeMux(sync.Pool{New: func() interface{} { return &Context{framework: s} }}, s.Logger)
		// set the public router API (and party)
		s.muxAPI = &muxAPI{mux: mux, relativePath: "/"}
		// set the server
		s.HTTPServer = newServer(&s.Config.Server)
	}

	return s
}

func (s *Framework) initialize() {
	// set sessions
	if s.Config.Sessions.Provider != "" {
		s.sessions = sessions.New(s.Config.Sessions)
	}

	// set the rest
	s.rest = rest.New(s.Config.Render.Rest)

	// set templates if not already setted
	s.prepareTemplates()

	// listen to websocket connections
	websocket.RegisterServer(s, s.Websocket, s.Logger)

	//  prepare the mux & the server
	s.mux.setCorrectPath(!s.Config.DisablePathCorrection)
	s.mux.setEscapePath(!s.Config.DisablePathEscape)
	s.mux.setHostname(s.HTTPServer.VirtualHostname())
	// set the debug profiling handlers if ProfilePath is setted
	if debugPath := s.Config.ProfilePath; debugPath != "" {
		s.Handle(MethodGet, debugPath+"/*action", profileMiddleware(debugPath)...)
	}

	if s.Config.MaxRequestBodySize > 0 {
		s.HTTPServer.MaxRequestBodySize = int(s.Config.MaxRequestBodySize)
	}
}

// prepareTemplates sets the templates if not nil, we make this check  because of .TemplateString, which can be called before Listen
func (s *Framework) prepareTemplates() {
	// prepare the templates
	if s.templates == nil {
		// These functions are directly contact with Iris' functionality.
		funcs := map[string]interface{}{
			"url":     s.URL,
			"urlpath": s.Path,
		}

		template.RegisterSharedFuncs(funcs)

		s.templates = template.New(s.Config.Render.Template)
	}
}

// openServer is internal method, open the server with specific options passed by the Listen and ListenTLS
// it's a blocking func
func (s *Framework) openServer() (err error) {
	s.initialize()
	s.Plugins.DoPreListen(s)
	// set the server's handler now, in order to give the chance to the plugins to add their own middlewares and routes to this station
	s.HTTPServer.SetHandler(s.mux)
	if err = s.HTTPServer.Open(); err == nil {
		// print the banner
		if !s.Config.DisableBanner {
			s.Logger.PrintBanner(banner,
				fmt.Sprintf("%s: Running at %s\n", time.Now().Format(config.TimeFormat),
					s.HTTPServer.Host()))
		}
		s.Plugins.DoPostListen(s)
		ch := make(chan os.Signal)
		<-ch
		s.Close()
	}
	return
}

// closeServer is used to close the tcp listener from the server, returns an error
func (s *Framework) closeServer() error {
	s.Plugins.DoPreClose(s)
	return s.HTTPServer.close()
}

// justServe initializes the whole framework but server doesn't listens to a specific net.Listener
func (s *Framework) justServe() *Server {
	s.initialize()
	s.Plugins.DoPreListen(s)
	s.HTTPServer.SetHandler(s.mux)
	s.Plugins.DoPostListen(s)
	return s.HTTPServer
}
