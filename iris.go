// Package iris is a web framework which provides efficient and well-designed
// tools with robust set of features to create your awesome and
// high-performance web application powered by unlimited potentials and portability
//
// For view engines, render engines, sessions,
// websockets, subdomains, automatic-TLS,
// context support with 50+ handy http functions,
// dynamic subdomains, router & routes,
// parties of subdomains & routes,
// access control, typescript compiler,
// basicauth,internalization, logging,
// and much more,
//
// please visit https://godoc.org/gopkg.in/kataras/iris.v6
package iris

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/geekypanda/httpcache"
	"github.com/kataras/go-errors"
)

const (
	// Version is the current version number of the Iris web framework
	Version = "6.2.0"

	codeName = `√Νεxτ`

	banner = `         _____      _
        |_   _|    (_)
          | |  ____ _  ___
          | | | __|| |/ __|
         _| |_| |  | |\__ \
        |_____|_|  |_||___/ ` + codeName
)

// Default is the field which keeps an empty `Framework`
// instance with its default configuration (config can change at runtime).
//
// Use that as `iris.Default.Handle(...)`
// or create a new, ex: `app := iris.New(); app.Handle(...)`
var (
	Default *Framework
	// ResetDefault resets the `.Default`
	// to an empty *Framework with the default configuration.
	//
	// Note: ResetDefault field itself can be setted
	//       to custom function too.
	ResetDefault = func() { Default = New() }
)

func init() {
	ResetDefault()
}

// Framework is our God |\| Google.Search('Greek mythology Iris').
type Framework struct {
	// Router is the Router API, REST Routing, Static files & favicon,
	// Grouping, Custom HTTP Errors,  Subdomains and more.
	//
	// This field is available before 'Boot' but the routes are actually registered after 'Boot'
	// if no RouterBuilderPolicy was .Adapt(ed) by user then
	// it throws a panic with detailed information of how-to-fix it.
	*Router

	// Config contains the configuration fields
	// all fields defaults to something that is working, developers don't have to set it.
	//
	// can be setted via .New, .Set and .New(.YAML)
	Config *Configuration

	// policies contains the necessary information about the application's components.
	// - LoggerPolicy
	// - EventPolicy
	//      - Boot
	//      - Build
	//      - Interrupted
	//      - Recover
	// - RouterReversionPolicy
	//      - StaticPath
	//      - WildcardPath
	//      - Param
	//      - URLPath
	// - RouterBuilderPolicy
	// - RouterWrapperPolicy
	// - RenderPolicy
	// - TemplateFuncsPolicy
	// - SessionsPolicy
	//
	// These are setted by user's call to .Adapt
	policies Policies

	// TLSNextProto optionally specifies a function to take over
	// ownership of the provided TLS connection when an NPN/ALPN
	// protocol upgrade has occurred. The map key is the protocol
	// name negotiated. The Handler argument should be used to
	// handle HTTP requests and will initialize the Request's TLS
	// and RemoteAddr if not already set. The connection is
	// automatically closed when the function returns.
	// If TLSNextProto is nil, HTTP/2 support is enabled automatically.
	TLSNextProto map[string]func(*http.Server, *tls.Conn, http.Handler) // same as http.Server.TLSNextProto

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state. See the
	// ConnState type and associated constants for details.
	ConnState func(net.Conn, http.ConnState) // same as http.Server.ConnState

	// Shutdown gracefully shuts down the server without interrupting any
	// active connections. Shutdown works by first closing all open
	// listeners, then closing all idle connections, and then waiting
	// indefinitely for connections to return to idle and then shut down.
	// If the provided context expires before the shutdown is complete,
	// then the context's error is returned.
	//
	// Shutdown does not attempt to close nor wait for hijacked
	// connections such as WebSockets. The caller of Shutdown should
	// separately notify such long-lived connections of shutdown and wait
	// for them to close, if desired.
	Shutdown func(context.Context) error

	closedManually bool // true if closed via .Shutdown, used to not throw a panic on s.handlePanic when closing the app's server

	once sync.Once // used to 'Boot' once
}

var defaultGlobalLoggerOuput = log.New(os.Stdout, "", log.LstdFlags)

