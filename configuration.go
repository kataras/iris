package iris

import (
	"io"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/imdario/mergo"
	"github.com/valyala/fasthttp"
	"gopkg.in/kataras/go-options.v0"
	"gopkg.in/kataras/go-sessions.v0"
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
// Configuration is also implements the OptionSet so it's a valid option itself, this is brilliant enough
type Configuration struct {
	// VHost is the addr or the domain that server listens to, which it's optional
	// When to set VHost manually:
	// 1. it's automatically setted when you're calling
	//     $instance.Listen/ListenUNIX/ListenTLS/ListenLETSENCRYPT functions or
	//     ln,_ := iris.TCP4/UNIX/TLS/LETSENCRYPT; $instance.Serve(ln)
	// 2. If you using a balancer, or something like nginx
	//    then set it in order to have the correct url
	//    when calling the template helper '{{url }}'
	//    *keep note that you can use {{urlpath }}) instead*
	//
	// Note: this is the main's server Host, you can setup unlimited number of fasthttp servers
	// listening to the $instance.Handler after the manually-called $instance.Build
	//
	// Default comes from iris.Listen/.Serve with iris' listeners (iris.TCP4/UNIX/TLS/LETSENCRYPT)
	VHost string

	// VScheme is the scheme (http:// or https://) putted at the template function '{{url }}'
	// It's an optional field,
	// When to set VScheme manually:
	// 1. You didn't start the main server using $instance.Listen/ListenTLS/ListenLETSENCRYPT or $instance.Serve($instance.TCP4()/.TLS...)
	// 2. if you're using something like nginx and have iris listening with addr only(http://) but the nginx mapper is listening to https://
	//
	// Default comes from iris.Listen/.Serve with iris' listeners (TCP4,UNIX,TLS,LETSENCRYPT)
	VScheme string

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

	// Maximum number of concurrent client connections allowed per IP.
	//
	// By default unlimited number of concurrent connections
	MaxConnsPerIP int

	// Maximum number of requests served per connection.
	//
	// The server closes connection after the last request.
	// 'Connection: close' header is added to the last response.
	//
	// By default unlimited number of requests may be served per connection.
	MaxRequestsPerConn int

	// CheckForUpdates will try to search for newer version of Iris based on the https://github.com/kataras/iris/releases
	// If a newer version found then the app will ask the he dev/user if want to update the 'x' version
	// if 'y' is pressed then the updater will try to install the latest version
	// the updater, will notify the dev/user that the update is finished and should restart the App manually.
	// Notes:
	// 1. Experimental feature
	// 2. If setted to true, the app will start the server normally and runs the updater in its own goroutine,
	//    for a sync operation see CheckForUpdatesSync.
	// 3. If you as developer edited the $GOPATH/src/github/kataras or any other Iris' Go dependencies at the past
	//    then the update process will fail.
	//
	// Usage: iris.Set(iris.OptionCheckForUpdates(true)) or
	//        iris.Config.CheckForUpdates = true or
	//        app := iris.New(iris.OptionCheckForUpdates(true))
	// Default is false
	CheckForUpdates bool
	// CheckForUpdatesSync checks for updates before server starts, it will have a little delay depends on the machine's download's speed
	// See CheckForUpdates for more
	// Notes:
	// 1. you could use the CheckForUpdatesSync while CheckForUpdates is false, set this or CheckForUpdates to true not both
	// 2. if both CheckForUpdates and CheckForUpdatesSync are setted to true then the updater will run in sync mode, before server server starts.
	//
	// Default is false
	CheckForUpdatesSync bool

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

	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405) and fires the 405 error instead of 404
	// Default is false
	FireMethodNotAllowed bool

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

	// DisableTemplateEngines set to true to disable loading the default template engine (html/template) and disallow the use of iris.UseEngine
	// Defaults to false
	DisableTemplateEngines bool

	// IsDevelopment iris will act like a developer, for example
	// If true then re-builds the templates on each request
	// Defaults to false
	IsDevelopment bool

	// TimeFormat time format for any kind of datetime parsing
	TimeFormat string

	// Charset character encoding for various rendering
	// used for templates and the rest of the responses
	// Defaults to "UTF-8"
	Charset string

	// Gzip enables gzip compression on your Render actions, this includes any type of render, templates and pure/raw content
	// If you don't want to enable it globaly, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
	// Defaults to false
	Gzip bool

	// Sessions contains the configs for sessions
	Sessions SessionsConfiguration

	// Websocket contains the configs for Websocket's server integration
	Websocket WebsocketConfiguration

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

	// OptionVHost is the addr or the domain that server listens to, which it's optional
	// When to set VHost manually:
	// 1. it's automatically setted when you're calling
	//     $instance.Listen/ListenUNIX/ListenTLS/ListenLETSENCRYPT functions or
	//     ln,_ := iris.TCP4/UNIX/TLS/LETSENCRYPT; $instance.Serve(ln)
	// 2. If you using a balancer, or something like nginx
	//    then set it in order to have the correct url
	//    when calling the template helper '{{url }}'
	//    *keep note that you can use {{urlpath }}) instead*
	//
	// Note: this is the main's server Host, you can setup unlimited number of fasthttp servers
	// listening to the $instance.Handler after the manually-called $instance.Build
	//
	// Default comes from iris.Listen/.Serve with iris' listeners (iris.TCP4/UNIX/TLS/LETSENCRYPT)
	OptionVHost = func(val string) OptionSet {
		return func(c *Configuration) {
			c.VHost = val
		}
	}

	// OptionVScheme is the scheme (http:// or https://) putted at the template function '{{url }}'
	// It's an optional field,
	// When to set Scheme manually:
	// 1. You didn't start the main server using $instance.Listen/ListenTLS/ListenLETSENCRYPT or $instance.Serve($instance.TCP4()/.TLS...)
	// 2. if you're using something like nginx and have iris listening with addr only(http://) but the nginx mapper is listening to https://
	//
	// Default comes from iris.Listen/.Serve with iris' listeners (TCP4,UNIX,TLS,LETSENCRYPT)
	OptionVScheme = func(val string) OptionSet {
		return func(c *Configuration) {
			c.VScheme = val
		}
	}

	// OptionMaxRequestBodySize Maximum request body size.
	//
	// The server rejects requests with bodies exceeding this limit.
	//
	// By default request body size is 8MB.
	OptionMaxRequestBodySize = func(val int) OptionSet {
		return func(c *Configuration) {
			c.MaxRequestBodySize = val
		}
	}

	// Per-connection buffer size for requests' reading.``
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is used if not set.
	OptionReadBufferSize = func(val int) OptionSet {
		return func(c *Configuration) {
			c.ReadBufferSize = val
		}
	}

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is used if not set.
	OptionWriteBufferSize = func(val int) OptionSet {
		return func(c *Configuration) {
			c.WriteBufferSize = val
		}
	}

	// Maximum duration for reading the full request (including body).
	//
	// This also limits the maximum duration for idle keep-alive
	// connections.
	//
	// By default request read timeout is unlimited.
	OptionReadTimeout = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.ReadTimeout = val
		}
	}

	// Maximum duration for writing the full response (including body).
	//
	// By default response write timeout is unlimited.
	OptionWriteTimeout = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.WriteTimeout = val
		}
	}

	// OptionMaxConnsPerIP Maximum number of concurrent client connections allowed per IP.
	//
	// By default unlimited number of concurrent connections
	// may be established to the server from a single IP address.
	OptionMaxConnsPerIP = func(val int) OptionSet {
		return func(c *Configuration) {
			c.MaxConnsPerIP = val
		}
	}

	// OptionMaxRequestsPerConn Maximum number of requests served per connection.
	//
	// The server closes connection after the last request.
	// 'Connection: close' header is added to the last response.
	//
	// By default unlimited number of requests may be served per connection.
	OptionMaxRequestsPerConn = func(val int) OptionSet {
		return func(c *Configuration) {
			c.MaxRequestsPerConn = val
		}
	}

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
	// CheckForUpdatesSync checks for updates before server starts, it will have a little delay depends on the machine's download's speed
	// See CheckForUpdates for more
	// Notes:
	// 1. you could use the CheckForUpdatesSync while CheckForUpdates is false, set this or CheckForUpdates to true not both
	// 2. if both CheckForUpdates and CheckForUpdatesSync are setted to true then the updater will run in sync mode, before server server starts.
	//
	// Default is false
	OptionCheckForUpdatesSync = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.CheckForUpdatesSync = val
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

	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405) and fires the 405 error instead of 404
	// Default is false
	OptionFireMethodNotAllowed = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.FireMethodNotAllowed = val
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
)

