//Package iris the fastest go web framework in (this) Earth.
///NOTE: When you see 'framework' or 'station' we mean the Iris web framework's main implementation.
//
//
// Basic usage
// ----------------------------------------------------------------------
//
// package main
//
// import  "gopkg.in/kataras/iris.v4"
//
// func main() {
//     iris.Get("/hi_json", func(c *iris.Context) {
//         c.JSON(iris.StatusOK, iris.Map{
//             "Name": "Iris",
//             "Released":  "13 March 2016",
//         })
//     })
//     iris.ListenLETSENCRYPT("mydomain.com")
// }
//
// ----------------------------------------------------------------------
//
// package main
//
// import  "gopkg.in/kataras/iris.v4"
//
// func main() {
// 	s1 := iris.New()
// 	s1.Get("/hi_json", func(c *iris.Context) {
// 		c.JSON(200, iris.Map{
// 			"Name": "Iris",
// 			"Released":  "13 March 2016",
// 		})
// 	})
//
// 	s2 := iris.New()
// 	s2.Get("/hi_raw_html", func(c *iris.Context) {
// 		c.HTML(iris.StatusOK, "<b> Iris </b> welcomes <h1>you!</h1>")
// 	})
//
// 	go s1.Listen(":8080")
// 	s2.Listen(":1993")
// }
//
// -----------------------------DOCUMENTATION----------------------------
// ----------------------------_______________---------------------------
// For middleware, template engines, response engines, sessions, websockets, mails, subdomains,
// dynamic subdomains, routes, party of subdomains & routes, ssh and much more
// visit https://www.gitbook.com/book/kataras/iris/details
package iris

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"bytes"
	"github.com/valyala/fasthttp"
	"gopkg.in/geekypanda/httpcache.v0"
	"gopkg.in/kataras/go-errors.v0"
	"gopkg.in/kataras/go-fs.v0"
	"gopkg.in/kataras/go-serializer.v0"
	"gopkg.in/kataras/go-sessions.v0"
	"gopkg.in/kataras/go-template.v0"
	"gopkg.in/kataras/go-template.v0/html"
	"gopkg.in/kataras/iris.v4/utils"
)

const (
	// IsLongTermSupport flag is true when the below version number is a long-term-support version
	IsLongTermSupport = true
	// Version is the current version number of the Iris web framework
	Version = "4"

	banner = `         _____      _
        |_   _|    (_)
          | |  ____ _  ___
          | | | __|| |/ __|
         _| |_| |  | |\__ \
        |_____|_|  |_||___/ ` + Version + ` `
)

// Default iris instance entry and its public fields, use it with iris.$anyPublicFuncOrField
var (
	Default   *Framework
	Config    *Configuration
	Logger    *log.Logger // if you want colors in your console then you should use this https://github.com/iris-contrib/logger instead.
	Plugins   PluginContainer
	Router    fasthttp.RequestHandler
	Websocket *WebsocketServer
	// Available is a channel type of bool, fired to true when the server is opened and all plugins ran
	// never fires false, if the .Close called then the channel is re-allocating.
	// the channel remains open until you close it.
	//
	// look at the http_test.go file for a usage example
	Available chan bool
)

// ResetDefault resets the iris.Default which is the instance which is used on the default iris station for
//  iris.Get(all api functions)
// iris.Config
// iris.Logger
// iris.Plugins
// iris.Router
// iris.Websocket
// iris.Available channel
// useful mostly when you are not using the form of app := iris.New() inside your tests, to make sure that you're using a new iris instance
func ResetDefault() {
	Default = New()
	Config = Default.Config
	Logger = Default.Logger
	Plugins = Default.Plugins
	Router = Default.Router
	Websocket = Default.Websocket
	Available = Default.Available
}

func init() {
	ResetDefault()
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------Framework implementation-----------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// FrameworkAPI contains the main Iris Public API
	FrameworkAPI interface {
		MuxAPI
		Set(...OptionSetter)
		Must(error)
		Build()
		Serve(net.Listener) error
		Listen(string)
		ListenTLS(string, string, string)
		ListenLETSENCRYPT(string, ...string)
		ListenUNIX(string, os.FileMode)
		Close() error
		Reserve() error
		AcquireCtx(*fasthttp.RequestCtx) *Context
		ReleaseCtx(*Context)
		CheckForUpdates(bool)
		UseSessionDB(sessions.Database)
		UseSerializer(string, serializer.Serializer)
		UseTemplate(template.Engine) *template.Loader
		UsePreRender(PreRender)
		UseGlobal(...Handler)
		UseGlobalFunc(...HandlerFunc)
		Lookup(string) Route
		Lookups() []Route
		Path(string, ...interface{}) string
		URL(string, ...interface{}) string
		TemplateString(string, interface{}, ...map[string]interface{}) string
		TemplateSourceString(string, interface{}) string
		SerializeToString(string, interface{}, ...map[string]interface{}) string
		Cache(HandlerFunc, time.Duration) HandlerFunc
		InvalidateCache(*Context)
	}

	// Framework is our God |\| Google.Search('Greek mythology Iris')
	//
	// Implements the FrameworkAPI
	Framework struct {
		*muxAPI
		// HTTP Server runtime fields is the iris' defined main server, developer can use unlimited number of servers
		// note: they're available after .Build, and .Serve/Listen/ListenTLS/ListenLETSENCRYPT/ListenUNIX
		ln        net.Listener
		fsrv      *fasthttp.Server
		Available chan bool
		//
		// Router field which can change the default iris' mux behavior
		// if you want to get benefit with iris' context make use of:
		// ctx:= iris.AcquireCtx(*fasthttp.RequestCtx) to get the context at the beginning of your handler
		// iris.ReleaseCtx(ctx) to release/put the context to the pool, at the very end of your custom handler.
		Router fasthttp.RequestHandler

		contextPool sync.Pool
		once        sync.Once
		Config      *Configuration
		sessions    sessions.Sessions
		serializers serializer.Serializers
		templates   *templateEngines
		Logger      *log.Logger
		Plugins     PluginContainer
		Websocket   *WebsocketServer
	}
)

var _ FrameworkAPI = &Framework{}

// New creates and returns a new Iris instance.
//
// Receives (optional) multi options, use iris.Option and your editor should show you the available options to set
// all options are inside ./configuration.go
// example 1: iris.New(iris.OptionIsDevelopment(true), iris.OptionCharset("UTF-8"), irisOptionSessionsCookie("mycookieid"),iris.OptionWebsocketEndpoint("my_endpoint"))
// example 2: iris.New(iris.Configuration{IsDevelopment:true, Charset: "UTF-8", Sessions: iris.SessionsConfiguration{Cookie:"mycookieid"}, Websocket: iris.WebsocketConfiguration{Endpoint:"/my_endpoint"}})
// both ways are totally valid and equal
func New(setters ...OptionSetter) *Framework {

	s := &Framework{}
	s.Set(setters...)

	// logger & plugins
	{
		// set the Logger, which it's configuration should be declared before .Listen because the servemux and plugins needs that
		s.Logger = log.New(s.Config.LoggerOut, s.Config.LoggerPreffix, log.LstdFlags)
		s.Plugins = newPluginContainer(s.Logger)
	}

	// rendering
	{
		s.serializers = serializer.Serializers{}
		// set the templates
		s.templates = newTemplateEngines(map[string]interface{}{
			"url":     s.URL,
			"urlpath": s.Path,
		})
	}

	// websocket & sessions
	{
		s.Websocket = NewWebsocketServer() // in order to be able to call $instance.Websocket.OnConnection

		// set the sessions, look .initialize for its GC
		s.sessions = sessions.New(sessions.DisableAutoGC(true))
	}

	// routing
	{
		// set the servemux, which will provide us the public API also, with its context pool
		mux := newServeMux(s.Logger)
		mux.setCorrectPath(!s.Config.DisablePathCorrection) // correctPath is re-setted on .Set and after build*

		mux.onLookup = s.Plugins.DoPreLookup
		s.contextPool.New = func() interface{} {
			return &Context{framework: s}
		}
		// set the public router API (and party)
		s.muxAPI = &muxAPI{mux: mux, relativePath: "/"}
		s.Available = make(chan bool)
	}

	return s
}

// Set sets an option aka configuration field to the default iris instance
func Set(setters ...OptionSetter) {
	Default.Set(setters...)
}

// Set sets an option aka configuration field to this iris instance
func (s *Framework) Set(setters ...OptionSetter) {
	if s.Config == nil {
		defaultConfiguration := DefaultConfiguration()
		s.Config = &defaultConfiguration
	}

	for _, setter := range setters {
		setter.Set(s.Config)
	}

	if s.muxAPI != nil && s.mux != nil { // if called after .New, which it does, correctPath is the only field we need to be updated before .Listen, so:
		s.mux.setCorrectPath(!s.Config.DisablePathCorrection)
	}
}

// Must panics on error, it panics on registed iris' logger
func Must(err error) {
	Default.Must(err)
}

// Must panics on error, it panics on registed iris' logger
func (s *Framework) Must(err error) {
	if err != nil {
		s.Logger.Panic(err.Error())
	}
}

// Build builds the whole framework's parts together
// DO NOT CALL IT MANUALLY IF YOU ARE NOT:
// SERVE IRIS BEHIND AN EXTERNAL CUSTOM fasthttp.Server, CAN BE CALLED ONCE PER IRIS INSTANCE FOR YOUR SAFETY
func Build() {
	Default.Build()
}

