package iris

import (
	"github.com/imdario/mergo"
	"github.com/kataras/go-options"
	"github.com/kataras/go-sessions"
	"github.com/valyala/fasthttp"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type (
	// OptionSetter sets a configuration field to the main configuration
	// used to help developers to write less and configure only what they really want and nothing else
	// example:
	// iris.New(iris.Configuration{Sessions:iris.SessionConfiguration{Cookie:"mysessionid"}, Websocket: iris.WebsocketConfiguration{Endpoint:"/my_endpoint"}})
	// now can be done also by using iris.Option$FIELD:
	// iris.New(irisOptionSessionsCookie("mycookieid"),iris.OptionWebsocketEndpoint("my_endpoint"))
	// benefits:
	// 1. user/dev have no worries what option to pass, he/she can just press iris.Option and all options should be shown to her/his editor's autocomplete-popup window
	// 2. can be passed with any order
	// 3. Can override previous configuration
	OptionSetter interface {
		// Set receives a pointer to the global Configuration type and does the job of filling it
		Set(c *Configuration)
	}
	// OptionSet implements the OptionSetter
	OptionSet func(c *Configuration)
)

// Set is the func which makes the OptionSet an OptionSetter, this is used mostly
func (o OptionSet) Set(c *Configuration) {
	o(c)
}

// Configuration the whole configuration for an iris instance ($instance.Config) or global iris instance (iris.Config)
// these can be passed via options also, look at the top of this file(configuration.go)
//
// Configuration is also implements the OptionSet so it's a valid option itself, this is briliant enough
type Configuration struct {
	// CheckForUpdates will try to search for newer version of Iris based on the https://github.com/kataras/iris/releases
	// If a newer version found then the app will ask the he dev/user if want to update the 'x' version
	// if 'y' is pressed then the updater will try to install the latest version
	// the updater, will notify the dev/user that the update is finished and should restart the App manually.
	// Notes:
	// 1. Experimental feature
	// 2. If setted to true, the app will have a little startup delay
	// 3. If you as developer edited the $GOPATH/src/github/kataras or any other Iris' Go dependencies at the past
	//    then the update process will fail.
	//
	// Usage: iris.Set(iris.OptionCheckForUpdates(true)) or
	//        iris.Config.CheckForUpdates = true or
	//        app := iris.New(iris.OptionCheckForUpdates(true))
	// Default is false
	CheckForUpdates bool

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

	// LoggerOut is the destination for output
	//
	// Default is os.Stdout
	LoggerOut io.Writer
	// LoggerPreffix is the logger's prefix to write at beginning of each line
	//
	// Default is [IRIS]
	LoggerPreffix string

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

	// DisableTemplateEngines set to true to disable loading the default template engine (html/template) and disallow the use of iris.UseEngine
	// default is false
	DisableTemplateEngines bool

	// IsDevelopment iris will act like a developer, for example
	// If true then re-builds the templates on each request
	// default is false
	IsDevelopment bool

	// TimeFormat time format for any kind of datetime parsing
	TimeFormat string

	// Charset character encoding for various rendering
	// used for templates and the rest of the responses
	// defaults to "UTF-8"
	Charset string

	// Gzip enables gzip compression on your Render actions, this includes any type of render, templates and pure/raw content
	// If you don't want to enable it globaly, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
	// defaults to false
	Gzip bool

	// Sessions contains the configs for sessions
	Sessions SessionsConfiguration

	// Websocket contains the configs for Websocket's server integration
	Websocket WebsocketConfiguration

	// Tester contains the configs for the test framework, so far we have only one because all test framework's configs are setted by the iris itself
	// You can find example on the https://github.com/kataras/iris/glob/master/context_test.go
	Tester TesterConfiguration

	// Other are the custom, dynamic options, can be empty
	// this fill used only by you to set any app's options you want
	// for each of an Iris instance
	Other options.Options
}

// Set implements the OptionSetter
func (c Configuration) Set(main *Configuration) {
	mergo.MergeWithOverwrite(main, c)
}

// All options starts with "Option" preffix in order to be easier to find what dev searching for
var (
	// OptionCheckForUpdates will try to search for newer version of Iris based on the https://github.com/kataras/iris/releases
	// If a newer version found then the app will ask the he dev/user if want to update the 'x' version
	// if 'y' is pressed then the updater will try to install the latest version
	// the updater, will notify the dev/user that the update is finished and should restart the App manually.
	// Notes:
	// 1. Experimental feature
	// 2. If setted to true, the app will have a little startup delay
	// 3. If you as developer edited the $GOPATH/src/github/kataras or any other Iris' Go dependencies at the past
	//    then the update process will fail.
	//
	// Usage: iris.Set(iris.OptionCheckForUpdates(true)) or
	//        iris.Config.CheckForUpdates = true or
	//        app := iris.New(iris.OptionCheckForUpdates(true))
	// Default is false
	OptionCheckForUpdates = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.CheckForUpdates = val
		}

	}

	// OptionDisablePathCorrection corrects and redirects the requested path to the registed path
	// for example, if /home/ path is requested but no handler for this Route found,
	// then the Router checks if /home handler exists, if yes,
	// (permant)redirects the client to the correct path /home
	//
	// Default is false
	OptionDisablePathCorrection = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.DisablePathCorrection = val
		}

	}

	// OptionDisablePathEscape when is false then its escapes the path, the named parameters (if any).
	OptionDisablePathEscape = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.DisablePathEscape = val
		}
	}

	// OptionDisableBanner outputs the iris banner at startup
	//
	// Default is false
	OptionDisableBanner = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.DisableBanner = val
		}
	}

	// OptionLoggerOut is the destination for output
	//
	// Default is os.Stdout
	OptionLoggerOut = func(val io.Writer) OptionSet {
		return func(c *Configuration) {
			c.LoggerOut = val
		}
	}

	// OptionLoggerPreffix is the logger's prefix to write at beginning of each line
	//
	// Default is [IRIS]
	OptionLoggerPreffix = func(val string) OptionSet {
		return func(c *Configuration) {
			c.LoggerPreffix = val
		}
	}

	// OptionProfilePath a the route path, set it to enable http pprof tool
	// Default is empty, if you set it to a $path, these routes will handled:
	OptionProfilePath = func(val string) OptionSet {
		return func(c *Configuration) {
			c.ProfilePath = val
		}
	}

	// OptionDisableTemplateEngines set to true to disable loading the default template engine (html/template) and disallow the use of iris.UseEngine
	// Default is false
	OptionDisableTemplateEngines = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.DisableTemplateEngines = val
		}
	}

	// OptionIsDevelopment iris will act like a developer, for example
	// If true then re-builds the templates on each request
	// Default is false
	OptionIsDevelopment = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.IsDevelopment = val
		}
	}

	// OptionTimeFormat time format for any kind of datetime parsing
	OptionTimeFormat = func(val string) OptionSet {
		return func(c *Configuration) {
			c.TimeFormat = val
		}
	}

	// OptionCharset character encoding for various rendering
	// used for templates and the rest of the responses
	// Default is "UTF-8"
	OptionCharset = func(val string) OptionSet {
		return func(c *Configuration) {
			c.Charset = val
		}
	}

	// OptionGzip enables gzip compression on your Render actions, this includes any type of render, templates and pure/raw content
	// If you don't want to enable it globaly, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
	// Default is false
	OptionGzip = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Gzip = val
		}
	}

	// OptionOther are the custom, dynamic options, can be empty
	// this fill used only by you to set any app's options you want
	// for each of an Iris instance
	OptionOther = func(val ...options.Options) OptionSet {
		opts := options.Options{}
		for _, opt := range val {
			for k, v := range opt {
				opts[k] = v
			}
		}
		return func(c *Configuration) {
			c.Other = opts
		}
	}
)