// DevLogger returns a new Logger which prints both ProdMode and DevMode messages
// to the default global logger printer.
//
// Usage: app := iris.New()
//        app.Adapt(iris.DevLogger())
//
// Users can always ignore that and adapt a custom LoggerPolicy,
// which will use your custom printer instead.
func DevLogger() LoggerPolicy {
	lastLog := time.Now()
	distanceDuration := 850 * time.Millisecond
	return func(mode LogMode, logMessage string) {
		if strings.Contains(logMessage, banner) {
			fmt.Print(logMessage)
			lastLog = time.Now()
			return
		}
		nowLog := time.Now()
		if nowLog.Before(lastLog.Add(distanceDuration)) {
			// don't use the log.Logger to print this message
			// if the last one was printed before some seconds.
			fmt.Println(logMessage)
			lastLog = nowLog
			return
		}
		// begin with new line in order to have the time once at the top
		// and the child logs below it.
		defaultGlobalLoggerOuput.Println("\u2192\n" + logMessage)
		lastLog = nowLog
	}
}

// New creates and returns a fresh Iris *Framework instance
// with the default configuration if no 'setters' parameters passed.
func New(setters ...OptionSetter) *Framework {
	cfg := DefaultConfiguration()
	s := &Framework{Config: &cfg}

	//  +------------------------------------------------------------+
	//  | Set the config passed from setters                         |
	//  | or use the default one                                     |
	//  +------------------------------------------------------------+
	s.Set(setters...)
	{
		//  +------------------------------------------------------------+
		//  | Module Name: Logger                                        |
		//  | On Init: If user didn't adapt a custom loggger then attach |
		//  |           a new logger using log.Logger as printer with    |
		//  |          some default options                              |
		//  +------------------------------------------------------------+

		// The logger policy is never nil and it doesn't defaults to an empty func,
		// instead it defaults to a logger with os.Stdout as the print target which prints
		// ONLY prodction level messages.
		// While in ProdMode Iris logs only panics and fatal errors.
		// You can override the default log policy with app.Adapt(iris.DevLogger())
		//                            or app.Adapt(iris.LoggerPolicy(customLogger))
		// to log both ProdMode and DevMode messages.
		//
		// Note:
		// The decision to not log everything and use middleware for http requests instead of built'n
		// is because I'm using Iris on production so I don't want many logs to my screens
		// while server is running.
		s.Adapt(LoggerPolicy(func(mode LogMode, logMessage string) {
			if mode == ProdMode {
				defaultGlobalLoggerOuput.Println(logMessage)
			}
		}))

		s.Adapt(EventPolicy{Boot: func(*Framework) {
			// Print the banner first, even if we have to print errors later.
			s.Log(DevMode, banner+"\n\n")
		}})

	}

	//  +------------------------------------------------------------+
	//  |                                                            |
	//  | Please take a look at the policy.go file first.            |
	//  | The EventPolicy contains all the necessary information     |
	//  | user should know about the framework's flow.               |
	//  |                                                            |
	//  +------------------------------------------------------------+

	//  +------------------------------------------------------------+
	//  | On Boot: Set the VHost and VScheme config fields           |
	//  |           based on the net.Listener which (or not)         |
	//  |           setted on Serve and Listen funcs.                |
	//  |                                                            |
	//  |           It's the only pre-defined Boot event because of  |
	//  |           any user's custom 'Build' events should know     |
	//  |           the Host of the server.                          |
	//  +------------------------------------------------------------+
	s.Adapt(EventPolicy{Boot: func(s *Framework) {
		// set the host and scheme
		if s.Config.VHost == "" { // if not setted by Listen functions
			s.Config.VHost = DefaultServerAddr
		}
		// if user didn't specified a scheme then get it from the VHost,
		// which is already setted at before statements
		if s.Config.VScheme == "" {
			// if :443 or :https then returns https:// otherwise http://
			s.Config.VScheme = ParseScheme(s.Config.VHost)
		}

	}})

	{
		//  +------------------------------------------------------------+
		//  | Module Name: Renderer                                      |
		//  | On Init: set templates and serializers                     |
		//  | and adapt the RenderPolicy for both                        |
		//  | templates and content-type specific renderer (serializer)  |
		//  | On Build: build the serializers and templates              |
		//  | based on the user's calls                                  |
		//  +------------------------------------------------------------+

		{
			//  +------------------------------------------------------------+
			//  | Module Name: Rich Content-Type Renderer                    |
			//  | On Init: Attach a new empty content-type serializers.      |
			//  | Adapt one RenderPolicy which is responsible                |
			//  | for json,jsonp,xml and markdown rendering                  |
			//  +------------------------------------------------------------+
			s.Adapt(restRenderPolicy)
		}
		{
			//  +------------------------------------------------------------+
			//  | Module Name: Template engine's funcs                       |
			//  | On Init: Adapt the reverse routing tmpl funcs              |
			//  |          for any template engine that will be registered   |
			//  +------------------------------------------------------------+
			s.Adapt(TemplateFuncsPolicy{
				"url":     s.URL,
				"urlpath": s.Path,
			}) // the entire template registration logic lives inside the ./adaptors/view now.

		}

	}

	{
		//  +------------------------------------------------------------+
		//  | Module Name: Router                                        |
		//  | On Init: Attach a new router, pass a new repository,       |
		//  |    an empty error handlers list, the context pool binded   |
		//  |    to the Framework and the root path "/"                  |
		//  | On Build: Use the policies to build the router's handler   |
		//  |            based on its route repository                   |
		//  +------------------------------------------------------------+

		s.Router = &Router{
			repository: new(routeRepository),
			Errors: &ErrorHandlers{
				handlers: make(map[int]Handler, 0),
			},
			Context: &contextPool{
				sync.Pool{New: func() interface{} { return &Context{framework: s} }},
			},
			relativePath: "/",
		}

		s.Adapt(EventPolicy{Build: func(*Framework) {
			// Author's notes:
			// Proxy for example has 0 routes registered but still uses the RouterBuilderPolicy
			// so we can't check only for it, we can check if it's nil and it has more than one registered
			// routes, then panic, if has no registered routes the user don't want to get errors about the router.

			// first check if it's not setted already by any Boot event.
			if s.Router.handler == nil {
				hasRoutes := s.Router.repository.Len() > 0
				routerBuilder := s.policies.RouterBuilderPolicy
				// and most importantly, check if the user has provided a router adaptor
				//                    at the same time has registered at least one route,
				// if not then it should panic here, iris can't run without a router attached to it
				// and default router not any more, user should select one from ./adaptors or
				// any other third-party adaptor may done by community.
				// I was coding the new iris version for more than 20 days(~200+ hours of code)
				// and I hope that once per application the addition of +1 line users have to put,
				// is not a big deal.
				if hasRoutes {
					if routerBuilder == nil {
						// this is important panic and app can't continue as we said.
						s.handlePanic(errRouterIsMissing.Format(s.Config.VHost))
						// don't trace anything else,
						// the detailed errRouterIsMissing message will tell the user what to do to fix that.
						os.Exit(0)
					}
				}

				if routerBuilder != nil {
					// buid the router using user's selection build policy
					s.Router.build(routerBuilder)

					s.Router.repository.OnMethodChanged(func(route RouteInfo, oldMethod string) {
						// set service not available temporarily until the router completes the building
						// this won't take more than 100ms, but we want to inform the user.
						s.Router.handler = ToNativeHandler(s, HandlerFunc(func(ctx *Context) {
							ctx.EmitError(StatusServiceUnavailable)
						}))
						// Re-build the whole router if state changed (from offline to online state mostly)
						s.Router.build(routerBuilder)
					})
				}
			}
		}})

	}

	{
		//  +------------------------------------------------------------+
		//  | Module Name: System                                        |
		//  | On Build: Check for updates on Build, async                |
		//  +------------------------------------------------------------+

		// On Build: local repository updates
		s.Adapt(EventPolicy{Build: func(*Framework) {
			if s.Config.CheckForUpdates {
				go CheckForUpdates(false)
			}
		}})
	}

	return s
}