// Build builds the whole framework's parts together
// DO NOT CALL IT MANUALLY IF YOU ARE NOT:
// SERVE IRIS BEHIND AN EXTERNAL CUSTOM fasthttp.Server, CAN BE CALLED ONCE PER IRIS INSTANCE FOR YOUR SAFETY
func (s *Framework) Build() {
	s.once.Do(func() {
		// .Build, normally*, auto-called after station's listener setted but before the real Serve, so here set the host, scheme
		// and the mux hostname(*this is here because user may not call .Serve/.Listen functions if listen by a custom server)

		if s.Config.VHost == "" { // if not setted by Listen functions
			if s.ln != nil { // but user called .Serve
				// then take the listener's addr
				s.Config.VHost = s.ln.Addr().String()
			} else {
				// if no .Serve or .Listen called, then the user should set the VHost manually,
				// however set it to a default value here for any case
				s.Config.VHost = DefaultServerAddr
			}
		}
		// if user didn't specified a scheme then get it from the VHost, which is already setted at before statements
		if s.Config.VScheme == "" {
			s.Config.VScheme = ParseScheme(s.Config.VHost)
		}

		s.Plugins.DoPreBuild(s) // once after configuration has been setted. *nothing stops you to change the VHost and VScheme at this point*
		// re-nwe logger's attrs
		s.Logger.SetPrefix(s.Config.LoggerPreffix)
		s.Logger.SetOutput(s.Config.LoggerOut)

		// prepare the serializers, if not any other serializers setted for the default serializer types(json,jsonp,xml,markdown,text,data) then the defaults are setted:
		serializer.RegisterDefaults(s.serializers)

		// prepare the templates if enabled
		if !s.Config.DisableTemplateEngines {

			s.templates.Reload = s.Config.IsDevelopment
			// check and prepare the templates
			if len(s.templates.Entries) == 0 { // no template engines were registered, let's use the default
				s.UseTemplate(html.New())
			}

			if err := s.templates.Load(); err != nil {
				s.Logger.Panic(err) // panic on templates loading before listening if we have an error.
			}
		}

		// init, starts the session manager if the Cookie configuration field is not empty
		if s.Config.Sessions.Cookie != "" {
			// re-set the configuration field for any case
			s.sessions.Set(s.Config.Sessions, sessions.DisableAutoGC(false))
		}

		if s.Config.Websocket.Endpoint != "" {
			// register the websocket server and listen to websocket connections when/if $instance.Websocket.OnConnection called by the dev
			s.Websocket.RegisterTo(s, s.Config.Websocket)
		}

		//  prepare the mux runtime fields again, for any case
		s.mux.setCorrectPath(!s.Config.DisablePathCorrection)
		s.mux.setEscapePath(!s.Config.DisablePathEscape)
		s.mux.setFireMethodNotAllowed(s.Config.FireMethodNotAllowed)

		// prepare the server's handler, we do that check because iris supports
		// custom routers (you can take the routes registed by iris using iris.Lookups function)
		if s.Router == nil {
			// build and get the default mux' handler(*Context)
			serve := s.mux.BuildHandler()
			// build the fasthttp handler to bind it to the servers
			defaultHandler := func(reqCtx *fasthttp.RequestCtx) {
				ctx := s.AcquireCtx(reqCtx)
				serve(ctx)
				s.ReleaseCtx(ctx)
			}

			s.Router = defaultHandler
		}

		// set the mux' hostname (for multi subdomain routing)
		s.mux.hostname = ParseHostname(s.Config.VHost)

		if s.ln != nil { // user called Listen functions or Serve,
			// create the main server
			srvName := "iris"
			if len(DefaultServerName) > 0 {
				srvName += "_" + DefaultServerName
			}
			s.fsrv = &fasthttp.Server{Name: srvName,
				MaxRequestBodySize: s.Config.MaxRequestBodySize,
				ReadBufferSize:     s.Config.ReadBufferSize,
				WriteBufferSize:    s.Config.WriteBufferSize,
				ReadTimeout:        s.Config.ReadTimeout,
				WriteTimeout:       s.Config.WriteTimeout,
				MaxConnsPerIP:      s.Config.MaxConnsPerIP,
				MaxRequestsPerConn: s.Config.MaxRequestsPerConn,
				Handler:            s.Router,
			}
		}

		// updates, to cover the default station's irs.Config.checkForUpdates
		// note: we could use the IsDevelopment configuration field to do that BUT
		// the developer may want to check for updates without, for example, re-build template files (comes from IsDevelopment) on each request
		if s.Config.CheckForUpdatesSync {
			s.CheckForUpdates(false)
		} else if s.Config.CheckForUpdates {
			go s.CheckForUpdates(false)
		}
	})
}

var (
	errServerAlreadyStarted = errors.New("Server is already started and listening")
)

// Serve serves incoming connections from the given listener.
//
// Serve blocks until the given listener returns permanent error.
func Serve(ln net.Listener) error {
	return Default.Serve(ln)
}

// Serve serves incoming connections from the given listener.
//
// Serve blocks until the given listener returns permanent error.
func (s *Framework) Serve(ln net.Listener) error {
	if s.IsRunning() {
		return errServerAlreadyStarted
	}
	// maybe a 'race' here but user should not call .Serve more than one time especially in more than one go routines...
	s.ln = ln

	s.Build()
	s.Plugins.DoPreListen(s)

	// This didn't helped me ,here, but maybe can help you:
	// https://www.oreilly.com/learning/run-strikingly-fast-parallel-file-searches-in-go-with-sync-errgroup?utm_source=golangweekly&utm_medium=email
	// new experimental package: errgroup

	defer func() {
		if err := recover(); err != nil {
			s.Logger.Panic(err)
		}
	}()

	// start the server in goroutine, .Available will block instead
	go func() { s.Must(s.fsrv.Serve(ln)) }()

	if !s.Config.DisableBanner {
		bannerMessage := fmt.Sprintf("%s: Running at %s", time.Now().Format(s.Config.TimeFormat), s.Config.VHost)
		// we don't print it via Logger because:
		// 1. The banner is only 'useful' when the developer logs to terminal and no file
		// 2. Prefix & LstdFlags options of the default s.Logger

		fmt.Printf("%s\n\n%s\n", banner, bannerMessage)
	}

	s.Plugins.DoPostListen(s)

	go func() { s.Available <- true }()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	if err := s.Close(); err != nil {
		if s.Config.IsDevelopment {
			s.Logger.Printf("Error while closing the server: %s\n", err)
		}
		return err
	}
	os.Exit(1)
	return nil
}

// Listen starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port
//
// It panics on error if you need a func to return an error, use the Serve
func Listen(addr string) {
	Default.Listen(addr)
}

// Listen starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port
//
// It panics on error if you need a func to return an error, use the Serve
func (s *Framework) Listen(addr string) {
	addr = ParseHost(addr)
	if s.Config.VHost == "" {
		s.Config.VHost = addr
		// this will be set as the front-end listening addr
	}

	ln, err := TCP4(addr)
	if err != nil {
		s.Logger.Panic(err)
	}

	s.Must(s.Serve(ln))
}

// ListenTLS Starts a https server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port
//
// It panics on error if you need a func to return an error, use the Serve
// ex: iris.ListenTLS(":8080","yourfile.cert","yourfile.key")
func ListenTLS(addr string, certFile string, keyFile string) {
	Default.ListenTLS(addr, certFile, keyFile)
}

// ListenTLS Starts a https server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port
//
// It panics on error if you need a func to return an error, use the Serve
// ex: iris.ListenTLS(":8080","yourfile.cert","yourfile.key")
func (s *Framework) ListenTLS(addr string, certFile, keyFile string) {
	addr = ParseHost(addr)
	if s.Config.VHost == "" {
		s.Config.VHost = addr
		// this will be set as the front-end listening addr
	}

	ln, err := TLS(addr, certFile, keyFile)
	if err != nil {
		s.Logger.Panic(err)
	}

	s.Must(s.Serve(ln))
}

// ListenLETSENCRYPT starts a server listening at the specific nat address
// using key & certification taken from the letsencrypt.org 's servers
// it's also starts a second 'http' server to redirect all 'http://$ADDR_HOSTNAME:80' to the' https://$ADDR'
// it creates a cache file to store the certifications, for performance reasons, this file by-default is "./letsencrypt.cache"
// if you skip the second parameter then the cache file is "./letsencrypt.cache"
// if you want to disable cache then simple pass as second argument an empty empty string ""
//
// example: https://github.com/iris-contrib/examples/blob/master/letsencyrpt/main.go
//
// supports localhost domains for testing,
// NOTE: if you are ready for production then use `$app.Serve(iris.LETSENCRYPTPROD("mydomain.com"))` instead
func ListenLETSENCRYPT(addr string, cacheFileOptional ...string) {
	Default.ListenLETSENCRYPT(addr, cacheFileOptional...)
}

// ListenLETSENCRYPT starts a server listening at the specific nat address
// using key & certification taken from the letsencrypt.org 's servers
// it's also starts a second 'http' server to redirect all 'http://$ADDR_HOSTNAME:80' to the' https://$ADDR'
// it creates a cache file to store the certifications, for performance reasons, this file by-default is "./letsencrypt.cache"
// if you skip the second parameter then the cache file is "./letsencrypt.cache"
// if you want to disable cache then simple pass as second argument an empty empty string ""
//
// example: https://github.com/iris-contrib/examples/blob/master/letsencyrpt/main.go
//
// supports localhost domains for testing,
// NOTE: if you are ready for production then use `$app.Serve(iris.LETSENCRYPTPROD("mydomain.com"))` instead
func (s *Framework) ListenLETSENCRYPT(addr string, cacheFileOptional ...string) {
	addr = ParseHost(addr)
	if s.Config.VHost == "" {
		s.Config.VHost = addr
		// this will be set as the front-end listening addr
	}
	ln, err := LETSENCRYPT(addr, cacheFileOptional...)
	if err != nil {
		s.Logger.Panic(err)
	}

	// starts a second server which listening on :80 to redirect all requests to the :443 (https://)
	Proxy(":80", "https://"+addr)
	s.Must(s.Serve(ln))
}