var (
	// DefaultTimeFormat default time format for any kind of datetime parsing
	DefaultTimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
	// StaticCacheDuration expiration duration for INACTIVE file handlers, it's a global configuration field to all iris instances
	StaticCacheDuration = 20 * time.Second
	// CompressedFileSuffix is the suffix to add to the name of
	// cached compressed file when using the .StaticFS function.
	//
	// Defaults to iris-fasthttp.gz
	CompressedFileSuffix = "iris-fasthttp.gz"
)

// Default values for base Iris conf
const (
	DefaultDisablePathCorrection = false
	DefaultDisablePathEscape     = false
	DefaultCharset               = "UTF-8"
	DefaultLoggerPreffix         = "[IRIS] "
)

var (
	// DefaultLoggerOut is the default logger's output
	DefaultLoggerOut = os.Stdout
)

// DefaultConfiguration returns the default configuration for an Iris station, fills the main Configuration
func DefaultConfiguration() Configuration {
	return Configuration{
		CheckForUpdates:        false,
		DisablePathCorrection:  DefaultDisablePathCorrection,
		DisablePathEscape:      DefaultDisablePathEscape,
		DisableBanner:          false,
		LoggerOut:              DefaultLoggerOut,
		LoggerPreffix:          DefaultLoggerPreffix,
		DisableTemplateEngines: false,
		IsDevelopment:          false,
		TimeFormat:             DefaultTimeFormat,
		Charset:                DefaultCharset,
		Gzip:                   false,
		ProfilePath:            "",
		Sessions:               DefaultSessionsConfiguration(),
		Websocket:              DefaultWebsocketConfiguration(),
		Tester:                 DefaultTesterConfiguration(),
		Other:                  options.Options{},
	}
}