var (
	// DefaultLoggerOut is the default logger's output
	DefaultLoggerOut = os.Stdout
	// DefaultServerName the response header of the 'Server' value when writes to the client
	DefaultServerName = ""
)

// DefaultConfiguration returns the default configuration for an Iris station, fills the main Configuration
func DefaultConfiguration() Configuration {
	return Configuration{
		VHost:                  "",
		VScheme:                "",
		MaxRequestBodySize:     DefaultMaxRequestBodySize,
		ReadBufferSize:         DefaultReadBufferSize,
		WriteBufferSize:        DefaultWriteBufferSize,
		MaxConnsPerIP:          0,
		MaxRequestsPerConn:     0,
		CheckForUpdates:        false,
		CheckForUpdatesSync:    false,
		DisablePathCorrection:  DefaultDisablePathCorrection,
		DisablePathEscape:      DefaultDisablePathEscape,
		FireMethodNotAllowed:   false,
		DisableBanner:          false,
		LoggerOut:              DefaultLoggerOut,
		LoggerPreffix:          DefaultLoggerPreffix,
		DisableTemplateEngines: false,
		IsDevelopment:          false,
		TimeFormat:             DefaultTimeFormat,
		Charset:                DefaultCharset,
		Gzip:                   false,
		Sessions:               DefaultSessionsConfiguration(),
		Websocket:              DefaultWebsocketConfiguration(),
		Other:                  options.Options{},
	}
}