// Set sets an option, configuration field to its Config
func (s *Framework) Set(setters ...OptionSetter) {
	for _, setter := range setters {
		setter.Set(s.Config)
	}
}

// Log logs to the defined logger policy.
//
// The default outputs to the os.Stdout when EnvMode is 'iris.ProdMode'
func (s *Framework) Log(mode LogMode, log string) {
	s.policies.LoggerPolicy(mode, log)
}

// Author's note:
//
// I implemented this and worked for configuration fields logs but I'm not sure
// if users want to print adaptor's configuration, i.e: websocket, sessions
// so comment this feature, and
// if/when I get good feedback from the private beta testers I'll uncomment these on public repo too.
//
//
// Logf same as .Log but receives arguments like fmt.Printf.
// Note: '%#v' and '%+v' named arguments are being printed, pretty.
//
// The default outputs to the os.Stdout when EnvMode is 'iris.ProdMode'
// func (s *Framework) Logf(mode LogMode, format string, v ...interface{}) {
// 	if mode == DevMode {
// 		marker := 0
// 		for i := 0; i < len(format); i++ {
// 			if format[i] == '%' {
// 				if i < len(format)-2 {
// 					if part := format[i : i+3]; part == "%#v" || part == "%+v" {
// 						if len(v) > marker {
// 							spew.Config.Indent = "\t\t\t"
// 							spew.Config.DisablePointerAddresses = true
// 							spew.Config.DisableCapacities = true
// 							spew.Config.DisablePointerMethods = true
// 							spew.Config.DisableMethods = true
// 							arg := v[marker]
// 							format = strings.Replace(format, part, "%s", 1)
// 							v[marker] = spew.Sdump(arg)
// 						}
// 					}
// 				}
// 				marker++
// 			}
// 		}
// 	}
//
// 	log := fmt.Sprintf(format, v...)
//
// 	s.policies.LoggerPolicy(mode, log)
// }