// SessionsConfiguration the configuration for sessions
// has 6 fields
// first is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
// second enable if you want to decode the cookie's key also
// third is the time which the client's cookie expires
// forth is the cookie length (sessionid) int, defaults to 32, do not change if you don't have any reason to do
// fifth is the gcDuration (time.Duration) when this time passes it removes the unused sessions from the memory until the user come back
// sixth is the DisableSubdomainPersistence which you can set it to true in order dissallow your q subdomains to have access to the session cook
type SessionsConfiguration sessions.Config

// Set implements the OptionSetter of the sessions package
func (s SessionsConfiguration) Set(c *sessions.Config) {
	*c = sessions.Config(s).Validate()
}

var (
	// OptionSessionsCookie string, the session's client cookie name, for example: "qsessionid"
	OptionSessionsCookie = func(val string) OptionSet {
		return func(c *Configuration) {
			c.Sessions.Cookie = val
		}
	}

	// OptionSessionsDecodeCookie set it to true to decode the cookie key with base64 URLEncoding
	// Defaults to false
	OptionSessionsDecodeCookie = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Sessions.DecodeCookie = val
		}
	}

	// OptionSessionsExpires the duration of which the cookie must expires (created_time.Add(Expires)).
	// If you want to delete the cookie when the browser closes, set it to -1 but in this case, the server side's session duration is up to GcDuration
	//
	// Default infinitive/unlimited life duration(0)
	OptionSessionsExpires = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.Sessions.Expires = val
		}
	}

	// OptionSessionsCookieLength the length of the sessionid's cookie's value, let it to 0 if you don't want to change it
	// Defaults to 32
	OptionSessionsCookieLength = func(val int) OptionSet {
		return func(c *Configuration) {
			c.Sessions.CookieLength = val
		}
	}

	// OptionSessionsGcDuration every how much duration(GcDuration) the memory should be clear for unused cookies (GcDuration)
	// for example: time.Duration(2)*time.Hour. it will check every 2 hours if cookie hasn't be used for 2 hours,
	// deletes it from backend memory until the user comes back, then the session continue to work as it was
	//
	// Default 2 hours
	OptionSessionsGcDuration = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.Sessions.GcDuration = val
		}
	}

	// OptionSessionsDisableSubdomainPersistence set it to true in order dissallow your q subdomains to have access to the session cookie
	// defaults to false
	OptionSessionsDisableSubdomainPersistence = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Sessions.DisableSubdomainPersistence = val
		}
	}
)

