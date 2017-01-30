/*
Package iris the fastest go web framework in (this) Earth.

Basic usage
----------------------------------------------------------------------

package main

import  "github.com/kataras/iris"

func main() {
    iris.Get("/hi_json", func(ctx *iris.Context) {
        ctx.JSON(iris.StatusOK, iris.Map{
            "Name": "Iris",
            "Released":  "13 March 2016",
						"Stars": "5883",
        })
    })
    iris.ListenLETSENCRYPT("mydomain.com")
}

----------------------------------------------------------------------

package main

import  "github.com/kataras/iris"

func main() {
	s1 := iris.New()
	s1.Get("/hi_json", func(ctx *iris.Context) {
		ctx.JSON(iris.StatusOK, iris.Map{
			"Name": "Iris",
			"Released":  "13 March 2016",
			"Stars": "5883",
		})
	})

	s2 := iris.New()
	s2.Get("/hi_raw_html", func(ctx *iris.Context) {
		ctx.HTML(iris.StatusOK, "<b> Iris </b> welcomes <h1>you!</h1>")
	})

	go s1.Listen(":8080")
	s2.Listen(":1993")
}

----------------------------------------------------------------------

For middleware, template engines, response engines, sessions, websockets, mails, subdomains,
dynamic subdomains, routes, party of subdomains & routes and more

visit https://docs.iris-go.com
*/
package iris // import "github.com/kataras/iris"

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kataras/go-errors"
	"github.com/kataras/go-fs"
	"github.com/kataras/go-serializer"
	"github.com/kataras/go-sessions"
	"github.com/kataras/go-template"
	"github.com/kataras/go-template/html"
)

const (
	// IsLongTermSupport flag is true when the below version number is a long-term-support version
	IsLongTermSupport = false
	// Version is the current version number of the Iris web framework
	Version = "6.1.3"

	banner = `         _____      _
        |_   _|    (_)
          | |  ____ _  ___
          | | | __|| |/ __|
         _| |_| |  | |\__ \
        |_____|_|  |_||___/ ` + Version + ` `
)

// Default iris instance entry and its public fields, use it with iris.$anyPublicFuncOrField
var (
	Default *Framework
	Config  *Configuration
	Logger  *log.Logger // if you want colors in your console then you should use this https://github.com/iris-contrib/logger instead.
	Plugins PluginContainer
	// Router field holds the main http.Handler which can be changed.
	// if you want to get benefit with iris' context make use of:
	// ctx:= iris.AcquireCtx(http.ResponseWriter, *http.Request) to get the context at the beginning of your handler
	// iris.ReleaseCtx(ctx) to release/put the context to the pool, at the very end of your custom handler.
	//
	// Want to change the default Router's behavior to something else like Gorilla's Mux?
	// See more: https://github.com/iris-contrib/plugin/tree/master/gorillamux
	Router    http.Handler
	Websocket *WebsocketServer
	// Available is a channel type of bool, fired to true when the server is opened and all plugins ran
	// never fires false, if the .Close called then the channel is re-allocating.
	// the channel remains open until you close it.
	//
	// look at the http_test.go file for a usage example
	Available chan bool
)