// Must checks if the error is not nil, if it isn't
// panics on registered iris' logger or
// to a recovery event handler, otherwise does nothing.
func (s *Framework) Must(err error) {
	if err != nil {
		s.handlePanic(err)
	}
}

func (s *Framework) handlePanic(err error) {
	// if x, ok := err.(*net.OpError); ok && x.Op == "accept" {
	// 	return
	// }

	if err.Error() == http.ErrServerClosed.Error() && s.closedManually {
		//.Shutdown was called, log to dev not in prod (prod is only for critical errors.)
		// also do not try to recover from this error, remember, Shutdown was called manually here.
		s.Log(DevMode, "HTTP Server closed manually")
		return
	}

	if recoveryHandler := s.policies.EventPolicy.Recover; recoveryHandler != nil {
		recoveryHandler(s, err)
		return
	}
	// if not a registered recovery event handler found
	// then call the logger's Panic.
	s.Log(ProdMode, err.Error())
}

// Boot runs only once, automatically
//  when 'Serve/Listen/ListenTLS/ListenUNIX/ListenLETSENCRYPT' called.
// It's exported because you may want to build the router
//  and its components but not run the server.
//
// See ./httptest/httptest.go to understand its usage.
func (s *Framework) Boot() (firstTime bool) {
	s.once.Do(func() {
		// here execute the boot events, before build events, if exists, here is
		// where the user can make an event module to adapt custom routers and other things
		// fire the before build event
		s.policies.EventPolicy.Fire(s.policies.EventPolicy.Boot, s)

		// here execute the build events if exists
		// right before the Listen, all methods have been setted
		// usually is used to adapt third-party servers or proxies or load balancer(s)
		s.policies.EventPolicy.Fire(s.policies.EventPolicy.Build, s)
		firstTime = true
	})
	return
}

func (s *Framework) setupServe() (srv *http.Server, deferFn func()) {
	s.closedManually = false

	s.Boot()

	deferFn = func() {
		// post any panics to the user defined logger.
		if rerr := recover(); rerr != nil {
			if err, ok := rerr.(error); ok {
				s.handlePanic(err)
			}
		}
	}

	srv = &http.Server{
		ReadTimeout:    s.Config.ReadTimeout,
		WriteTimeout:   s.Config.WriteTimeout,
		MaxHeaderBytes: s.Config.MaxHeaderBytes,
		TLSNextProto:   s.TLSNextProto,
		ConnState:      s.ConnState,
		Addr:           s.Config.VHost,
		ErrorLog:       s.policies.LoggerPolicy.ToLogger(log.LstdFlags),
		Handler:        s.Router,
	}

	// Set the grace shutdown, it's just a func no need to make things complicated
	// all are managed by net/http now.
	s.Shutdown = func(ctx context.Context) error {
		// order matters, look s.handlePanic
		s.closedManually = true
		err := srv.Shutdown(ctx)
		return err
	}

	return
}