var (
	universe time.Time // 0001-01-01 00:00:00 +0000 UTC
	// CookieExpireNever the default cookie's life for sessions, unlimited (23 years)
	CookieExpireNever = time.Now().AddDate(23, 0, 0)
)

const (
	// DefaultCookieName the secret cookie's name for sessions
	DefaultCookieName = "irissessionid"
	// DefaultSessionGcDuration  is the default Session Manager's GCDuration , which is 2 hours
	DefaultSessionGcDuration = time.Duration(2) * time.Hour
	// DefaultCookieLength is the default Session Manager's CookieLength, which is 32
	DefaultCookieLength = 32
)

// DefaultSessionsConfiguration the default configs for Sessions
func DefaultSessionsConfiguration() SessionsConfiguration {
	return SessionsConfiguration{
		Cookie:                      DefaultCookieName,
		CookieLength:                DefaultCookieLength,
		DecodeCookie:                false,
		Expires:                     0,
		GcDuration:                  DefaultSessionGcDuration,
		DisableSubdomainPersistence: false,
		DisableAutoGC:               true,
	}
}

// WebsocketConfiguration the config contains options for the Websocket main config field
type WebsocketConfiguration struct {
	// WriteTimeout time allowed to write a message to the connection.
	// Default value is 15 * time.Second
	WriteTimeout time.Duration
	// PongTimeout allowed to read the next pong message from the connection
	// Default value is 60 * time.Second
	PongTimeout time.Duration
	// PingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
	// Default value is (PongTimeout * 9) / 10
	PingPeriod time.Duration
	// MaxMessageSize max message size allowed from connection
	// Default value is 1024
	MaxMessageSize int64
	// BinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
	// see https://github.com/kataras/iris/issues/387#issuecomment-243006022 for more
	// defaults to false
	BinaryMessages bool
	// Endpoint is the path which the websocket server will listen for clients/connections
	// Default value is empty string, if you don't set it the Websocket server is disabled.
	Endpoint string
	// ReadBufferSize is the buffer size for the underline reader
	ReadBufferSize int
	// WriteBufferSize is the buffer size for the underline writer
	WriteBufferSize int
}

var (
	// OptionWebsocketWriteTimeout time allowed to write a message to the connection.
	// Default value is 15 * time.Second
	OptionWebsocketWriteTimeout = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.Websocket.WriteTimeout = val
		}
	}
	// OptionWebsocketPongTimeout allowed to read the next pong message from the connection
	// Default value is 60 * time.Second
	OptionWebsocketPongTimeout = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.Websocket.PongTimeout = val
		}
	}
	// OptionWebsocketPingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
	// Default value is (PongTimeout * 9) / 10
	OptionWebsocketPingPeriod = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.Websocket.PingPeriod = val
		}
	}
	// OptionWebsocketMaxMessageSize max message size allowed from connection
	// Default value is 1024
	OptionWebsocketMaxMessageSize = func(val int64) OptionSet {
		return func(c *Configuration) {
			c.Websocket.MaxMessageSize = val
		}
	}
	// OptionWebsocketBinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
	// see https://github.com/kataras/iris/issues/387#issuecomment-243006022 for more
	// defaults to false
	OptionWebsocketBinaryMessages = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Websocket.BinaryMessages = val
		}
	}
	// OptionWebsocketEndpoint is the path which the websocket server will listen for clients/connections
	// Default value is empty string, if you don't set it the Websocket server is disabled.
	OptionWebsocketEndpoint = func(val string) OptionSet {
		return func(c *Configuration) {
			c.Websocket.Endpoint = val
		}
	}
	// OptionWebsocketReadBufferSize is the buffer size for the underline reader
	OptionWebsocketReadBufferSize = func(val int) OptionSet {
		return func(c *Configuration) {
			c.Websocket.ReadBufferSize = val
		}
	}
	// OptionWebsocketWriteBufferSize is the buffer size for the underline writer
	OptionWebsocketWriteBufferSize = func(val int) OptionSet {
		return func(c *Configuration) {
			c.Websocket.WriteBufferSize = val
		}
	}
)