// ResetDefault resets the iris.Default which is the instance which is used on the default iris station for
// iris.Get(all api functions)
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
		Set(options ...OptionSetter)
		Must(err error)

		Build()
		Serve(ln net.Listener) error
		Listen(addr string)
		ListenTLS(addr string, certFilePath string, keyFilePath string)
		ListenLETSENCRYPT(addr string, cacheOptionalStoreFilePath ...string)
		ListenUNIX(fileOrAddr string, fileMode os.FileMode)
		Close() error
		Reserve() error

		AcquireCtx(w http.ResponseWriter, r *http.Request) *Context
		ReleaseCtx(ctx *Context)

		CheckForUpdates(check bool)

		UseSessionDB(sessDB sessions.Database)
		DestroySessionByID(sid string)
		DestroyAllSessions()

		UseSerializer(contentType string, serializerEngine serializer.Serializer)
		UsePreRender(prerenderFunc PreRender)
		UseTemplateFunc(functionName string, function interface{})
		UseTemplate(tmplEngine template.Engine) *template.Loader

		UseGlobal(middleware ...Handler)
		UseGlobalFunc(middleware ...HandlerFunc)

		ChangeRouter(http.Handler)
		Lookup(routeName string) Route
		Lookups() []Route
		SetRouteOnline(r Route, HTTPMethod string) bool
		SetRouteOffline(r Route) bool
		ChangeRouteState(r Route, HTTPMethod string) bool

		Path(routeName string, optionalPathParameters ...interface{}) (routePath string)
		URL(routeName string, optionalPathParameters ...interface{}) (routeURL string)

		TemplateString(file string, binding interface{}, options ...map[string]interface{}) (parsedTemplate string)
		TemplateSourceString(src string, binding interface{}) (parsedTemplate string)
		SerializeToString(string, interface{}, ...map[string]interface{}) (serializedContent string)

		Cache(handlerToCache HandlerFunc, expiration time.Duration) (cachedHandler HandlerFunc)
	}

	// MuxAPI the visible api for the serveMux
	MuxAPI interface {
		Party(reqRelativeRootPath string, middleware ...HandlerFunc) MuxAPI
		// middleware serial, appending
		Use(middleware ...Handler) MuxAPI
		UseFunc(middleware ...HandlerFunc) MuxAPI
		Done(middleware ...Handler) MuxAPI
		DoneFunc(middleware ...HandlerFunc) MuxAPI

		// main handlers
		Handle(method string, reqPath string, middleware ...Handler) RouteNameFunc
		HandleFunc(method string, reqPath string, middleware ...HandlerFunc) RouteNameFunc
		API(reqRelativeRootPath string, api HandlerAPI, middleware ...HandlerFunc)

		// virtual method for "offline" routes new feature
		None(reqRelativePath string, Middleware ...HandlerFunc) RouteNameFunc
		// http methods
		Get(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Post(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Put(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Delete(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Connect(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Head(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Options(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Patch(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Trace(reqRelativePath string, middleware ...HandlerFunc) RouteNameFunc
		Any(reqRelativePath string, middleware ...HandlerFunc)

		// static content
		StaticServe(systemFilePath string, optionalReqRelativePath ...string) RouteNameFunc
		StaticContent(reqRelativePath string, contentType string, contents []byte) RouteNameFunc
		StaticEmbedded(reqRelativePath string, contentType string, assets func(string) ([]byte, error), assetsNames func() []string) RouteNameFunc
		Favicon(systemFilePath string, optionalReqRelativePath ...string) RouteNameFunc
		// static file system
		StaticHandler(reqRelativePath string, systemPath string, showList bool, enableGzip bool, exceptRoutes ...Route) HandlerFunc
		StaticWeb(reqRelativePath string, systemPath string, exceptRoutes ...Route) RouteNameFunc

		// party layout for template engines
		Layout(layoutTemplateFileName string) MuxAPI

		// errors
		OnError(statusCode int, handler HandlerFunc)
		EmitError(statusCode int, ctx *Context)
	}

	// RouteNameFunc the func returns from the MuxAPi's methods, optionally sets the name of the Route (*route)
	//
	// You can find the Route by iris.Lookup("theRouteName")
	// you can set a route name as: myRoute := iris.Get("/mypath", handler)("theRouteName")
	// that will set a name to the route and returns its iris.Route instance for further usage.
	//
	RouteNameFunc func(customRouteName string) Route
)

// Framework is our God |\| Google.Search('Greek mythology Iris')
//
// Implements the FrameworkAPI
type Framework struct {
	*muxAPI
	// HTTP Server runtime fields is the iris' defined main server, developer can use unlimited number of servers
	// note: they're available after .Build, and .Serve/Listen/ListenTLS/ListenLETSENCRYPT/ListenUNIX
	ln        net.Listener
	srv       *http.Server
	Available chan bool
	//
	// Router field holds the main http.Handler which can be changed.
	// if you want to get benefit with iris' context make use of:
	// ctx:= iris.AcquireCtx(http.ResponseWriter, *http.Request) to get the context at the beginning of your handler
	// iris.ReleaseCtx(ctx) to release/put the context to the pool, at the very end of your custom handler.
	//
	// Want to change the default Router's behavior to something else like Gorilla's Mux?
	// See more: https://github.com/iris-contrib/plugin/tree/master/gorillamux
	Router http.Handler

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
		// in order to be able to call $instance.Websocket.OnConnection
		// the whole ws configuration and websocket server is really initialized only on first OnConnection
		s.Websocket = NewWebsocketServer(s)

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

// Must panics on error, it panics on registered iris' logger
func Must(err error) {
	Default.Must(err)
}

// Must panics on error, it panics on registered iris' logger
func (s *Framework) Must(err error) {
	if err != nil {
		//	s.Logger.Panicf("%s. Trace:\n%s", err, debug.Stack())
		s.Logger.Panic(err)
	}
}

// Build builds the whole framework's parts together
// DO NOT CALL IT MANUALLY IF YOU ARE NOT:
// SERVE IRIS BEHIND AN EXTERNAL CUSTOM net/http.Server, CAN BE CALLED ONCE PER IRIS INSTANCE FOR YOUR SAFETY
func Build() {
	Default.Build()
}

// Build builds the whole framework's parts together
// DO NOT CALL IT MANUALLY IF YOU ARE NOT:
// SERVE IRIS BEHIND AN EXTERNAL CUSTOM nethttp.Server, CAN BE CALLED ONCE PER IRIS INSTANCE FOR YOUR SAFETY
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

		//  prepare the mux runtime fields again, for any case
		s.mux.setCorrectPath(!s.Config.DisablePathCorrection)
		s.mux.setFireMethodNotAllowed(s.Config.FireMethodNotAllowed)

		// prepare the server's handler, we do that check because iris supports
		// custom routers (you can take the routes registered by iris using iris.Lookups function)
		if s.Router == nil {
			// build and get the default mux' handler(*Context)
			serve := s.mux.BuildHandler()
			// build the net/http.Handler to bind it to the servers
			defaultHandler := ToNativeHandler(s, serve)

			s.Router = defaultHandler
		}

		// set the mux' hostname (for multi subdomain routing)
		s.mux.hostname = ParseHostname(s.Config.VHost)
		if s.ln != nil { // user called Listen functions or Serve,
			// create the main server
			s.srv = &http.Server{
				ReadTimeout:    s.Config.ReadTimeout,
				WriteTimeout:   s.Config.WriteTimeout,
				MaxHeaderBytes: s.Config.MaxHeaderBytes,
				TLSNextProto:   s.Config.TLSNextProto,
				ConnState:      s.Config.ConnState,
				Handler:        s.Router,
				Addr:           s.Config.VHost,
				ErrorLog:       s.Logger,
			}
			if s.Config.TLSNextProto != nil {
				s.srv.TLSNextProto = s.Config.TLSNextProto
			}
			if s.Config.ConnState != nil {
				s.srv.ConnState = s.Config.ConnState
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
	// build the handler and all other components
	s.Build()
	// fire all PreListen plugins
	s.Plugins.DoPreListen(s)

	// catch any panics to the user defined logger.
	defer func() {
		if err := recover(); err != nil {
			s.Logger.Panic(err)
		}
	}()

	// prepare for 'after serve' actions
	var stop uint32
	go func() {
		// wait for the server's Serve func (309 mill is a lot or not, I found that this number is the perfect for most of the environments.)
		time.Sleep(309 * time.Millisecond)
		if atomic.LoadUint32(&stop) > 0 {
			return
		}
		// print the banner
		// fire the PostListen plugins
		// wait for system channel interrupt
		// fire the PostInterrupt plugins or close and exit.
		s.postServe()
	}()

	serverStartUpErr := s.srv.Serve(ln)
	if serverStartUpErr != nil {
		// if an error then it would be nice to stop the banner and all next plugin events.
		atomic.AddUint32(&stop, 1)
	}
	// finally return the error or block here, remember,
	// you can always use the iris.Available to 'see' if and when the server is up (virtually),
	// until go1.8 these are our best options.
	return serverStartUpErr
}

// runs only when server starts without errors
// what it does?
// 0. print the banner
// 1. fire the PostListen plugins
// 2. wait for system channel interrupt
// 3. fire the PostInterrupt plugins or close and exit.
func (s *Framework) postServe() {

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

	// catch custom plugin event for interrupt
	// Example: https://github.com/iris-contrib/examples/tree/master/os_interrupt
	s.Plugins.DoPostInterrupt(s)
	if !s.Plugins.PostInterruptFired() {
		// if no PostInterrupt events fired, then I assume that the user doesn't cares
		// so close the server automatically.
		if err := s.Close(); err != nil {
			if s.Config.IsDevelopment {
				s.Logger.Printf("Error while closing the server: %s\n", err)
			}
		}
		os.Exit(1)
	}
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
	// only here, other Listen functions should throw an error if port is missing.
	// User should know how to fix them on ListenUNIX/ListenTLS/ListenLETSENCRYPT/Serve,
	// they are used by more 'advanced' devs, mostly.

	if portIdx := strings.IndexByte(addr, ':'); portIdx < 0 {
		// missing port part, add it
		addr = addr + ":80"
	}

	ln, err := TCPKeepAlive(addr)
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
// example: https://github.com/iris-contrib/examples/blob/master/letsencrypt/main.go
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
// example: https://github.com/iris-contrib/examples/blob/master/letsencrypt/main.go
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

	// starts a second server which listening on HOST:80 to redirect all requests to the HTTPS://HOST:PORT
	Proxy(ParseHostname(addr)+":80", "https://"+addr)
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
func AcquireCtx(w http.ResponseWriter, r *http.Request) *Context {
	return Default.AcquireCtx(w, r)
}

// AcquireCtx gets an Iris' Context from pool
// see .ReleaseCtx & .Serve
func (s *Framework) AcquireCtx(w http.ResponseWriter, r *http.Request) *Context {
	ctx := s.contextPool.Get().(*Context) // Changed to use the pool's New 09/07/2016, ~ -4k nanoseconds(9 bench tests) per requests (better performance)
	ctx.ResponseWriter = acquireResponseWriter(w)
	ctx.Request = r
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
	// flush the body (on recorder) or just the status code (on basic response writer)
	// when all finished
	ctx.ResponseWriter.flushResponse()

	ctx.Middleware = nil
	ctx.session = nil
	ctx.Request = nil
	///TODO:
	ctx.ResponseWriter.releaseMe()
	ctx.values.Reset()

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
			return
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

// DestroySessionByID removes the session entry
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
//
// It's safe to use it even if you are not sure if a session with that id exists.
func DestroySessionByID(sid string) {
	Default.DestroySessionByID(sid)
}

// DestroySessionByID removes the session entry
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
//
// It's safe to use it even if you are not sure if a session with that id exists.
func (s *Framework) DestroySessionByID(sid string) {
	s.sessions.DestroyByID(sid)
}

// DestroyAllSessions removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
func DestroyAllSessions() {
	Default.DestroyAllSessions()
}

// DestroyAllSessions removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
func (s *Framework) DestroyAllSessions() {
	s.sessions.DestroyAll()
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
//
// Example: https://github.com/iris-contrib/examples/tree/master/template_engines/template_prerender
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
//
// Example: https://github.com/iris-contrib/examples/tree/master/template_engines/template_prerender
func (s *Framework) UsePreRender(pre PreRender) {
	s.templates.usePreRender(pre)
}

// UseTemplateFunc sets or replaces a TemplateFunc from the shared available TemplateFuncMap
// defaults are the iris.URL and iris.Path, all the template engines supports the following:
// {{ url "mynamedroute" "pathParameter_ifneeded"} }
// {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
// {{ render "header.html" }}
// {{ render_r "header.html" }} // partial relative path to current page
// {{ yield }}
// {{ current }}
//
// See more https:/github.com/iris-contrib/examples/tree/master/template_engines/template_funcmap
func UseTemplateFunc(functionName string, function interface{}) {
	Default.UseTemplateFunc(functionName, function)
}

// UseTemplateFunc sets or replaces a TemplateFunc from the shared available TemplateFuncMap
// defaults are the iris.URL and iris.Path, all the template engines supports the following:
// {{ url "mynamedroute" "pathParameter_ifneeded"} }
// {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
// {{ render "header.html" }}
// {{ render_r "header.html" }} // partial relative path to current page
// {{ yield }}
// {{ current }}
//
// See more https:/github.com/iris-contrib/examples/tree/master/template_engines/template_funcmap
func (s *Framework) UseTemplateFunc(functionName string, function interface{}) {
	s.templates.SharedFuncs[functionName] = function
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
// It should be called right before Listen functions
func UseGlobal(handlers ...Handler) {
	Default.UseGlobal(handlers...)
}

// UseGlobalFunc registers HandlerFunc middleware  to the beginning, prepends them instead of append
//
// Use it when you want to add a global middleware to all parties, to all routes in  all subdomains
// It should be called right before Listen functions
func UseGlobalFunc(handlersFn ...HandlerFunc) {
	Default.UseGlobalFunc(handlersFn...)
}

// UseGlobal registers Handler middleware  to the beginning, prepends them instead of append
//
// Use it when you want to add a global middleware to all parties, to all routes in  all subdomains
// It should be called right before Listen functions
func (s *Framework) UseGlobal(handlers ...Handler) {
	if len(s.mux.lookups) > 0 {
		for _, r := range s.mux.lookups {
			r.middleware = append(handlers, r.middleware...)
		}
		return
	}

	s.Use(handlers...)
}

// UseGlobalFunc registers HandlerFunc middleware to the beginning, prepends them instead of append
//
// Use it when you want to add a global middleware to all parties, to all routes in  all subdomains
// It should be called right before Listen functions
func (s *Framework) UseGlobalFunc(handlersFn ...HandlerFunc) {
	s.UseGlobal(convertToHandlers(handlersFn)...)
}

// ChangeRouter force-changes the pre-defined iris' router while RUNTIME
// this function can be used to wrap the existing router with other.
// You can already do all these things with plugins, this function is a sugar for the craziest among us.
//
// Example of its only usage:
// https://github.com/iris-contrib/plugin/blob/master/cors/plugin.go#L22
// https://github.com/iris-contrib/plugin/blob/master/cors/plugin.go#L25
// https://github.com/iris-contrib/plugin/blob/master/cors/plugin.go#L28
//
// It's recommended that you use Plugin.PreBuild to change the router BEFORE the BUILD state.
func ChangeRouter(h http.Handler) {
	Default.ChangeRouter(h)
}

// ChangeRouter force-changes the pre-defined iris' router while RUNTIME
// this function can be used to wrap the existing router with other.
// You can already do all these things with plugins, this function is a sugar for the craziest among us.
//
// Example of its only usage:
// https://github.com/iris-contrib/plugin/blob/master/cors/plugin.go#L22
// https://github.com/iris-contrib/plugin/blob/master/cors/plugin.go#L25
// https://github.com/iris-contrib/plugin/blob/master/cors/plugin.go#L28
//
// It's recommended that you use Plugin.PreBuild to change the router BEFORE the BUILD state.
func (s *Framework) ChangeRouter(h http.Handler) {
	s.Router = h
	s.srv.Handler = h
}

///TODO: Inside note for author:
// make one and only one common API interface for all iris' supported Routers(gorillamux,httprouter,corsrouter)

// Lookup returns a registered route by its name
func Lookup(routeName string) Route {
	return Default.Lookup(routeName)
}

// Lookups returns all registered routes
func Lookups() []Route {
	return Default.Lookups()
}

// Lookup returns a registered route by its name
func (s *Framework) Lookup(routeName string) Route {
	r := s.mux.lookup(routeName)
	if nil == r {
		return nil
	}
	return r
}

// Lookups returns all registered routes
func (s *Framework) Lookups() (routes []Route) {
	// silly but...
	for i := range s.mux.lookups {
		routes = append(routes, s.mux.lookups[i])
	}
	return
}

// SetRouteOnline sets the state of the route to "online" with a specific http method
// it re-builds the router
//
// returns true if state was actually changed
//
// see context.ExecRoute(routeName),
// iris.None(...) and iris.SetRouteOnline/SetRouteOffline
// For more details look: https://github.com/kataras/iris/issues/585
//
// Example: https://github.com/iris-contrib/examples/tree/master/route_state
func SetRouteOnline(r Route, HTTPMethod string) bool {
	return Default.SetRouteOnline(r, HTTPMethod)
}

// SetRouteOffline sets the state of the route to "offline" and re-builds the router
//
// returns true if state was actually changed
//
// see context.ExecRoute(routeName),
// iris.None(...) and iris.SetRouteOnline/SetRouteOffline
// For more details look: https://github.com/kataras/iris/issues/585
//
// Example: https://github.com/iris-contrib/examples/tree/master/route_state
func SetRouteOffline(r Route) bool {
	return Default.SetRouteOffline(r)
}

// ChangeRouteState changes the state of the route.
// iris.MethodNone for offline
// and iris.MethodGet/MethodPost/MethodPut/MethodDelete /MethodConnect/MethodOptions/MethodHead/MethodTrace/MethodPatch for online
// it re-builds the router
//
// returns true if state was actually changed
//
// see context.ExecRoute(routeName),
// iris.None(...) and iris.SetRouteOnline/SetRouteOffline
// For more details look: https://github.com/kataras/iris/issues/585
//
// Example: https://github.com/iris-contrib/examples/tree/master/route_state
func ChangeRouteState(r Route, HTTPMethod string) bool {
	return Default.ChangeRouteState(r, HTTPMethod)
}

// SetRouteOnline sets the state of the route to "online" with a specific http method
// it re-builds the router
//
// returns true if state was actually changed
func (s *Framework) SetRouteOnline(r Route, HTTPMethod string) bool {
	return s.ChangeRouteState(r, HTTPMethod)
}

// SetRouteOffline sets the state of the route to "offline" and re-builds the router
//
// returns true if state was actually changed
func (s *Framework) SetRouteOffline(r Route) bool {
	return s.ChangeRouteState(r, MethodNone)
}

// ChangeRouteState changes the state of the route.
// iris.MethodNone for offline
// and iris.MethodGet/MethodPost/MethodPut/MethodDelete /MethodConnect/MethodOptions/MethodHead/MethodTrace/MethodPatch for online
// it re-builds the router
//
// returns true if state was actually changed
func (s *Framework) ChangeRouteState(r Route, HTTPMethod string) bool {
	if r != nil {
		nonSpecificMethod := len(HTTPMethod) == 0
		if r.Method() != HTTPMethod {
			if nonSpecificMethod {
				r.SetMethod(MethodGet) // if no method given, then do it for "GET" only
			} else {
				r.SetMethod(HTTPMethod)
			}
			// re-build the router/main handler
			s.Router = ToNativeHandler(s, s.mux.BuildHandler())
			return true
		}
	}
	return false
}

// Path used to check arguments with the route's named parameters and return the correct url
// if parse failed returns empty string
func Path(routeName string, args ...interface{}) string {
	return Default.Path(routeName, args...)
}

func joinPathArguments(args ...interface{}) []interface{} {
	arguments := args[0:]
	for i, v := range arguments {
		if arr, ok := v.([]string); ok {
			if len(arr) > 0 {
				interfaceArr := make([]interface{}, len(arr))
				for j, sv := range arr {
					interfaceArr[j] = sv
				}
				// replace the current slice
				// with the first string element (always as interface{})
				arguments[i] = interfaceArr[0]
				// append the rest of them to the slice itself
				// the range is not affected by these things in go,
				// so we are safe to do it.
				arguments = append(args, interfaceArr[1:]...)
			}
		}
	}
	return arguments
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

	arguments := joinPathArguments(args...)

	return fmt.Sprintf(r.formattedPath, arguments...)
}

// DecodeQuery returns the uri parameter as url (string)
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.QueryUnescape
func DecodeQuery(path string) string {
	if path == "" {
		return ""
	}
	encodedPath, err := url.QueryUnescape(path)
	if err != nil {
		return path
	}
	return encodedPath
}

// DecodeURL returns the decoded uri
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.Parse
func DecodeURL(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return u.String()
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
	arguments := joinPathArguments(args...)

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
// if <=2 seconds then it tries to find it though request header's "cache-control" maxage value
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
	ce := newCachedMuxEntry(s, bodyHandler, expiration)
	return ce.Serve
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------MuxAPI implementation------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type muxAPI struct {
	mux            *serveMux
	doneMiddleware Middleware
	apiRoutes      []*route // used to register the .Done middleware
	relativePath   string
	middleware     Middleware
}

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
func Use(handlers ...Handler) MuxAPI {
	return Default.Use(handlers...)
}

// UseFunc registers HandlerFunc middleware
func UseFunc(handlersFn ...HandlerFunc) MuxAPI {
	return Default.UseFunc(handlersFn...)
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
// returns itself
func (api *muxAPI) Use(handlers ...Handler) MuxAPI {
	api.middleware = append(api.middleware, handlers...)
	return api
}

// UseFunc registers HandlerFunc middleware
// returns itself
func (api *muxAPI) UseFunc(handlersFn ...HandlerFunc) MuxAPI {
	return api.Use(convertToHandlers(handlersFn)...)
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
func Handle(method string, registeredPath string, handlers ...Handler) RouteNameFunc {
	return Default.Handle(method, registeredPath, handlers...)
}

// HandleFunc registers and returns a route with a method string, path string and a handler
// registeredPath is the relative url path
func HandleFunc(method string, registeredPath string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.HandleFunc(method, registeredPath, handlersFn...)
}

// Handle registers a route to the server's router
// if empty method is passed then registers handler(s) for all methods, same as .Any, but returns nil as result
func (api *muxAPI) Handle(method string, registeredPath string, handlers ...Handler) RouteNameFunc {
	if method == "" { // then use like it was .Any
		for _, k := range AllMethods {
			api.Handle(k, registeredPath, handlers...)
		}
		return nil
	}

	fullpath := api.relativePath + registeredPath // for now, keep the last "/" if any,  "/xyz/"

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
	if api.mux.correctPath && registeredPath == slash { // check the given relative path
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
	r := api.mux.register(method, subdomain, path, middleware)

	api.apiRoutes = append(api.apiRoutes, r)
	// should we remove the api.apiRoutes on the .Party (new children party) ?, No, because the user maybe use this party later
	// should we add to the 'inheritance tree' the api.apiRoutes, No, these are for this specific party only, because the user propably, will have unexpected behavior when using Use/UseFunc, Done/DoneFunc
	return r.setName
}

// HandleFunc registers and returns a route with a method string, path string and a handler
// registeredPath is the relative url path
func (api *muxAPI) HandleFunc(method string, registeredPath string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.Handle(method, registeredPath, convertToHandlers(handlersFn)...)
}

// API converts & registers a custom struct to the router
// receives two parameters
// first is the request path (string)
// second is the custom struct (interface{}) which can be anything that has a *iris.Context as field.
// third is the common middlewares, it's optional
//
// Note that API's routes have their default-name to the full registered path,
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
// Note that API's routes have their default-name to the full registered path,
// no need to give a special name for it, because it's not supposed to be used inside your templates.
//
// Recommend to use when you retrieve data from an external database,
// and the router-performance is not the (only) thing which slows the server's overall performance.
//
// This is a slow method, if you care about router-performance use the Handle/HandleFunc/Get/Post/Put/Delete/Trace/Options... instead
func (api *muxAPI) API(path string, restAPI HandlerAPI, middleware ...HandlerFunc) {
	// here we need to find the registered methods and convert them to handler funcs
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

	paramPrefix := "param"
	for _, methodName := range AllMethods {
		methodWithBy := strings.Title(strings.ToLower(methodName)) + "By"
		if method, found := typ.MethodByName(methodWithBy); found {
			methodFunc := method.Func
			if !methodFunc.IsValid() || methodFunc.Type().NumIn() < 2 { //it's By but it has not receive any arguments so its not api's
				continue
			}
			methodFuncType := methodFunc.Type()
			numInLen := methodFuncType.NumIn() // how much data we should receive from the request
			registeredPath := path

			for i := 1; i < numInLen; i++ { // from 1 because the first is the 'object'
				if registeredPath[len(registeredPath)-1] == slashByte {
					registeredPath += ":" + string(paramPrefix) + strconv.Itoa(i)
				} else {
					registeredPath += "/:" + string(paramPrefix) + strconv.Itoa(i)
				}
			}

			func(registeredPath string, typ reflect.Type, contextField reflect.StructField, methodFunc reflect.Value, paramsLen int, method string) {
				var handlersFn []HandlerFunc

				handlersFn = append(handlersFn, middleware...)
				handlersFn = append(handlersFn, func(ctx *Context) {
					newController := reflect.New(typ).Elem()
					newController.FieldByName("Context").Set(reflect.ValueOf(ctx))
					args := make([]reflect.Value, paramsLen+1, paramsLen+1)
					args[0] = newController
					j := 1

					ctx.VisitValues(func(k string, v interface{}) {
						if strings.HasPrefix(k, paramPrefix) {
							args[j] = reflect.ValueOf(v.(string))

							j++ // the first parameter is the context, other are the path parameters, j++ to be align with (API's registered)paramsLen
						}
					})

					methodFunc.Call(args)
				})
				// register route
				api.HandleFunc(method, registeredPath, handlersFn...)
			}(registeredPath, typ, contextField, methodFunc, numInLen-1, methodName)

		}

	}

}

// None registers an "offline" route
// see context.ExecRoute(routeName),
// iris.None(...) and iris.SetRouteOnline/SetRouteOffline
// For more details look: https://github.com/kataras/iris/issues/585
//
// Example: https://github.com/iris-contrib/examples/tree/master/route_state
func None(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return Default.None(path, handlersFn...)
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
func Any(registeredPath string, handlersFn ...HandlerFunc) {
	Default.Any(registeredPath, handlersFn...)
}

// None registers an "offline" route
// see context.ExecRoute(routeName),
// iris.None(...) and iris.SetRouteOnline/SetRouteOffline
// For more details look: https://github.com/kataras/iris/issues/585
//
// Example: https://github.com/iris-contrib/examples/tree/master/route_state
func (api *muxAPI) None(path string, handlersFn ...HandlerFunc) RouteNameFunc {
	return api.HandleFunc(MethodNone, path, handlersFn...)
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
func (api *muxAPI) Any(registeredPath string, handlersFn ...HandlerFunc) {
	for _, k := range AllMethods {
		api.HandleFunc(k, registeredPath, handlersFn...)
	}
}

// if / then returns /*wildcard or /something then /something/*wildcard
// if empty then returns /*wildcard too
func validateWildcard(reqPath string, paramName string) string {
	if reqPath[len(reqPath)-1] != slashByte {
		reqPath += slash
	}
	reqPath += "*" + paramName
	return reqPath
}

func (api *muxAPI) registerResourceRoute(reqPath string, h HandlerFunc) RouteNameFunc {
	api.Head(reqPath, h)
	return api.Get(reqPath, h)
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
	h := func(ctx *Context) {
		ctx.SetClientCachedBody(StatusOK, content, cType, modtime)
	}

	return api.registerResourceRoute(reqPath, h)
}

// StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
// First parameter is the request path, the path which the files in the vdir(second parameter) will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets"
// Third parameter is the Asset function
// Forth parameter is the AssetNames function
//
// For more take a look at the
// example: https://github.com/iris-contrib/examples/tree/master/static_files_embedded
func StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) RouteNameFunc {
	return Default.StaticEmbedded(requestPath, vdir, assetFn, namesFn)
}

// StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
// First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets"
// Third parameter is the Asset function
// Forth parameter is the AssetNames function
//
// For more take a look at the
// example: https://github.com/iris-contrib/examples/tree/master/static_files_embedded
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
	h := func(ctx *Context) {

		reqPath := ctx.Param(paramName)

		for _, path := range names {

			if path != reqPath {
				continue
			}

			cType := fs.TypeByExtension(path)
			fullpath := vdir + path

			buf, err := assetFn(fullpath)

			if err != nil {
				continue
			}

			ctx.SetClientCachedBody(StatusOK, buf, cType, modtime)
			return
		}

		// not found or error
		ctx.EmitError(StatusNotFound)

	}

	return api.registerResourceRoute(requestPath, h)
}

// Favicon serves static favicon
// accepts 2 parameters, second is optional
// favPath (string), declare the system directory path of the __.ico
// requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
// you can declare your own path if you have more than one favicon (desktop, mobile and so on)
//
// this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico (nothing special that you can't handle by yourself)
// Note that you have to call it on every favicon you have to serve automatically (desktop, mobile and so on)
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
// Note that you have to call it on every favicon you have to serve automatically (desktop, mobile and so on)
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

			ctx.ResponseWriter.Header().Del(contentType)
			ctx.ResponseWriter.Header().Del(contentLength)
			ctx.SetStatusCode(StatusNotModified)
			return
		}

		ctx.ResponseWriter.Header().Set(contentType, cType)
		ctx.ResponseWriter.Header().Set(lastModified, modtime)
		ctx.SetStatusCode(StatusOK)
		ctx.Write(cacheFav)
	}

	reqPath := "/favicon" + path.Ext(fi.Name()) //we could use the filename, but because standards is /favicon.ico/.png.
	if len(requestPath) > 0 {
		reqPath = requestPath[0]
	}

	return api.registerResourceRoute(reqPath, h)
}

// StaticHandler returns a new Handler which serves static files
func StaticHandler(reqPath string, systemPath string, showList bool, enableGzip bool) HandlerFunc {
	return Default.StaticHandler(reqPath, systemPath, showList, enableGzip)
}

// StaticHandler returns a new Handler which serves static files
func (api *muxAPI) StaticHandler(reqPath string, systemPath string, showList bool, enableGzip bool, exceptRoutes ...Route) HandlerFunc {
	// here we separate the path from the subdomain (if any), we care only for the path
	// fixes a bug when serving static files via a subdomain
	fullpath := api.relativePath + reqPath
	path := fullpath
	if dotWSlashIdx := strings.Index(path, subdomainIndicator); dotWSlashIdx > 0 {
		path = fullpath[dotWSlashIdx+1:]
	}

	h := NewStaticHandlerBuilder(systemPath).
		Path(path).
		Listing(showList).
		Gzip(enableGzip).
		Except(exceptRoutes...).
		Build()

	managedStaticHandler := func(ctx *Context) {
		h(ctx)
		prevStatusCode := ctx.ResponseWriter.StatusCode()
		if prevStatusCode >= 400 { // we have an error
			// fire the custom error handler
			api.mux.fireError(prevStatusCode, ctx)
		}
		// go to the next middleware
		if ctx.Pos < len(ctx.Middleware)-1 {
			ctx.Next()
		}
	}
	return managedStaticHandler
}

// StaticWeb returns a handler that serves HTTP requests
// with the contents of the file system rooted at directory.
//
// first parameter: the route path
// second parameter: the system directory
// third OPTIONAL parameter: the exception routes
//      (= give priority to these routes instead of the static handler)
// for more options look iris.StaticHandler.
//
//     iris.StaticWeb("/static", "./static")
//
// As a special case, the returned file server redirects any request
// ending in "/index.html" to the same path, without the final
// "index.html".
//
// StaticWeb calls the StaticHandler(reqPath, systemPath, listingDirectories: false, gzip: false ).
func StaticWeb(reqPath string, systemPath string, exceptRoutes ...Route) RouteNameFunc {
	return Default.StaticWeb(reqPath, systemPath, exceptRoutes...)
}

// StaticWeb returns a handler that serves HTTP requests
// with the contents of the file system rooted at directory.
//
// first parameter: the route path
// second parameter: the system directory
// third OPTIONAL parameter: the exception routes
//      (= give priority to these routes instead of the static handler)
// for more options look iris.StaticHandler.
//
//     iris.StaticWeb("/static", "./static")
//
// As a special case, the returned file server redirects any request
// ending in "/index.html" to the same path, without the final
// "index.html".
//
// StaticWeb calls the StaticHandler(reqPath, systemPath, listingDirectories: false, gzip: false ).
func (api *muxAPI) StaticWeb(reqPath string, systemPath string, exceptRoutes ...Route) RouteNameFunc {
	h := api.StaticHandler(reqPath, systemPath, false, false, exceptRoutes...)
	routePath := validateWildcard(reqPath, "file")
	return api.registerResourceRoute(routePath, h)
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
			if w, ok := ctx.IsRecording(); ok {
				w.Reset()
			}
			ctx.SetStatusCode(statusCode)
			ctx.WriteString(statusText[statusCode])
		})
	}

	func(statusCode int, staticPath string, prevErrHandler Handler, newHandler Handler) { // to separate the logic
		errHandler := HandlerFunc(func(ctx *Context) {
			if strings.HasPrefix(ctx.Path(), staticPath) { // yes the user should use OnError from longest to lower static path's length in order this to work, so we can find another way, like a builder on the end.
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