// Serve serves incoming connections from the given listener.
//
// Serve blocks until the given listener returns permanent error.
func (s *Framework) Serve(ln net.Listener) error {
	if ln == nil {
		return errors.New("nil net.Listener on Serve")
	}

	// if user called .Serve and doesn't uses any nginx-like balancers.
	if s.Config.VHost == "" {
		s.Config.VHost = ParseHost(ln.Addr().String())
	} // Scheme will be checked from Boot state.

	srv, fn := s.setupServe()
	defer fn()

	// print the banner and wait for system channel interrupt
	go s.postServe()
	return srv.Serve(ln)
}

func (s *Framework) postServe() {
	routerInfo := ""
	if routerNameVal := s.Config.Other[RouterNameConfigKey]; routerNameVal != nil {
		if routerName, ok := routerNameVal.(string); ok {
			routerInfo = "using " + routerName
		}
	}

	bannerMessage := fmt.Sprintf("Serving HTTP on %s port %d %s", ParseHostname(s.Config.VHost), ParsePort(s.Config.VHost), routerInfo)

	s.Log(DevMode, bannerMessage)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	// fire any custom interrupted events and at the end close and exit
	// if the custom event blocks then it decides what to do next.
	s.policies.Fire(s.policies.Interrupted, s)

	s.Shutdown(context.Background())
	os.Exit(1)
}

// Listen starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port
//
// If you need to manually monitor any error please use `.Serve` instead.
func (s *Framework) Listen(addr string) {
	addr = ParseHost(addr)

	// if .Listen called normally and VHost is not setted,
	// so it's Host is the Real listening addr and user-given
	if s.Config.VHost == "" {
		s.Config.VHost = addr // as it is
		// this will be set as the front-end listening addr
	} // VScheme will be checked on Boot.

	// this check, only here, other Listen functions should throw an error if port is missing.
	if portIdx := strings.IndexByte(addr, ':'); portIdx < 0 {
		// missing port part, add it
		addr = addr + ":80"
	}

	ln, err := TCPKeepAlive(addr)
	if err != nil {
		s.handlePanic(err)
		return
	}

	s.Must(s.Serve(ln))
}

// ListenTLS Starts a https server with certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port
//
//
// If you need to manually monitor any error please use `.Serve` instead.
func (s *Framework) ListenTLS(addr string, certFile, keyFile string) {
	addr = ParseHost(addr)

	{
		// set it before Boot, be-careful VHost and VScheme are used by nginx users too
		// we don't want to alt them.
		if s.Config.VHost == "" {
			s.Config.VHost = addr
			// this will be set as the front-end listening addr
		}
		if s.Config.VScheme == "" {
			s.Config.VScheme = SchemeHTTPS
		}
	}

	srv, fn := s.setupServe()
	// We are doing the same parts as .Serve does but instead we run srv.ListenAndServeTLS
	// because of un-exported net/http.server.go:setupHTTP2_ListenAndServeTLS function which
	// broke our previous flow but no problem :)
	defer fn()
	// print the banner and wait for system channel interrupt
	go s.postServe()
	s.Must(srv.ListenAndServeTLS(certFile, keyFile))
}

// ListenLETSENCRYPT starts a server listening at the specific nat address
// using key & certification taken from the letsencrypt.org 's servers
// it's also starts a second 'http' server to redirect all 'http://$ADDR_HOSTNAME:80' to the' https://$ADDR'
// it creates a cache file to store the certifications, for performance reasons, this file by-default is "./certcache"
// if you skip the second parameter then the cache file is "./letsencrypt.cache"
// if you want to disable cache then simple pass as second argument an empty empty string ""
//
// Note: HTTP/2 Push is not working with LETSENCRYPT, you have to use ListenTLS to enable HTTP/2
// Because net/http's author didn't exported the functions to tell the server that is using HTTP/2...
//
// example: https://github.com/kataras/iris/blob/v6/_examples/beginner/listen-letsencrypt/main.go
func (s *Framework) ListenLETSENCRYPT(addr string, cacheFileOptional ...string) {
	addr = ParseHost(addr)

	{
		// set it before Boot, be-careful VHost and VScheme are used by nginx users too
		// we don't want to alt them.
		if s.Config.VHost == "" {
			s.Config.VHost = addr
			// this will be set as the front-end listening addr
		}
		if s.Config.VScheme == "" {
			s.Config.VScheme = SchemeHTTPS
		}
	}

	ln, err := LETSENCRYPT(addr, addr, cacheFileOptional...)
	if err != nil {
		s.handlePanic(err)
		return
	}

	// starts a second server which listening on HOST:80 to redirect all requests to the HTTPS://HOST:PORT
	Proxy(ParseHostname(addr)+":80", "https://"+addr)
	s.Must(s.Serve(ln))
}