const (
	// DefaultWriteTimeout 15 * time.Second
	DefaultWriteTimeout = 15 * time.Second
	// DefaultPongTimeout 60 * time.Second
	DefaultPongTimeout = 60 * time.Second
	// DefaultPingPeriod (DefaultPongTimeout * 9) / 10
	DefaultPingPeriod = (DefaultPongTimeout * 9) / 10
	// DefaultMaxMessageSize 1024
	DefaultMaxMessageSize = 1024
)

// DefaultWebsocketConfiguration returns the default config for iris-ws websocket package
func DefaultWebsocketConfiguration() WebsocketConfiguration {
	return WebsocketConfiguration{
		WriteTimeout:    DefaultWriteTimeout,
		PongTimeout:     DefaultPongTimeout,
		PingPeriod:      DefaultPingPeriod,
		MaxMessageSize:  DefaultMaxMessageSize,
		BinaryMessages:  false,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		Endpoint:        "",
	}
}

// TesterConfiguration configuration used inside main config field 'Tester'
type TesterConfiguration struct {
	// ListeningAddr is the virtual server's listening addr (host)
	// Default is "iris-go.com:1993"
	ListeningAddr string
	// ExplicitURL If true then the url (should) be prepended manually, useful when want to test subdomains
	// Default is false
	ExplicitURL bool
	// Debug if true then debug messages from the httpexpect will be shown when a test runs
	// Default is false
	Debug bool
}

var (
	// OptionTesterListeningAddr is the virtual server's listening addr (host)
	// Default is "iris-go.com:1993"
	OptionTesterListeningAddr = func(val string) OptionSet {
		return func(c *Configuration) {
			c.Tester.ListeningAddr = val
		}
	}
	// OptionTesterExplicitURL If true then the url (should) be prepended manually, useful when want to test subdomains
	// Default is false
	OptionTesterExplicitURL = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Tester.ExplicitURL = val
		}
	}
	// OptionTesterDebug if true then debug messages from the httpexpect will be shown when a test runs
	// Default is false
	OptionTesterDebug = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Tester.Debug = val
		}
	}
)

// DefaultTesterConfiguration returns the default configuration for a tester
// the ListeningAddr is used as virtual only when no running server is founded
func DefaultTesterConfiguration() TesterConfiguration {
	return TesterConfiguration{ListeningAddr: "iris-go.com:1993", ExplicitURL: false, Debug: false}
}