// ListenUNIX starts the process of listening to the new requests using a 'socket file', this works only on unix
//
// It panics on error if you need a func to return an error, use the Serve
// ex: iris.ListenUNIX(":8080", Mode: os.FileMode)
func ListenUNIX(addr string, mode os.FileMode) {
	Default.ListenUNIX(addr, mode)
}

// ListenUNIX starts the process of listening to the new requests using a 'socket file', this works only on unix
//
// It panics on error if you need a func to return an error, use the Serve
// ex: iris.ListenUNIX(":8080", Mode: os.FileMode)
func (s *Framework) ListenUNIX(addr string, mode os.FileMode) {
	// *on unix listen we don't parse the host, because sometimes it causes problems to the user
	if s.Config.VHost == "" {
		s.Config.VHost = addr
		// this will be set as the front-end listening addr
	}
	ln, err := UNIX(addr, mode)
	if err != nil {
		s.Logger.Panic(err)
	}

	s.Must(s.Serve(ln))
}

// IsRunning returns true if server is running
func IsRunning() bool {
	return Default.IsRunning()
}

// IsRunning returns true if server is running
func (s *Framework) IsRunning() bool {
	return s != nil && s.ln != nil && s.ln.Addr() != nil && s.ln.Addr().String() != ""
}

// DisableKeepalive whether to disable keep-alive connections.
//
// The server will close all the incoming connections after sending
// the first response to client if this option is set to true.
//
// By default keep-alive connections are enabled
//
// Note: Used on packages like graceful, after the server runs.
func DisableKeepalive(val bool) {
	Default.DisableKeepalive(val)
}

// DisableKeepalive whether to disable keep-alive connections.
//
// The server will close all the incoming connections after sending
// the first response to client if this option is set to true.
//
// By default keep-alive connections are enabled
//
// Note: Used on packages like graceful, after the server runs.
func (s *Framework) DisableKeepalive(val bool) {
	if s.fsrv != nil {
		s.fsrv.DisableKeepalive = val
	}
}

// Close terminates all the registered servers and returns an error if any
// if you want to panic on this error use the iris.Must(iris.Close())
func Close() error {
	return Default.Close()
}

// Close terminates all the registered servers and returns an error if any
// if you want to panic on this error use the iris.Must(iris.Close())
func (s *Framework) Close() error {
	if s.IsRunning() {
		s.Plugins.DoPreClose(s)
		s.Available = make(chan bool)

		return s.ln.Close()
	}

	return nil
}

// Reserve re-starts the server using the last .Serve's listener
func Reserve() error {
	return Default.Reserve()
}

// Reserve re-starts the server using the last .Serve's listener
func (s *Framework) Reserve() error {
	return s.Serve(s.ln)
}

// AcquireCtx gets an Iris' Context from pool
// see .ReleaseCtx & .Serve
func AcquireCtx(reqCtx *fasthttp.RequestCtx) *Context {
	return Default.AcquireCtx(reqCtx)
}

// AcquireCtx gets an Iris' Context from pool
// see .ReleaseCtx & .Serve
func (s *Framework) AcquireCtx(reqCtx *fasthttp.RequestCtx) *Context {
	ctx := s.contextPool.Get().(*Context) // Changed to use the pool's New 09/07/2016, ~ -4k nanoseconds(9 bench tests) per requests (better performance)
	ctx.RequestCtx = reqCtx
	return ctx
}

// ReleaseCtx puts the Iris' Context back to the pool in order to be re-used
// see .AcquireCtx & .Serve
func ReleaseCtx(ctx *Context) {
	Default.ReleaseCtx(ctx)
}

// ReleaseCtx puts the Iris' Context back to the pool in order to be re-used
// see .AcquireCtx & .Serve
func (s *Framework) ReleaseCtx(ctx *Context) {
	ctx.Middleware = nil
	ctx.session = nil
	s.contextPool.Put(ctx)
}

// global once because is not necessary to check for updates on more than one iris station*
var updateOnce sync.Once

const (
	githubOwner = "kataras"
	githubRepo  = "iris"
)

// CheckForUpdates will try to search for newer version of Iris based on the https://github.com/kataras/iris/releases
// If a newer version found then the app will ask the he dev/user if want to update the 'x' version
// if 'y' is pressed then the updater will try to install the latest version
// the updater, will notify the dev/user that the update is finished and should restart the App manually.
func CheckForUpdates(force bool) {
	Default.CheckForUpdates(force)
}

// CheckForUpdates will try to search for newer version of Iris based on the https://github.com/kataras/iris/releases
// If a newer version found then the app will ask the he dev/user if want to update the 'x' version
// if 'y' is pressed then the updater will try to install the latest version
// the updater, will notify the dev/user that the update is finished and should restart the App manually.
// Note: exported func CheckForUpdates exists because of the reason that an update can be executed while Iris is running
func (s *Framework) CheckForUpdates(force bool) {
	updated := false
	checker := func() {
		writer := s.Config.LoggerOut

		if writer == nil {
			writer = os.Stdout // we need a writer because the update process will not be silent.
		}

		fs.DefaultUpdaterAlreadyInstalledMessage = "INFO: Running with the latest version(%s)\n"
		updater, err := fs.GetUpdater(githubOwner, githubRepo, Version)

		if err != nil {
			writer.Write([]byte("Update failed: " + err.Error()))
		}

		updated = updater.Run(fs.Stdout(writer), fs.Stderr(writer), fs.Silent(false))
	}

	if force {
		checker()
	} else {
		updateOnce.Do(checker)
	}

	if updated { // if updated, then do not run the web server
		if s.Logger != nil {
			s.Logger.Println("exiting now...")
		}
		os.Exit(1)
	}

}

// UseSessionDB registers a session database, you can register more than one
// accepts a session database which implements a Load(sid string) map[string]interface{} and an Update(sid string, newValues map[string]interface{})
// the only reason that a session database will be useful for you is when you want to keep the session's values/data after the app restart
// a session database doesn't have write access to the session, it doesn't accept the context, so forget 'cookie database' for sessions, I will never allow that, for your protection.
//
// Note: Don't worry if no session database is registered, your context.Session will continue to work.
func UseSessionDB(db sessions.Database) {
	Default.UseSessionDB(db)
}

// UseSessionDB registers a session database, you can register more than one
// accepts a session database which implements a Load(sid string) map[string]interface{} and an Update(sid string, newValues map[string]interface{})
// the only reason that a session database will be useful for you is when you want to keep the session's values/data after the app restart
// a session database doesn't have write access to the session, it doesn't accept the context, so forget 'cookie database' for sessions, I will never allow that, for your protection.
//
// Note: Don't worry if no session database is registered, your context.Session will continue to work.
func (s *Framework) UseSessionDB(db sessions.Database) {
	s.sessions.UseDatabase(db)
}

// UseSerializer accepts a Serializer and the key or content type on which the developer wants to register this serializer
// the gzip and charset are automatically supported by Iris, by passing the iris.RenderOptions{} map on the context.Render
// context.Render renders this response or a template engine if no response engine with the 'key' found
// with these engines you can inject the context.JSON,Text,Data,JSONP,XML also
// to do that just register with UseSerializer(mySerializer,"application/json") and so on
// look at the https://github.com/kataras/go-serializer for examples
//
// if more than one serializer with the same key/content type exists then the results will be appended to the final request's body
// this allows the developer to be able to create 'middleware' responses engines
//
// Note: if you pass an engine which contains a dot('.') as key, then the engine will not be registered.
// you don't have to import and use github.com/iris-contrib/json, jsonp, xml, data, text, markdown
// because iris uses these by default if no other response engine is registered for these content types
func UseSerializer(forContentType string, e serializer.Serializer) {
	Default.UseSerializer(forContentType, e)
}

// UseSerializer accepts a Serializer and the key or content type on which the developer wants to register this serializer
// the gzip and charset are automatically supported by Iris, by passing the iris.RenderOptions{} map on the context.Render
// context.Render renders this response or a template engine if no response engine with the 'key' found
// with these engines you can inject the context.JSON,Text,Data,JSONP,XML also
// to do that just register with UseSerializer(mySerializer,"application/json") and so on
// look at the https://github.com/kataras/go-serializer for examples
//
// if more than one serializer with the same key/content type exists then the results will be appended to the final request's body
// this allows the developer to be able to create 'middleware' responses engines
//
// Note: if you pass an engine which contains a dot('.') as key, then the engine will not be registered.
// you don't have to import and use github.com/iris-contrib/json, jsonp, xml, data, text, markdown
// because iris uses these by default if no other response engine is registered for these content types
func (s *Framework) UseSerializer(forContentType string, e serializer.Serializer) {
	s.serializers.For(forContentType, e)
}

// UsePreRender adds a Template's PreRender
// PreRender is typeof func(*iris.Context, filenameOrSource string, binding interface{}, options ...map[string]interface{}) bool
// PreRenders helps developers to pass middleware between the route Handler and a context.Render call
// all parameter receivers can be changed before passing it to the actual context's Render
// so, you can change the filenameOrSource, the page binding, the options, and even add cookies, session value or a flash message through ctx
// the return value of a PreRender is a boolean, if returns false then the next PreRender will not be executed, keep note
// that the actual context's Render will be called at any case.
func UsePreRender(pre PreRender) {
	Default.UsePreRender(pre)
}