// ListenUNIX starts the process of listening to the new requests using a 'socket file',
// Note: this works only on unix
//
// If you need to manually monitor any error please use `.Serve` instead.
//
// example: https://github.com/kataras/iris/blob/v6/_examples/beginner/listen-unix/main.go
func (s *Framework) ListenUNIX(socketFile string, mode os.FileMode) {
	ln, err := UNIX(socketFile, mode)
	if err != nil {
		s.handlePanic(err)
		return
	}
	// *on unix listen we don't parse the host, because sometimes it causes problems to the user
	if s.Config.VHost == "" {
		s.Config.VHost = ln.Addr().String()
		// this will be set as the front-end listening addr
	}
	s.Must(s.Serve(ln))
}

func (s *Framework) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

// Adapt adapds a policy to the Framework.
// It accepts single or more objects that implements the iris.Policy.
// Iris provides some of them but you can build your own based on one or more of these:
// - iris.EventPolicy
// - iris.RouterReversionPolicy
// - iris.RouterBuilderPolicy
// - iris.RouterWrapperPolicy
// - iris.TemplateRenderPolicy
// - iris.TemplateFuncsPolicy
//
// With a Policy you can change the behavior of almost each of the existing Iris' features.
// See policy.go for more.
func (s *Framework) Adapt(policies ...Policy) {
	for i := range policies {
		policies[i].Adapt(&s.policies)
	}
}

// cachedMuxEntry is just a wrapper for the Cache functionality
// it seems useless but I prefer to keep the cached handler on its own memory stack,
// reason:  no clojures hell in the Cache function
type cachedMuxEntry struct {
	cachedHandler http.Handler
}

func newCachedMuxEntry(s *Framework, bodyHandler HandlerFunc, expiration time.Duration) *cachedMuxEntry {
	httpHandler := ToNativeHandler(s, bodyHandler)

	cachedHandler := httpcache.Cache(httpHandler, expiration)
	return &cachedMuxEntry{
		cachedHandler: cachedHandler,
	}
}

func (c *cachedMuxEntry) Serve(ctx *Context) {
	c.cachedHandler.ServeHTTP(ctx.ResponseWriter, ctx.Request)
}

// Cache is just a wrapper for a route's handler which you want to enable body caching
// Usage: iris.Default.Get("/", iris.Cache(func(ctx *iris.Context){
//    ctx.WriteString("Hello, world!") // or a template or anything else
// }, time.Duration(10*time.Second))) // duration of expiration
// if <=time.Second then it tries to find it though request header's "cache-control" maxage value
func (s *Framework) Cache(bodyHandler HandlerFunc, expiration time.Duration) HandlerFunc {
	ce := newCachedMuxEntry(s, bodyHandler, expiration)
	return ce.Serve
}

// Path used to check arguments with the route's named parameters and return the correct url
// if parse failed returns empty string.
// Used for reverse routing, depends on router adaptor.
//
// Examples:
// https://github.com/kataras/iris/tree/v6/adaptors/view/_examples/template_html_3 (gorillamux)
// https://github.com/kataras/iris/tree/v6/adaptors/view/_examples/template_html_4 (httprouter)
func (s *Framework) Path(routeName string, args ...interface{}) string {
	r := s.Router.Routes().Lookup(routeName)
	if r == nil {
		return ""
	}

	// why receive interface{}
	// but need string?
	// because the key:value are string for a route path
	// but in the template functions all works fine with ...string
	// except when the developer wants to pass that string from a binding
	// via Render, then the template will fail to render
	// because of expecting string; but got []string

	var argsString []string
	if len(args) > 0 {
		argsString = make([]string, len(args))
	}

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

	return s.policies.RouterReversionPolicy.URLPath(r, argsString...)
}