// ServerConfiguration is the configuration which is used inside iris' server(s) for listening to
type ServerConfiguration struct {
	// ListenningAddr the addr that server listens to
	ListeningAddr string
	CertFile      string
	KeyFile       string
	// AutoTLS enable to get certifications from the Letsencrypt
	// when this configuration field is true, the CertFile & KeyFile are empty, no need to provide a key.
	//
	// example: https://github.com/iris-contrib/examples/blob/master/letsencyrpt/main.go
	AutoTLS bool
	// Mode this is for unix only
	Mode os.FileMode
	// MaxRequestBodySize Maximum request body size.
	//
	// The server rejects requests with bodies exceeding this limit.
	//
	// By default request body size is 8MB.
	MaxRequestBodySize int

	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is used if not set.
	ReadBufferSize int

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is used if not set.
	WriteBufferSize int

	// Maximum duration for reading the full request (including body).
	//
	// This also limits the maximum duration for idle keep-alive
	// connections.
	//
	// By default request read timeout is unlimited.
	ReadTimeout time.Duration

	// Maximum duration for writing the full response (including body).
	//
	// By default response write timeout is unlimited.
	WriteTimeout time.Duration

	// RedirectTo, defaults to empty, set it in order to override the station's handler and redirect all requests to this address which is of form(HOST:PORT or :PORT)
	//
	// NOTE: the http status is 'StatusMovedPermanently', means one-time-redirect(the browser remembers the new addr and goes to the new address without need to request something from this server
	// which means that if you want to change this address you have to clear your browser's cache in order this to be able to change to the new addr.
	//
	// example: https://github.com/iris-contrib/examples/tree/master/multiserver_listening2
	RedirectTo string
	// Virtual If this server is not really listens to a real host, it mostly used in order to achieve testing without system modifications
	Virtual bool
	// VListeningAddr, can be used for both virtual = true or false,
	// if it's setted to not empty, then the server's Host() will return this addr instead of the ListeningAddr.
	// server's Host() is used inside global template helper funcs
	// set it when you are sure you know what it does.
	//
	// Default is empty ""
	VListeningAddr string
	// VScheme if setted to not empty value then all template's helper funcs prepends that as the url scheme instead of the real scheme
	// server's .Scheme returns VScheme if  not empty && differs from real scheme
	//
	// Default is empty ""
	VScheme string
	// Name the server's name, defaults to "iris".
	// You're free to change it, but I will trust you to don't, this is the only setting whose somebody, like me, can see if iris web framework is used
	Name string
}

// note: ServerConfiguration is the only one config which has its own option setter because
// it's independent from a specific iris instance:
// same server can run on multi iris instance
// one iris instance/station can have and listening to more than one server.

// OptionServerSettter server configuration option setter
type OptionServerSettter interface {
	Set(c *ServerConfiguration)
}

// OptionServerSet is the func which implements the OptionServerSettter, this is used widely
type OptionServerSet func(c *ServerConfiguration)

// Set is the func which makes OptionServerSet implements the OptionServerSettter
func (o OptionServerSet) Set(c *ServerConfiguration) {
	o(c)
}

// Set implements the OptionServerSettter to the ServerConfiguration
func (c ServerConfiguration) Set(main *ServerConfiguration) {
	mergo.MergeWithOverwrite(main, c)
}

// Options for ServerConfiguration
var (
	OptionServerListeningAddr = func(val string) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.ListeningAddr = val
		}
	}

	OptionServerCertFile = func(val string) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.CertFile = val
		}
	}

	OptionServerKeyFile = func(val string) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.KeyFile = val
		}
	}

	// AutoTLS enable to get certifications from the Letsencrypt
	// when this configuration field is true, the CertFile & KeyFile are empty, no need to provide a key.
	//
	// example: https://github.com/iris-contrib/examples/blob/master/letsencyrpt/main.go
	OptionServerAutoTLS = func(val bool) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.AutoTLS = val
		}
	}

	// Mode this is for unix only
	OptionServerMode = func(val os.FileMode) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.Mode = val
		}
	}

	// OptionServerMaxRequestBodySize Maximum request body size.
	//
	// The server rejects requests with bodies exceeding this limit.
	//
	// By default request body size is 8MB.
	OptionServerMaxRequestBodySize = func(val int) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.MaxRequestBodySize = val
		}
	}

	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is used if not set.
	OptionServerReadBufferSize = func(val int) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.ReadBufferSize = val
		}
	}

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is used if not set.
	OptionServerWriteBufferSize = func(val int) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.WriteBufferSize = val
		}
	}

	// Maximum duration for reading the full request (including body).
	//
	// This also limits the maximum duration for idle keep-alive
	// connections.
	//
	// By default request read timeout is unlimited.
	OptionServerReadTimeout = func(val time.Duration) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.ReadTimeout = val
		}
	}

	// Maximum duration for writing the full response (including body).
	//
	// By default response write timeout is unlimited.
	OptionServerWriteTimeout = func(val time.Duration) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.WriteTimeout = val
		}
	}

	// RedirectTo, defaults to empty, set it in order to override the station's handler and redirect all requests to this address which is of form(HOST:PORT or :PORT)
	//
	// NOTE: the http status is 'StatusMovedPermanently', means one-time-redirect(the browser remembers the new addr and goes to the new address without need to request something from this server
	// which means that if you want to change this address you have to clear your browser's cache in order this to be able to change to the new addr.
	//
	// example: https://github.com/iris-contrib/examples/tree/master/multiserver_listening2
	OptionServerRedirectTo = func(val string) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.RedirectTo = val
		}
	}

	// OptionServerVirtual If this server is not really listens to a real host, it mostly used in order to achieve testing without system modifications
	OptionServerVirtual = func(val bool) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.Virtual = val
		}
	}
	// OptionServerVListeningAddr, can be used for both virtual = true or false,
	// if it's setted to not empty, then the server's Host() will return this addr instead of the ListeningAddr.
	// server's Host() is used inside global template helper funcs
	// set it when you are sure you know what it does.
	//
	// Default is empty ""
	OptionServerVListeningAddr = func(val string) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.VListeningAddr = val
		}
	}

	// OptionServerVScheme if setted to not empty value then all template's helper funcs prepends that as the url scheme instead of the real scheme
	// server's .Scheme returns VScheme if  not empty && differs from real scheme
	//
	// Default is empty ""
	OptionServerVScheme = func(val string) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.VScheme = val
		}
	}

	// OptionServerName the server's name, defaults to "iris".
	// You're free to change it, but I will trust you to don't, this is the only setting whose somebody, like me, can see if iris web framework is used
	OptionServerName = func(val string) OptionServerSet {
		return func(c *ServerConfiguration) {
			c.ListeningAddr = val
		}
	}
)