// UsePreRender adds a Template's PreRender
// PreRender is typeof func(*iris.Context, filenameOrSource string, binding interface{}, options ...map[string]interface{}) bool
// PreRenders helps developers to pass middleware between the route Handler and a context.Render call
// all parameter receivers can be changed before passing it to the actual context's Render
// so, you can change the filenameOrSource, the page binding, the options, and even add cookies, session value or a flash message through ctx
// the return value of a PreRender is a boolean, if returns false then the next PreRender will not be executed, keep note
// that the actual context's Render will be called at any case.
func (s *Framework) UsePreRender(pre PreRender) {
	s.templates.usePreRender(pre)
}

// UseTemplate adds a template engine to the iris view system
// it does not build/load them yet
func UseTemplate(e template.Engine) *template.Loader {
	return Default.UseTemplate(e)
}

// UseTemplate adds a template engine to the iris view system
// it does not build/load them yet
func (s *Framework) UseTemplate(e template.Engine) *template.Loader {
	return s.templates.AddEngine(e)
}

// UseGlobal registers Handler middleware  to the beginning, prepends them instead of append
//
// Use it when you want to add a global middleware to all parties, to all routes in  all subdomains
// It can be called after other, (but before .Listen of course)
func UseGlobal(handlers ...Handler) {
	Default.UseGlobal(handlers...)
}

// UseGlobalFunc registers HandlerFunc middleware  to the beginning, prepends them instead of append
//
// Use it when you want to add a global middleware to all parties, to all routes in  all subdomains
// It can be called after other, (but before .Listen of course)
func UseGlobalFunc(handlersFn ...HandlerFunc) {
	Default.UseGlobalFunc(handlersFn...)
}

// UseGlobal registers Handler middleware  to the beginning, prepends them instead of append
//
// Use it when you want to add a global middleware to all parties, to all routes in  all subdomains
// It can be called after other, (but before .Listen of course)
func (s *Framework) UseGlobal(handlers ...Handler) {
	for _, r := range s.mux.lookups {
		r.middleware = append(handlers, r.middleware...)
	}
}

// UseGlobalFunc registers HandlerFunc middleware to the beginning, prepends them instead of append
//
// Use it when you want to add a global middleware to all parties, to all routes in  all subdomains
// It can be called after other, (but before .Listen of course)
func (s *Framework) UseGlobalFunc(handlersFn ...HandlerFunc) {
	s.UseGlobal(convertToHandlers(handlersFn)...)
}

// Lookup returns a registed route by its name
func Lookup(routeName string) Route {
	return Default.Lookup(routeName)
}

// Lookups returns all registed routes
func Lookups() []Route {
	return Default.Lookups()
}

// Lookup returns a registed route by its name
func (s *Framework) Lookup(routeName string) Route {
	r := s.mux.lookup(routeName)
	if nil == r {
		return nil
	}
	return r
}

// Lookups returns all registed routes
func (s *Framework) Lookups() (routes []Route) {
	// silly but...
	for i := range s.mux.lookups {
		routes = append(routes, s.mux.lookups[i])
	}
	return
}

// Path used to check arguments with the route's named parameters and return the correct url
// if parse failed returns empty string
func Path(routeName string, args ...interface{}) string {
	return Default.Path(routeName, args...)
}

// Path used to check arguments with the route's named parameters and return the correct url
// if parse failed returns empty string
func (s *Framework) Path(routeName string, args ...interface{}) string {
	r := s.mux.lookup(routeName)
	if r == nil {
		return ""
	}

	argsLen := len(args)

	// we have named parameters but arguments not given
	if argsLen == 0 && r.formattedParts > 0 {
		return ""
	} else if argsLen == 0 && r.formattedParts == 0 {
		// it's static then just return the path
		return r.path
	}

	// we have arguments but they are much more than the named parameters

	// 1 check if we have /*, if yes then join all arguments to one as path and pass that as parameter
	if argsLen > r.formattedParts {
		if r.path[len(r.path)-1] == matchEverythingByte {
			// we have to convert each argument to a string in this case

			argsString := make([]string, argsLen, argsLen)

			for i, v := range args {
				if s, ok := v.(string); ok {
					argsString[i] = s
				} else if num, ok := v.(int); ok {
					argsString[i] = strconv.Itoa(num)
				} else if b, ok := v.(bool); ok {
					argsString[i] = strconv.FormatBool(b)
				} else if arr, ok := v.([]string); ok {
					if len(arr) > 0 {
						argsString[i] = arr[0]
						argsString = append(argsString, arr[1:]...)
					}
				}
			}

			parameter := strings.Join(argsString, slash)
			result := fmt.Sprintf(r.formattedPath, parameter)
			return result
		}
		// 2 if !1 return false
		return ""
	}

	arguments := args[0:]

	// check for arrays
	for i, v := range arguments {
		if arr, ok := v.([]string); ok {
			if len(arr) > 0 {
				interfaceArr := make([]interface{}, len(arr))
				for j, sv := range arr {
					interfaceArr[j] = sv
				}
				arguments[i] = interfaceArr[0]
				arguments = append(arguments, interfaceArr[1:]...)
			}

		}
	}

	return fmt.Sprintf(r.formattedPath, arguments...)
}

// DecodeURL returns the uri parameter as url (string)
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.QueryUnescape
func DecodeURL(uri string) string {
	if uri == "" {
		return ""
	}
	encodedPath, _ := url.QueryUnescape(uri)
	return encodedPath
}

// DecodeFasthttpURL returns the path decoded as url
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
/* Credits to Manish Singh @kryptodev for URLDecode by post issue share code */
// simple things, if DecodeURL doesn't gives you the results you waited, use this function
// I know it is not the best  way to describe it, but I don't think you will ever need this, it is here for ANY CASE
func DecodeFasthttpURL(path string) string {
	if path == "" {
		return ""
	}
	u := fasthttp.AcquireURI()
	u.SetPath(path)
	encodedPath := u.String()[8:]
	fasthttp.ReleaseURI(u)
	return encodedPath
}

// URL returns the subdomain+ host + Path(...optional named parameters if route is dynamic)
// returns an empty string if parse is failed
func URL(routeName string, args ...interface{}) (url string) {
	return Default.URL(routeName, args...)
}

// URL returns the subdomain+ host + Path(...optional named parameters if route is dynamic)
// returns an empty string if parse is failed
func (s *Framework) URL(routeName string, args ...interface{}) (url string) {
	r := s.mux.lookup(routeName)
	if r == nil {
		return
	}

	scheme := s.Config.VScheme // if s.Config.VScheme was setted, that will be used instead of the real, in order to make easy to run behind nginx
	host := s.Config.VHost     // if s.Config.VHost was setted, that will be used instead of the real, in order to make easy to run behind nginx
	arguments := args[0:]

	// join arrays as arguments
	for i, v := range arguments {
		if arr, ok := v.([]string); ok {
			if len(arr) > 0 {
				interfaceArr := make([]interface{}, len(arr))
				for j, sv := range arr {
					interfaceArr[j] = sv
				}
				arguments[i] = interfaceArr[0]
				arguments = append(arguments, interfaceArr[1:]...)
			}

		}
	}

	// if it's dynamic subdomain then the first argument is the subdomain part
	if r.subdomain == dynamicSubdomainIndicator {
		if len(arguments) == 0 { // it's a wildcard subdomain but not arguments
			return
		}

		if subdomain, ok := arguments[0].(string); ok {
			host = subdomain + "." + host
		} else {
			// it is not array because we join them before. if not pass a string then this is not a subdomain part, return empty uri
			return
		}

		arguments = arguments[1:]
	}

	if parsedPath := s.Path(routeName, arguments...); parsedPath != "" {
		url = scheme + host + parsedPath
	}

	return
}

// TemplateString executes a template from the default template engine and returns its result as string, useful when you want it for sending rich e-mails
// returns empty string on error
func TemplateString(templateFile string, pageContext interface{}, options ...map[string]interface{}) string {
	return Default.TemplateString(templateFile, pageContext, options...)
}

// TemplateString executes a template from the default template engine and returns its result as string, useful when you want it for sending rich e-mails
// returns empty string on error
func (s *Framework) TemplateString(templateFile string, pageContext interface{}, options ...map[string]interface{}) string {
	if s.Config.DisableTemplateEngines {
		return ""
	}

	res, err := s.templates.ExecuteString(templateFile, pageContext, options...)
	if err != nil {
		return ""
	}
	return res
}

// TemplateSourceString executes a template source(raw string contents) from  the first template engines which supports raw parsing returns its result as string,
//  useful when you want it for sending rich e-mails
// returns empty string on error
func TemplateSourceString(src string, pageContext interface{}) string {
	return Default.TemplateSourceString(src, pageContext)
}

// TemplateSourceString executes a template source(raw string contents) from  the first template engines which supports raw parsing returns its result as string,
//  useful when you want it for sending rich e-mails
// returns empty string on error
func (s *Framework) TemplateSourceString(src string, pageContext interface{}) string {
	if s.Config.DisableTemplateEngines {
		return ""
	}
	res, err := s.templates.ExecuteRawString(src, pageContext)
	if err != nil {
		res = ""
	}
	return res
}

// SerializeToString returns the string of a serializer,
// does not render it to the client
// returns empty string on error
func SerializeToString(keyOrContentType string, obj interface{}, options ...map[string]interface{}) string {
	return Default.SerializeToString(keyOrContentType, obj, options...)
}

// SerializeToString returns the string of a serializer,
// does not render it to the client
// returns empty string on error
func (s *Framework) SerializeToString(keyOrContentType string, obj interface{}, options ...map[string]interface{}) string {
	res, err := s.serializers.SerializeToString(keyOrContentType, obj, options...)
	if err != nil {
		if s.Config.IsDevelopment {
			s.Logger.Printf("Error on SerializeToString, Key(content-type): %s. Trace: %s\n", keyOrContentType, err)
		}
		return ""
	}
	return res
}