// URL returns the subdomain + host + Path(...optional named parameters if route is dynamic)
// returns an empty string if parse is failed.
// Used for reverse routing, depends on router adaptor.
//
// Examples:
// https://github.com/kataras/iris/tree/v6/adaptors/view/_examples/template_html_3 (gorillamux)
// https://github.com/kataras/iris/tree/v6/adaptors/view/_examples/template_html_4 (httprouter)
func (s *Framework) URL(routeName string, args ...interface{}) (url string) {
	r := s.Router.Routes().Lookup(routeName)
	if r == nil {
		return
	}

	scheme := s.Config.VScheme // if s.Config.VScheme was setted, that will be used instead of the real, in order to make easy to run behind nginx
	host := s.Config.VHost     // if s.Config.VHost was setted, that will be used instead of the real, in order to make easy to run behind nginx

	// if it's dynamic subdomain then the first argument is the subdomain part
	// for this part we are responsible not the custom routers
	if r.Subdomain() == DynamicSubdomainIndicator {
		if len(args) == 0 { // it's a wildcard subdomain but not arguments
			return
		}

		if subdomain, ok := args[0].(string); ok {
			host = subdomain + "." + host
		} else {
			// it is not array because we join them before. if not pass a string then this is not a subdomain part, return empty uri
			return
		}
		args = args[1:] // remove the subdomain part for the arguments,
	}

	if parsedPath := s.Path(routeName, args...); parsedPath != "" {
		url = scheme + host + parsedPath
	}

	return
}

// Regex takes pairs with the named path (without symbols) following by its expression
// and returns a middleware which will do a pure but effective validation using the regexp package.
//
// Note: '/adaptors/gorillamux' already supports regex path validation.
// It's useful while the developer uses the '/adaptors/httprouter' instead.
func (s *Framework) Regex(pairParamExpr ...string) HandlerFunc {
	srvErr := func(ctx *Context) {
		ctx.EmitError(StatusInternalServerError)
	}

	// just to check if router is adapted.
	wp := s.policies.RouterReversionPolicy.WildcardPath
	if wp == nil {
		s.Log(ProdMode, "regex cannot be used when a router policy is missing\n"+errRouterIsMissing.Format(s.Config.VHost).Error())
		return srvErr
	}

	if len(pairParamExpr)%2 != 0 {
		s.Log(ProdMode,
			"regex pre-compile error: the correct format is paramName, expression"+
				"paramName2, expression2. The len should be %2==0")
		return srvErr
	}

	// we do compile first to reduce the performance cost at serve time.
	pairs := make(map[string]*regexp.Regexp, len(pairParamExpr)/2)

	for i := 0; i < len(pairParamExpr)-1; i++ {
		expr := pairParamExpr[i+1]
		r, err := regexp.Compile(expr)
		if err != nil {
			s.Log(ProdMode, "regex '"+expr+"' failed. Trace: "+err.Error())
			return srvErr
		}

		pairs[pairParamExpr[i]] = r
		i++
	}

	// return the middleware
	return func(ctx *Context) {
		for k, v := range pairs {
			if !ctx.ParamValidate(v, k) {
				ctx.EmitError(StatusNotFound)
				return
			}
		}
		// otherwise continue to the next handler...
		ctx.Next()
	}
}

func (s *Framework) RegexSingle(paramName string, expr string, onFail HandlerFunc) HandlerFunc {

	// just to check if router is adapted.
	wp := s.policies.RouterReversionPolicy.WildcardPath
	if wp == nil {
		s.Log(ProdMode, "regex cannot be used when a router policy is missing\n"+errRouterIsMissing.Format(s.Config.VHost).Error())
		return onFail
	}

	// we do compile first to reduce the performance cost at serve time.
	r, err := regexp.Compile(expr)
	if err != nil {
		s.Log(ProdMode, "regex '"+expr+"' failed. Trace: "+err.Error())
		return onFail
	}

	// return the middleware
	return func(ctx *Context) {
		if !ctx.ParamValidate(r, paramName) {
			onFail(ctx)
			return
		}
		// otherwise continue to the next handler...
		ctx.Next()
	}
}