// ServerParseAddr parses the listening addr and returns this
func ServerParseAddr(listeningAddr string) string {
	// check if addr has :port, if not do it +:80 ,we need the hostname for many cases
	a := listeningAddr
	if a == "" {
		// check for os environments
		if oshost := os.Getenv("HOST"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("ADDR"); oshost != "" {
			a = oshost
		} else if osport := os.Getenv("PORT"); osport != "" {
			a = ":" + osport
		}

		if a == "" {
			a = DefaultServerAddr
		}

	}
	if portIdx := strings.IndexByte(a, ':'); portIdx == 0 {
		// if contains only :port	,then the : is the first letter, so we dont have setted a hostname, lets set it
		a = DefaultServerHostname + a
	}
	if portIdx := strings.IndexByte(a, ':'); portIdx < 0 {
		// missing port part, add it
		a = a + ":80"
	}

	return a
}

// Default values for base Server conf
const (
	// DefaultServerHostname returns the default hostname which is 0.0.0.0
	DefaultServerHostname = "0.0.0.0"
	// DefaultServerPort returns the default port which is 8080
	DefaultServerPort = 8080
	// DefaultMaxRequestBodySize is 8MB
	DefaultMaxRequestBodySize = 2 * fasthttp.DefaultMaxRequestBodySize

	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is 8MB
	DefaultReadBufferSize = 8096

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is 8MB
	DefaultWriteBufferSize = 8096

	// DefaultServerName the response header of the 'Server' value when writes to the client
	DefaultServerName = "iris"
)

var (
	// DefaultServerAddr the default server addr which is: 0.0.0.0:8080
	DefaultServerAddr = DefaultServerHostname + ":" + strconv.Itoa(DefaultServerPort)
)

// DefaultServerConfiguration returns the default configs for the server
func DefaultServerConfiguration() ServerConfiguration {
	return ServerConfiguration{
		ListeningAddr:      DefaultServerAddr,
		Name:               DefaultServerName,
		MaxRequestBodySize: DefaultMaxRequestBodySize,
		ReadBufferSize:     DefaultReadBufferSize,
		WriteBufferSize:    DefaultWriteBufferSize,
		RedirectTo:         "",
		Virtual:            false,
		VListeningAddr:     "",
		VScheme:            "",
	}
}