// Cache is just a wrapper for a route's handler which you want to enable body caching
// Usage: iris.Get("/", iris.Cache(func(ctx *iris.Context){
//    ctx.WriteString("Hello, world!") // or a template or anything else
// }, time.Duration(10*time.Second))) // duration of expiration
// if <=time.Second then it tries to find it though request header's "cache-control" maxage value
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.Cache instead of iris.Cache
func Cache(bodyHandler HandlerFunc, expiration time.Duration) HandlerFunc {
	return Default.Cache(bodyHandler, expiration)
}

// Cache is just a wrapper for a route's handler which you want to enable body caching
// Usage: iris.Get("/", iris.Cache(func(ctx *iris.Context){
//    ctx.WriteString("Hello, world!") // or a template or anything else
// }, time.Duration(10*time.Second))) // duration of expiration
// if <=time.Second then it tries to find it though request header's "cache-control" maxage value
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.Cache instead of iris.Cache
func (s *Framework) Cache(bodyHandler HandlerFunc, expiration time.Duration) HandlerFunc {
	fh := httpcache.Fasthttp.Cache(func(reqCtx *fasthttp.RequestCtx) {
		ctx := s.AcquireCtx(reqCtx)
		bodyHandler.Serve(ctx)
		s.ReleaseCtx(ctx)
	}, expiration)

	return func(ctx *Context) {
		fh(ctx.RequestCtx)
	}
}

// InvalidateCache clears the cache body for a specific context's url path(cache unique key)
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.InvalidateCache instead of iris.InvalidateCache
func InvalidateCache(ctx *Context) {
	Default.InvalidateCache(ctx)
}

// InvalidateCache clears the cache body for a specific context's url path(cache unique key)
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.InvalidateCache instead of iris.InvalidateCache
func (s *Framework) InvalidateCache(ctx *Context) {
	httpcache.Fasthttp.Invalidate(ctx.RequestCtx)
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------MuxAPI implementation------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
type (
	// RouteNameFunc the func returns from the MuxAPi's methods, optionally sets the name of the Route (*route)
	RouteNameFunc func(string)
	// MuxAPI the visible api for the serveMux
	MuxAPI interface {
		Party(string, ...HandlerFunc) MuxAPI
		// middleware serial, appending
		Use(...Handler)
		UseFunc(...HandlerFunc)
		// returns itself, because at the most-cases used like .Layout, at the first-line party's declaration
		Done(...Handler) MuxAPI
		DoneFunc(...HandlerFunc) MuxAPI
		//

		// main handlers
		Handle(string, string, ...Handler) RouteNameFunc
		HandleFunc(string, string, ...HandlerFunc) RouteNameFunc
		API(string, HandlerAPI, ...HandlerFunc)

		// http methods
		Get(string, ...HandlerFunc) RouteNameFunc
		Post(string, ...HandlerFunc) RouteNameFunc
		Put(string, ...HandlerFunc) RouteNameFunc
		Delete(string, ...HandlerFunc) RouteNameFunc
		Connect(string, ...HandlerFunc) RouteNameFunc
		Head(string, ...HandlerFunc) RouteNameFunc
		Options(string, ...HandlerFunc) RouteNameFunc
		Patch(string, ...HandlerFunc) RouteNameFunc
		Trace(string, ...HandlerFunc) RouteNameFunc
		Any(string, ...HandlerFunc)

		// static content
		StaticHandler(string, int, bool, bool, []string) HandlerFunc
		Static(string, string, int) RouteNameFunc
		StaticFS(string, string, int) RouteNameFunc
		StaticWeb(string, string, int) RouteNameFunc
		StaticServe(string, ...string) RouteNameFunc
		StaticContent(string, string, []byte) RouteNameFunc
		StaticEmbedded(string, string, func(string) ([]byte, error), func() []string) RouteNameFunc
		Favicon(string, ...string) RouteNameFunc

		// templates
		Layout(string) MuxAPI // returns itself

		// errors
		OnError(int, HandlerFunc)
		EmitError(int, *Context)
	}

	muxAPI struct {
		mux            *serveMux
		doneMiddleware Middleware
		apiRoutes      []*route // used to register the .Done middleware
		relativePath   string
		middleware     Middleware
	}
)

var _ MuxAPI = &muxAPI{}

var (
	// errAPIContextNotFound returns an error with message: 'From .API: "Context *iris.Context could not be found..'
	errAPIContextNotFound = errors.New("From .API: Context *iris.Context could not be found.")
	// errDirectoryFileNotFound returns an error with message: 'Directory or file %s couldn't found. Trace: +error trace'
	errDirectoryFileNotFound = errors.New("Directory or file %s couldn't found. Trace: %s")
)

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party chosen because it has more fun
func Party(relativePath string, handlersFn ...HandlerFunc) MuxAPI {
	return Default.Party(relativePath, handlersFn...)
}

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party chosen because it has more fun
func (api *muxAPI) Party(relativePath string, handlersFn ...HandlerFunc) MuxAPI {
	parentPath := api.relativePath
	dot := string(subdomainIndicator[0])
	if len(parentPath) > 0 && parentPath[0] == slashByte && strings.HasSuffix(relativePath, dot) { // if ends with . , example: admin., it's subdomain->
		parentPath = parentPath[1:] // remove first slash
	}

	fullpath := parentPath + relativePath
	middleware := convertToHandlers(handlersFn)
	// append the parent's +child's handlers
	middleware = joinMiddleware(api.middleware, middleware)

	return &muxAPI{relativePath: fullpath, mux: api.mux, apiRoutes: make([]*route, 0), middleware: middleware, doneMiddleware: api.doneMiddleware}
}

// Use registers Handler middleware
func Use(handlers ...Handler) {
	Default.Use(handlers...)
}

// UseFunc registers HandlerFunc middleware
func UseFunc(handlersFn ...HandlerFunc) {
	Default.UseFunc(handlersFn...)
}

// Done registers Handler 'middleware' the only difference from .Use is that it
// should be used BEFORE any party route registered or AFTER ALL party's routes have been registered.
//
// returns itself
func Done(handlers ...Handler) MuxAPI {
	return Default.Done(handlers...)
}

// DoneFunc registers HandlerFunc 'middleware' the only difference from .Use is that it
// should be used BEFORE any party route registered or AFTER ALL party's routes have been registered.
//
// returns itself
func DoneFunc(handlersFn ...HandlerFunc) MuxAPI {
	return Default.DoneFunc(handlersFn...)
}

// Use registers Handler middleware
func (api *muxAPI) Use(handlers ...Handler) {
	api.middleware = append(api.middleware, handlers...)
}

// UseFunc registers HandlerFunc middleware
func (api *muxAPI) UseFunc(handlersFn ...HandlerFunc) {
	api.Use(convertToHandlers(handlersFn)...)
}

// Done registers Handler 'middleware' the only difference from .Use is that it
// should be used BEFORE any party route registered or AFTER ALL party's routes have been registered.
//
// returns itself
func (api *muxAPI) Done(handlers ...Handler) MuxAPI {
	if len(api.apiRoutes) > 0 { // register these middleware on previous-party-defined routes, it called after the party's route methods (Handle/HandleFunc/Get/Post/Put/Delete/...)
		for i, n := 0, len(api.apiRoutes); i < n; i++ {
			api.apiRoutes[i].middleware = append(api.apiRoutes[i].middleware, handlers...)
		}
	} else {
		// register them on the doneMiddleware, which will be used on Handle to append these middlweare as the last handler(s)
		api.doneMiddleware = append(api.doneMiddleware, handlers...)
	}

	return api
}

// Done registers HandlerFunc 'middleware' the only difference from .Use is that it
// should be used BEFORE any party route registered or AFTER ALL party's routes have been registered.
//
// returns itself
func (api *muxAPI) DoneFunc(handlersFn ...HandlerFunc) MuxAPI {
	return api.Done(convertToHandlers(handlersFn)...)
}

// Handle registers a route to the server's router
// if empty method is passed then registers handler(s) for all methods, same as .Any, but returns nil as result
func Handle(method string, registedPath string, handlers ...Handler) RouteNameFunc {
	return Default.Handle(method, registedPath, handlers...)
}

// HandleFunc registers and returns a route with a method string, path string and a handler
// registedPath is the relative url path
func HandleFunc(method string, registedPath string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.HandleFunc(method, registedPath, handlersFn...)
}

// Handle registers a route to the server's router
// if empty method is passed then registers handler(s) for all methods, same as .Any, but returns nil as result
func (api *muxAPI) Handle(method string, registedPath string, handlers ...Handler) RouteNameFunc {
	if method == "" { // then use like it was .Any
		for _, k := range AllMethods {
			api.Handle(k, registedPath, handlers...)
		}
		return nil
	}

	fullpath := api.relativePath + registedPath // for now, keep the last "/" if any,  "/xyz/"

	middleware := joinMiddleware(api.middleware, handlers)

	// here we separate the subdomain and relative path
	subdomain := ""
	path := fullpath

	if dotWSlashIdx := strings.Index(path, subdomainIndicator); dotWSlashIdx > 0 {
		subdomain = fullpath[0 : dotWSlashIdx+1] // admin.
		path = fullpath[dotWSlashIdx+1:]         // /
	}

	// we splitted the path and subdomain parts so we're ready to check only the path,
	// otherwise we will had problems with subdomains
	// if the user wants beta:= iris.Party("/beta"); beta.Get("/") to be registered as
	//: /beta/ then should disable the path correction OR register it like: beta.Get("//")
	// this is only for the party's roots in order to have expected paths,
	// as we do with iris.Get("/") which is localhost:8080 as RFC points, not localhost:8080/
	if api.mux.correctPath && registedPath == slash { // check the given relative path
		// remove last "/" if any, "/xyz/"
		if len(path) > 1 { // if it's the root, then keep it*
			if path[len(path)-1] == slashByte {
				// ok we are inside /xyz/
			}
		}
	}

	path = strings.Replace(path, "//", "/", -1) // fix the path if double //

	if len(api.doneMiddleware) > 0 {
		middleware = append(middleware, api.doneMiddleware...) // register the done middleware, if any
	}
	r := api.mux.register([]byte(method), subdomain, path, middleware)
	api.apiRoutes = append(api.apiRoutes, r)

	// should we remove the api.apiRoutes on the .Party (new children party) ?, No, because the user maybe use this party later
	// should we add to the 'inheritance tree' the api.apiRoutes, No, these are for this specific party only, because the user propably, will have unexpected behavior when using Use/UseFunc, Done/DoneFunc
	return r.setName
}

// HandleFunc registers and returns a route with a method string, path string and a handler
// registedPath is the relative url path
func (api *muxAPI) HandleFunc(method string, registedPath string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.Handle(method, registedPath, convertToHandlers(handlersFn)...)
}

// API converts & registers a custom struct to the router
// receives two parameters
// first is the request path (string)
// second is the custom struct (interface{}) which can be anything that has a *iris.Context as field.
// third is the common middlewares, it's optional
//
// Note that API's routes have their default-name to the full registed path,
// no need to give a special name for it, because it's not supposed to be used inside your templates.
//
// Recommend to use when you retrieve data from an external database,
// and the router-performance is not the (only) thing which slows the server's overall performance.
//
// This is a slow method, if you care about router-performance use the Handle/HandleFunc/Get/Post/Put/Delete/Trace/Options... instead
func API(path string, restAPI HandlerAPI, middleware ...HandlerFunc) {
	Default.API(path, restAPI, middleware...)
}

// API converts & registers a custom struct to the router
// receives two parameters
// first is the request path (string)
// second is the custom struct (interface{}) which can be anything that has a *iris.Context as field.
// third is the common middleware, it's optional
//
// Note that API's routes have their default-name to the full registed path,
// no need to give a special name for it, because it's not supposed to be used inside your templates.
//
// Recommend to use when you retrieve data from an external database,
// and the router-performance is not the (only) thing which slows the server's overall performance.
//
// This is a slow method, if you care about router-performance use the Handle/HandleFunc/Get/Post/Put/Delete/Trace/Options... instead
func (api *muxAPI) API(path string, restAPI HandlerAPI, middleware ...HandlerFunc) {
	// here we need to find the registed methods and convert them to handler funcs
	// methods are collected by method naming:  Get(),GetBy(...), Post(),PostBy(...), Put() and so on
	if len(path) == 0 {
		path = "/"
	}
	if path[0] != slashByte {
		//  the route's paths always starts with "/", when the client navigates, the router works without "/" also ,
		// but the developer should always prepend the slash ("/") to register the routes
		path = "/" + path
	}
	typ := reflect.ValueOf(restAPI).Type()
	contextField, found := typ.FieldByName("Context")
	if !found {
		panic(errAPIContextNotFound)
	}

	// check & register the Get(),Post(),Put(),Delete() and so on
	for _, methodName := range AllMethods {

		methodCapitalName := strings.Title(strings.ToLower(methodName))

		if method, found := typ.MethodByName(methodCapitalName); found {
			methodFunc := method.Func
			if !methodFunc.IsValid() || methodFunc.Type().NumIn() > 1 { // for any case
				continue
			}

			func(path string, typ reflect.Type, contextField reflect.StructField, methodFunc reflect.Value, method string) {
				var handlersFn []HandlerFunc

				handlersFn = append(handlersFn, middleware...)
				handlersFn = append(handlersFn, func(ctx *Context) {
					newController := reflect.New(typ).Elem()
					newController.FieldByName("Context").Set(reflect.ValueOf(ctx))
					methodFunc.Call([]reflect.Value{newController})
				})
				// register route
				api.HandleFunc(method, path, handlersFn...)
			}(path, typ, contextField, methodFunc, methodName)

		}

	}

	// check for GetBy/PostBy(id string, something_else string) , these must be requested by the same order.
	// (we could do this in the same top loop but I don't want)
	// GET, DELETE -> with url named parameters (/users/:id/:secondArgumentIfExists)
	// POST, PUT -> with post values (form)
	// all other with URL Parameters (?something=this&else=other
	//
	// or no, I changed my mind, let all be named parameters and let users to decide what info they need,
	// using the Context to take more values (post form,url params and so on).-

	paramPrefix := []byte("param")
	for _, methodName := range AllMethods {
		methodWithBy := strings.Title(strings.ToLower(methodName)) + "By"
		if method, found := typ.MethodByName(methodWithBy); found {
			methodFunc := method.Func
			if !methodFunc.IsValid() || methodFunc.Type().NumIn() < 2 { //it's By but it has not receive any arguments so its not api's
				continue
			}
			methodFuncType := methodFunc.Type()
			numInLen := methodFuncType.NumIn() // how much data we should receive from the request
			registedPath := path

			for i := 1; i < numInLen; i++ { // from 1 because the first is the 'object'
				if registedPath[len(registedPath)-1] == slashByte {
					registedPath += ":" + string(paramPrefix) + strconv.Itoa(i)
				} else {
					registedPath += "/:" + string(paramPrefix) + strconv.Itoa(i)
				}
			}

			func(registedPath string, typ reflect.Type, contextField reflect.StructField, methodFunc reflect.Value, paramsLen int, method string) {
				var handlersFn []HandlerFunc

				handlersFn = append(handlersFn, middleware...)
				handlersFn = append(handlersFn, func(ctx *Context) {
					newController := reflect.New(typ).Elem()
					newController.FieldByName("Context").Set(reflect.ValueOf(ctx))
					args := make([]reflect.Value, paramsLen+1, paramsLen+1)
					args[0] = newController
					j := 1

					ctx.VisitUserValues(func(k []byte, v interface{}) {
						if bytes.HasPrefix(k, paramPrefix) {
							args[j] = reflect.ValueOf(v.(string))

							j++ // the first parameter is the context, other are the path parameters, j++ to be align with (API's registered)paramsLen
						}
					})

					methodFunc.Call(args)
				})
				// register route
				api.HandleFunc(method, registedPath, handlersFn...)
			}(registedPath, typ, contextField, methodFunc, numInLen-1, methodName)

		}

	}

}

// Get registers a route for the Get http method
func Get(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Get(path, handlersFn...)
}

// Post registers a route for the Post http method
func Post(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Post(path, handlersFn...)
}

// Put registers a route for the Put http method
func Put(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Put(path, handlersFn...)
}

// Delete registers a route for the Delete http method
func Delete(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Delete(path, handlersFn...)
}

// Connect registers a route for the Connect http method
func Connect(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Connect(path, handlersFn...)
}

// Head registers a route for the Head http method
func Head(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Head(path, handlersFn...)
}

// Options registers a route for the Options http method
func Options(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Options(path, handlersFn...)
}

// Patch registers a route for the Patch http method
func Patch(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Patch(path, handlersFn...)
}

// Trace registers a route for the Trace http method
func Trace(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.Trace(path, handlersFn...)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func Any(registedPath string, handlersFn ...HandlerFunc) {
	Default.Any(registedPath, handlersFn...)

}

// Get registers a route for the Get http method
func (api *muxAPI) Get(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodGet, path, handlersFn...)
}

// Post registers a route for the Post http method
func (api *muxAPI) Post(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodPost, path, handlersFn...)
}