// RouteParam returns a named parameter as each router defines named path parameters.
// For example, with the httprouter(: as named param symbol):
// userid should return :userid.
// with gorillamux, userid should return {userid}
// or userid[1-9]+ should return {userid[1-9]+}.
// so basically we just wrap the raw parameter name
// with the start (and end) dynamic symbols of each router implementing the RouterReversionPolicy.
// It's an optional functionality but it can be used to create adaptors without even know the router
// that the user uses (which can be taken by app.Config.Other[iris.RouterNameConfigKey].
//
// Note: we don't need a function like ToWildcardParam because the developer
// can use the RouterParam with a combination with RouteWildcardPath.
//
// Example: https://github.com/iris-contrib/adaptors/blob/master/oauth/oauth.go
func (s *Framework) RouteParam(paramName string) string {
	if s.policies.RouterReversionPolicy.Param == nil {
		// all Iris' routers are implementing all features but third-parties may not, so make sure that the user
		// will get a useful message back.
		s.Log(DevMode, "cannot wrap a route named path parameter because the functionality was not implemented by the current router.")
		return ""
	}

	return s.policies.RouterReversionPolicy.Param(paramName)
}

// RouteWildcardPath returns a path converted to a 'dynamic' path
// for example, with the httprouter(wildcard symbol: '*'):
// ("/static", "path") should return /static/*path
// ("/myfiles/assets", "anything") should return /myfiles/assets/*anything
func (s *Framework) RouteWildcardPath(path string, paramName string) string {
	if s.policies.RouterReversionPolicy.WildcardPath == nil {
		s.Log(DevMode, "please use WildcardPath after .Adapt(router).\n"+errRouterIsMissing.Format(s.Config.VHost).Error())
		return ""
	}
	return s.policies.RouterReversionPolicy.WildcardPath(path, paramName)
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

var errTemplateRendererIsMissing = errors.New(
	`
manually call of Render for a template: '%s' without specified RenderPolicy!
Please .Adapt one of the available view engines inside 'kataras/iris/adaptors/view'.
By-default Iris supports five template engines:
 - standard html  | view.HTML(...)
 - django         | view.Django(...)
 - handlebars     | view.Handlebars(...)
 - pug(jade)      | view.Pug(...)
 - amber          | view.Amber(...)

Edit your main .go source file to adapt one of these and restart your app.
	i.e: lines (<---) were missing.
	-------------------------------------------------------------------
	import (
		"gopkg.in/kataras/iris.v6"
		"gopkg.in/kataras/iris.v6/adaptors/httprouter" // or gorillamux
		"gopkg.in/kataras/iris.v6/adaptors/view" // <--- this line
	)

	func main(){
		app := iris.New()
		// right below the iris.New():
		app.Adapt(httprouter.New()) // or gorillamux.New()

		app.Adapt(view.HTML("./templates", ".html")) // <--- and this line were missing.

		// the rest of your source code...
		// ...

		app.Listen("%s")
	}
	-------------------------------------------------------------------
 `)

// RenderOptions is a helper type for  the optional runtime options can be passed by user when Render called.
// I.e the "layout" or "gzip" option
// same as iris.Map but more specific name
type RenderOptions map[string]interface{}

// Render renders using the specific template or any other rich content renderer to the 'w'.
//
// Example of usage:
// - send an e-mail using a template engine that you already
//   adapted via: app.Adapt(view.HTML("./templates", ".html"))  or app.Adapt(iris.RenderPolicy(mycustomRenderer)).
//
// It can also render json,xml,jsonp and markdown by-default before or after .Build too.
func (s *Framework) Render(w io.Writer, name string, bind interface{}, options ...map[string]interface{}) error {
	ok, err := s.policies.RenderPolicy(w, name, bind, options...)
	if !ok {
		// ok is false ONLY WHEN there is no registered render policy
		// that is responsible for that 'name` (if contains dot '.' it's for templates).
		// We don't use default template engines on the new version,
		// so we should notice the user here, we could make it to panic but because that is on runtime
		// we don't want to panic for that, let's give a message if the user adapted a logger for dev.
		// And return that error in the case the user wasn't in dev mode, she/he can catch this error.

		// Also on the README we will add the .Adapt(iris.DevLogger()) to mention that
		// logging for any runtime info(except http server's panics and unexpected serious errors) is not enabled by-default.
		if strings.Contains(name, ".") {
			err = errTemplateRendererIsMissing.Format(name, s.Config.VHost)
			s.Log(DevMode, err.Error())
			return err
		}

	}
	return err
}