// SessionsConfiguration the configuration for sessions
// has 6 fields
// first is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
// second enable if you want to decode the cookie's key also
// third is the time which the client's cookie expires
// forth is the cookie length (sessionid) int, Defaults to 32, do not change if you don't have any reason to do
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
	// Defaults to false
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
	// Defaults to false
	BinaryMessages bool
	// Endpoint is the path which the websocket server will listen for clients/connections
	// Default value is empty string, if you don't set it the Websocket server is disabled.
	Endpoint string
	// ReadBufferSize is the buffer size for the underline reader
	ReadBufferSize int
	// WriteBufferSize is the buffer size for the underline writer
	WriteBufferSize int
	// Headers  if true then the client's headers are copy to the websocket connection
	//
	// Default is true
	Headers bool
	// Error specifies the function for generating HTTP error responses.
	//
	// The default behavior is to store the reason in the context (ctx.Set(reason)) and fire any custom error (ctx.EmitError(status))
	Error func(ctx *Context, status int, reason string)
	// CheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, the host in the Origin header must not be set or
	// must match the host of the request.
	//
	// The default behavior is to allow all origins
	// you can change this behavior by setting the iris.Config.Websocket.CheckOrigin = iris.WebsocketCheckSameOrigin
	CheckOrigin func(ctx *Context) bool
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
	// Defaults to false
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
	// OptionWebsocketHeaders  if true then the client's headers are copy to the websocket connection
	OptionWebsocketHeaders = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Websocket.Headers = val
		}
	}
	// OptionWebsocketError specifies the function for generating HTTP error responses.
	OptionWebsocketError = func(val func(*Context, int, string)) OptionSet {
		return func(c *Configuration) {
			c.Websocket.Error = val
		}
	}
	// OptionWebsocketCheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, the host in the Origin header must not be set or
	// must match the host of the request.
	OptionWebsocketCheckOrigin = func(val func(*Context) bool) OptionSet {
		return func(c *Configuration) {
			c.Websocket.CheckOrigin = val
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

var (
	// DefaultWebsocketError is the default method to manage the handshake websocket errors
	DefaultWebsocketError = func(ctx *Context, status int, reason string) {
		ctx.Set("WsError", reason)
		ctx.EmitError(status)
	}
	// DefaultWebsocketCheckOrigin is the default method to allow websocket clients to connect to this server
	// you can change this behavior by setting the iris.Config.Websocket.CheckOrigin = iris.WebsocketCheckSameOrigin
	DefaultWebsocketCheckOrigin = func(ctx *Context) bool {
		return true
	}
	// WebsocketCheckSameOrigin returns true if the origin is not set or is equal to the request host
	WebsocketCheckSameOrigin = func(ctx *Context) bool {
		origin := ctx.RequestHeader("origin")
		if len(origin) == 0 {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return u.Host == ctx.HostString()
	}
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
		Headers:         true,
	}
}

// Default values for base Server conf
const (
	// DefaultServerHostname returns the default hostname which is 0.0.0.0
	DefaultServerHostname = "0.0.0.0"
	// DefaultServerPort returns the default port which is 8080, not used
	DefaultServerPort = 8080
)

var (
	// DefaultServerAddr the default server addr which is: 0.0.0.0:8080
	DefaultServerAddr = DefaultServerHostname + ":" + strconv.Itoa(DefaultServerPort)
)