// Put registers a route for the Put http method
func (api *muxAPI) Put(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodPut, path, handlersFn...)
}

// Delete registers a route for the Delete http method
func (api *muxAPI) Delete(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodDelete, path, handlersFn...)
}

// Connect registers a route for the Connect http method
func (api *muxAPI) Connect(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodConnect, path, handlersFn...)
}

// Head registers a route for the Head http method
func (api *muxAPI) Head(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodHead, path, handlersFn...)
}

// Options registers a route for the Options http method
func (api *muxAPI) Options(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodOptions, path, handlersFn...)
}

// Patch registers a route for the Patch http method
func (api *muxAPI) Patch(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodPatch, path, handlersFn...)
}

// Trace registers a route for the Trace http method
func (api *muxAPI) Trace(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodTrace, path, handlersFn...)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (api *muxAPI) Any(registedPath string, handlersFn ...HandlerFunc) {
	for _, k := range AllMethods {
		api.HandleFunc(k, registedPath, handlersFn...)
	}
}

// StaticHandler returns a Handler to serve static system directory
// Accepts 5 parameters
//
// first is the systemPath (string)
// Path to the root directory to serve files from.
//
// second is the  stripSlashes (int) level
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
//
// third is the compress (bool)
// Transparently compresses responses if set to true.
//
// The server tries minimizing CPU usage by caching compressed files.
// It adds fasthttp.FSCompressedFileSuffix suffix to the original file name and
// tries saving the resulting compressed file under the new file name.
// So it is advisable to give the server write access to Root
// and to all inner folders in order to minimze CPU usage when serving
// compressed responses.
//
// fourth is the generateIndexPages (bool)
// Index pages for directories without files matching IndexNames
// are automatically generated if set.
//
// Directory index generation may be quite slow for directories
// with many files (more than 1K), so it is discouraged enabling
// index pages' generation for such directories.
//
// fifth is the indexNames ([]string)
// List of index file names to try opening during directory access.
//
// For example:
//
//     * index.html
//     * index.htm
//     * my-super-index.xml
//
func StaticHandler(systemPath string, stripSlashes int, compress bool, generateIndexPages bool, indexNames []string) HandlerFunc {
	return Default.StaticHandler(systemPath, stripSlashes, compress, generateIndexPages, indexNames)
}

// StaticHandler returns a Handler to serve static system directory
// Accepts 5 parameters
//
// first is the systemPath (string)
// Path to the root directory to serve files from.
//
// second is the  stripSlashes (int) level
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
//
// third is the compress (bool)
// Transparently compresses responses if set to true.
//
// The server tries minimizing CPU usage by caching compressed files.
// It adds fasthttp.FSCompressedFileSuffix suffix to the original file name and
// tries saving the resulting compressed file under the new file name.
// So it is advisable to give the server write access to Root
// and to all inner folders in order to minimze CPU usage when serving
// compressed responses.
//
// fourth is the generateIndexPages (bool)
// Index pages for directories without files matching IndexNames
// are automatically generated if set.
//
// Directory index generation may be quite slow for directories
// with many files (more than 1K), so it is discouraged enabling
// index pages' generation for such directories.
//
// fifth is the indexNames ([]string)
// List of index file names to try opening during directory access.
//
// For example:
//
//     * index.html
//     * index.htm
//     * my-super-index.xml
//
func (api *muxAPI) StaticHandler(systemPath string, stripSlashes int, compress bool, generateIndexPages bool, indexNames []string) HandlerFunc {
	if indexNames == nil {
		indexNames = []string{}
	}
	fs := &fasthttp.FS{
		// Path to directory to serve.
		Root:       systemPath,
		IndexNames: indexNames,
		// Generate index pages if client requests directory contents.
		GenerateIndexPages: generateIndexPages,

		// Enable transparent compression to save network traffic.
		Compress:             compress,
		CacheDuration:        StaticCacheDuration,
		CompressedFileSuffix: CompressedFileSuffix,
	}

	if stripSlashes > 0 {
		fs.PathRewrite = fasthttp.NewPathSlashesStripper(stripSlashes)
	}

	// Create request handler for serving static files.
	h := fs.NewRequestHandler()
	return HandlerFunc(func(ctx *Context) {
		h(ctx.RequestCtx)
		errCode := ctx.RequestCtx.Response.StatusCode()
		if errCode == StatusNotFound || errCode == StatusBadRequest || errCode == StatusInternalServerError {
			api.mux.fireError(errCode, ctx)
		}
		if ctx.Pos < uint8(len(ctx.Middleware))-1 {
			ctx.Next() // for any case
		}

	})
}

// Static registers a route which serves a system directory
// this doesn't generates an index page which list all files
// no compression is used also, for these features look at StaticFS func
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
func Static(reqPath string, systemPath string, stripSlashes int) RouteNameFunc {
	return Default.Static(reqPath, systemPath, stripSlashes)
}

// Static registers a route which serves a system directory
// this doesn't generates an index page which list all files
// no compression is used also, for these features look at StaticFS func
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
func (api *muxAPI) Static(reqPath string, systemPath string, stripSlashes int) RouteNameFunc {
	if reqPath[len(reqPath)-1] != slashByte { // if / then /*filepath, if /something then /something/*filepath
		reqPath += slash
	}

	h := api.StaticHandler(systemPath, stripSlashes, false, false, nil)

	api.Head(reqPath+"*filepath", h)
	return api.Get(reqPath+"*filepath", h)
}

// StaticFS registers a route which serves a system directory
// this is the fastest method to serve static files
// generates an index page which list all files
// if you use this method it will generate compressed files also
// think this function as small fileserver with http
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
func StaticFS(reqPath string, systemPath string, stripSlashes int) RouteNameFunc {
	return Default.StaticFS(reqPath, systemPath, stripSlashes)
}

// StaticFS registers a route which serves a system directory
// this is the fastest method to serve static files
// generates an index page which list all files
// if you use this method it will generate compressed files also
// think this function as small fileserver with http
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
func (api *muxAPI) StaticFS(reqPath string, systemPath string, stripSlashes int) RouteNameFunc {
	if reqPath[len(reqPath)-1] != slashByte {
		reqPath += slash
	}

	h := api.StaticHandler(systemPath, stripSlashes, true, true, nil)
	api.Head(reqPath+"*filepath", h)
	return api.Get(reqPath+"*filepath", h)
}

// StaticWeb same as Static but if index.html exists and request uri is '/' then display the index.html's contents
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
// * if you don't know what to put on stripSlashes just 1
func StaticWeb(reqPath string, systemPath string, stripSlashes int) RouteNameFunc {
	return Default.StaticWeb(reqPath, systemPath, stripSlashes)
}

// StaticWeb same as Static but if index.html exists and request uri is '/' then display the index.html's contents
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
// * if you don't know what to put on stripSlashes just 1
// example: https://github.com/iris-contrib/examples/tree/4.0.0/static_web
func (api *muxAPI) StaticWeb(reqPath string, systemPath string, stripSlashes int) RouteNameFunc {
	if reqPath[len(reqPath)-1] != slashByte { // if / then /*filepath, if /something then /something/*filepath
		reqPath += slash
	}
	//todo: fs.go
	hasIndex := utils.Exists(systemPath + utils.PathSeparator + "index.html")
	var indexNames []string
	if hasIndex {
		indexNames = []string{"index.html"}
	}
	serveHandler := api.StaticHandler(systemPath, stripSlashes, false, !hasIndex, indexNames) // if not index.html exists then generate index.html which shows the list of files
	api.Head(reqPath+"*filepath", serveHandler)
	return api.Get(reqPath+"*filepath", serveHandler)
}

// StaticServe serves a directory as web resource
// it's the simpliest form of the Static* functions
// Almost same usage as StaticWeb
// accepts only one required parameter which is the systemPath ( the same path will be used to register the GET&HEAD routes)
// if second parameter is empty, otherwise the requestPath is the second parameter
// it uses gzip compression (compression on each request, no file cache)
func StaticServe(systemPath string, requestPath ...string) RouteNameFunc {
	return Default.StaticServe(systemPath, requestPath...)
}

// StaticServe serves a directory as web resource
// it's the simpliest form of the Static* functions
// Almost same usage as StaticWeb
// accepts only one required parameter which is the systemPath ( the same path will be used to register the GET&HEAD routes)
// if second parameter is empty, otherwise the requestPath is the second parameter
// it uses gzip compression (compression on each request, no file cache)
func (api *muxAPI) StaticServe(systemPath string, requestPath ...string) RouteNameFunc {
	var reqPath string

	if len(requestPath) == 0 {
		reqPath = strings.Replace(systemPath, fs.PathSeparator, slash, -1) // replaces any \ to /
		reqPath = strings.Replace(reqPath, "//", slash, -1)                // for any case, replaces // to /
		reqPath = strings.Replace(reqPath, ".", "", -1)                    // replace any dots (./mypath -> /mypath)
	} else {
		reqPath = requestPath[0]
	}

	return api.Get(reqPath+"/*file", func(ctx *Context) {
		filepath := ctx.Param("file")

		spath := strings.Replace(filepath, "/", fs.PathSeparator, -1)
		spath = path.Join(systemPath, spath)

		if !fs.DirectoryExists(spath) {
			ctx.NotFound()
			return
		}

		ctx.ServeFile(spath, true)
	})
}

// StaticContent serves bytes, memory cached, on the reqPath
// a good example of this is how the websocket server uses that to auto-register the /iris-ws.js
func StaticContent(reqPath string, contentType string, content []byte) RouteNameFunc {
	return Default.StaticContent(reqPath, contentType, content)
}

// StaticContent serves bytes, memory cached, on the reqPath
// a good example of this is how the websocket server uses that to auto-register the /iris-ws.js
func (api *muxAPI) StaticContent(reqPath string, cType string, content []byte) RouteNameFunc { // func(string) because we use that on websockets
	modtime := time.Now()
	modtimeStr := ""
	h := func(ctx *Context) {
		if modtimeStr == "" {
			modtimeStr = modtime.UTC().Format(ctx.framework.Config.TimeFormat)
		}

		if t, err := time.Parse(ctx.framework.Config.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && modtime.Before(t.Add(StaticCacheDuration)) {
			ctx.Response.Header.Del(contentType)
			ctx.Response.Header.Del(contentLength)
			ctx.SetStatusCode(StatusNotModified)
			return
		}

		ctx.Response.Header.Set(contentType, cType)
		ctx.Response.Header.Set(lastModified, modtimeStr)
		ctx.SetStatusCode(StatusOK)
		ctx.Response.SetBody(content)
	}
	api.Head(reqPath, h)
	return api.Get(reqPath, h)
}

// StaticEmbedded  used when files are distrubuted inside the app executable, using go-bindata mostly
// First parameter is the request path, the path which the files in the vdir(second parameter) will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets"
// Third parameter is the Asset function
// Forth parameter is the AssetNames function
//
// For more take a look at the
// example: https://github.com/iris-contrib/examples/tree/4.0.0/static_files_embedded
func StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) RouteNameFunc {
	return Default.StaticEmbedded(requestPath, vdir, assetFn, namesFn)
}

// StaticEmbedded  used when files are distrubuted inside the app executable, using go-bindata mostly
// First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets"
// Third parameter is the Asset function
// Forth parameter is the AssetNames function
//
// For more take a look at the
// example: https://github.com/iris-contrib/examples/tree/4.0.0/static_files_embedded
func (api *muxAPI) StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) RouteNameFunc {

	// check if requestPath already contains an asterix-match to anything symbol:  /path/*
	requestPath = strings.Replace(requestPath, "//", "/", -1)
	matchEverythingIdx := strings.IndexByte(requestPath, matchEverythingByte)
	paramName := "path"

	if matchEverythingIdx != -1 {
		// found so it should has a param name, take it
		paramName = requestPath[matchEverythingIdx+1:]
	} else {
		// make the requestPath
		if requestPath[len(requestPath)-1] == slashByte {
			// ends with / remove it
			requestPath = requestPath[0 : len(requestPath)-2]
		}

		requestPath += slash + "*" + paramName // $requestPath/*path
	}

	if len(vdir) > 0 {
		if vdir[0] == '.' { // first check for .wrong
			vdir = vdir[1:]
		}
		if vdir[0] == '/' || vdir[0] == os.PathSeparator { // second check for /something, (or ./something if we had dot on 0 it will be removed
			vdir = vdir[1:]
		}
	}

	// collect the names we are care for, because not all Asset used here, we need the vdir's assets.
	allNames := namesFn()

	var names []string
	for _, path := range allNames {
		// check if path is the path name we care for
		if !strings.HasPrefix(path, vdir) {
			continue
		}

		path = strings.Replace(path, "\\", "/", -1) // replace system paths with double slashes
		path = strings.Replace(path, "./", "/", -1) // replace ./assets/favicon.ico to /assets/favicon.ico in order to be ready for compare with the reqPath later
		path = path[len(vdir):]                     // set it as the its 'relative' ( we should re-setted it when assetFn will be used)
		names = append(names, path)

	}
	if len(names) == 0 {
		// we don't start the server yet, so:
		panic("iris.StaticEmbedded: Unable to locate any embedded files located to the (virtual) directory: " + vdir)
	}

	modtime := time.Now()
	modtimeStr := ""
	h := func(ctx *Context) {

		reqPath := ctx.Param(paramName)

		for _, path := range names {

			if path != reqPath {
				continue
			}

			if modtimeStr == "" {
				modtimeStr = modtime.UTC().Format(ctx.framework.Config.TimeFormat)
			}

			if t, err := time.Parse(ctx.framework.Config.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && modtime.Before(t.Add(StaticCacheDuration)) {
				ctx.Response.Header.Del(contentType)
				ctx.Response.Header.Del(contentLength)
				ctx.SetStatusCode(StatusNotModified)
				return
			}

			cType := fs.TypeByExtension(path)
			fullpath := vdir + path

			buf, err := assetFn(fullpath)

			if err != nil {
				continue
			}

			ctx.Response.Header.Set(contentType, cType)
			ctx.Response.Header.Set(lastModified, modtimeStr)

			ctx.SetStatusCode(StatusOK)
			ctx.SetContentType(cType)

			ctx.Response.SetBody(buf)
			return
		}

		// not found
		ctx.EmitError(StatusNotFound)

	}

	api.Head(requestPath, h)

	return api.Get(requestPath, h)
}

// Favicon serves static favicon
// accepts 2 parameters, second is optional
// favPath (string), declare the system directory path of the __.ico
// requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
// you can declare your own path if you have more than one favicon (desktop, mobile and so on)
//
// this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico (nothing special that you can't handle by yourself)
// Note that you have to call it on every favicon you have to serve automatically (dekstop, mobile and so on)
//
// panics on error
func Favicon(favPath string, requestPath ...string) RouteNameFunc {
	return Default.Favicon(favPath, requestPath...)
}

// Favicon serves static favicon
// accepts 2 parameters, second is optional
// favPath (string), declare the system directory path of the __.ico
// requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
// you can declare your own path if you have more than one favicon (desktop, mobile and so on)
//
// this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico (nothing special that you can't handle by yourself)
// Note that you have to call it on every favicon you have to serve automatically (dekstop, mobile and so on)
//
// panics on error
func (api *muxAPI) Favicon(favPath string, requestPath ...string) RouteNameFunc {
	f, err := os.Open(favPath)
	if err != nil {
		panic(errDirectoryFileNotFound.Format(favPath, err.Error()))
	}
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() { // if it's dir the try to get the favicon.ico
		fav := path.Join(favPath, "favicon.ico")
		f, err = os.Open(fav)
		if err != nil {
			//we try again with .png
			return api.Favicon(path.Join(favPath, "favicon.png"))
		}
		favPath = fav
		fi, _ = f.Stat()
	}

	cType := fs.TypeByExtension(favPath)
	// copy the bytes here in order to cache and not read the ico on each request.
	cacheFav := make([]byte, fi.Size())
	if _, err = f.Read(cacheFav); err != nil {
		panic(errDirectoryFileNotFound.Format(favPath, "Couldn't read the data bytes for Favicon: "+err.Error()))
	}
	modtime := ""
	h := func(ctx *Context) {
		if modtime == "" {
			modtime = fi.ModTime().UTC().Format(ctx.framework.Config.TimeFormat)
		}
		if t, err := time.Parse(ctx.framework.Config.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && fi.ModTime().Before(t.Add(StaticCacheDuration)) {
			ctx.Response.Header.Del(contentType)
			ctx.Response.Header.Del(contentLength)
			ctx.SetStatusCode(StatusNotModified)
			return
		}

		ctx.Response.Header.Set(contentType, cType)
		ctx.Response.Header.Set(lastModified, modtime)
		ctx.SetStatusCode(StatusOK)
		ctx.Response.SetBody(cacheFav)
	}

	reqPath := "/favicon" + path.Ext(fi.Name()) //we could use the filename, but because standards is /favicon.ico/.png.
	if len(requestPath) > 0 {
		reqPath = requestPath[0]
	}
	api.Head(reqPath, h)
	return api.Get(reqPath, h)
}

// Layout oerrides the parent template layout with a more specific layout for this Party
// returns this Party, to continue as normal
// example:
// my := iris.Party("/my").Layout("layouts/mylayout.html")
// 	{
// 		my.Get("/", func(ctx *iris.Context) {
// 			ctx.MustRender("page1.html", nil)
// 		})
// 	}
//
func Layout(tmplLayoutFile string) MuxAPI {
	return Default.Layout(tmplLayoutFile)
}

// Layout oerrides the parent template layout with a more specific layout for this Party
// returns this Party, to continue as normal
// example:
// my := iris.Party("/my").Layout("layouts/mylayout.html")
// 	{
// 		my.Get("/", func(ctx *iris.Context) {
// 			ctx.MustRender("page1.html", nil)
// 		})
// 	}
//
func (api *muxAPI) Layout(tmplLayoutFile string) MuxAPI {
	api.UseFunc(func(ctx *Context) {
		ctx.Set(TemplateLayoutContextKey, tmplLayoutFile)
		ctx.Next()
	})

	return api
}

// OnError registers a custom http error handler
func OnError(statusCode int, handlerFn HandlerFunc) {
	Default.OnError(statusCode, handlerFn)
}

// EmitError fires a custom http error handler to the client
//
// if no custom error defined with this statuscode, then iris creates one, and once at runtime
func EmitError(statusCode int, ctx *Context) {
	Default.EmitError(statusCode, ctx)
}

// OnError registers a custom http error handler
func (api *muxAPI) OnError(statusCode int, handlerFn HandlerFunc) {

	path := strings.Replace(api.relativePath, "//", "/", -1) // fix the path if double //
	staticPath := path
	// find the static path (on Party the path should be ALWAYS a static path, as we all know,
	// but do this check for any case)
	dynamicPathIdx := strings.IndexByte(path, parameterStartByte) // check for /mypath/:param

	if dynamicPathIdx == -1 {
		dynamicPathIdx = strings.IndexByte(path, matchEverythingByte) // check for /mypath/*param
	}

	if dynamicPathIdx > 1 { //yes after / and one character more ( /*param or /:param  will break the root path, and this is not allowed even on error handlers).
		staticPath = api.relativePath[0:dynamicPathIdx]
	}

	if staticPath == "/" {
		api.mux.registerError(statusCode, handlerFn) // register the user-specific error message, as the global error handler, for now.
		return
	}

	//after this, we have more than one error handler for one status code, and that's dangerous some times, but use it for non-globals error catching by your own risk
	// NOTES:
	// subdomains error will not work if same path of a non-subdomain (maybe a TODO for later)
	// errors for parties should be registered from the biggest path length to the smaller.

	// get the previous
	prevErrHandler := api.mux.errorHandlers[statusCode]
	if prevErrHandler == nil {
		/*
		 make a new one with the standard error message,
		 this will be used as the last handler if no other error handler catches the error (by prefix(?))
		*/
		prevErrHandler = HandlerFunc(func(ctx *Context) {
			ctx.ResetBody()
			ctx.SetStatusCode(statusCode)
			ctx.SetBodyString(statusText[statusCode])
		})
	}

	func(statusCode int, staticPath string, prevErrHandler Handler, newHandler Handler) { // to separate the logic
		errHandler := HandlerFunc(func(ctx *Context) {
			if strings.HasPrefix(ctx.PathString(), staticPath) { // yes the user should use OnError from longest to lower static path's length in order this to work, so we can find another way, like a builder on the end.
				newHandler.Serve(ctx)
				return
			}
			// serve with the user-specific global ("/") pure iris.OnError receiver Handler or the standar handler if OnError called only from inside a no-relative Party.
			prevErrHandler.Serve(ctx)
		})

		api.mux.registerError(statusCode, errHandler)
	}(statusCode, staticPath, prevErrHandler, handlerFn)

}

// EmitError fires a custom http error handler to the client
//
// if no custom error defined with this statuscode, then iris creates one, and once at runtime
func (api *muxAPI) EmitError(statusCode int, ctx *Context) {
	api.mux.fireError(statusCode, ctx)
}
