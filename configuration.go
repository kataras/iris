package iris

import (
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/imdario/mergo"
	"github.com/kataras/go-options"
)

type (
	// OptionSetter sets a configuration field to the main configuration
	// used to help developers to write less and configure only what they really want and nothing else
	// example:
	// iris.New(iris.Configuration{Charset: "UTF-8", Gzip:true})
	// now can be done also by using iris.Option$FIELD:
	// iris.New(iris.OptionCharset("UTF-8"), iris.OptionGzip(true))
	// benefits:
	// 1. dev has no worries what option to pass,
	//    he/she can just press iris.Option and all options should be shown to her/his editor's autocomplete-popup window
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

// Configuration the whole configuration for an iris instance ($instance.Config) or global iris instance (iris.Default.Config)
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
	// Note: this is the main's server Host, you can setup unlimited number of net/http servers
	// listening to the $instance.Handler after the manually-called $instance.Build
	//
	// Default comes from iris.Default.Listen/.Serve with iris' listeners (iris.TCP4/UNIX/TLS/LETSENCRYPT)
	VHost string

	// VScheme is the scheme (http:// or https://) putted at the template function '{{url }}'
	// It's an optional field,
	// When to set VScheme manually:
	// 1. You didn't start the main server using $instance.Listen/ListenTLS/ListenLETSENCRYPT or $instance.Serve($instance.TCP4()/.TLS...)
	// 2. if you're using something like nginx and have iris listening with addr only(http://) but the nginx mapper is listening to https://
	//
	// Default comes from iris.Default.Listen/.Serve with iris' listeners (TCP4,UNIX,TLS,LETSENCRYPT)
	VScheme string

	ReadTimeout  time.Duration // maximum duration before timing out read of the request
	WriteTimeout time.Duration // maximum duration before timing out write of the response

	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and
	// values, including the request line. It does not limit the
	// size of the request body.
	// If zero, DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int

	// TLSNextProto optionally specifies a function to take over
	// ownership of the provided TLS connection when an NPN/ALPN
	// protocol upgrade has occurred. The map key is the protocol
	// name negotiated. The Handler argument should be used to
	// handle HTTP requests and will initialize the Request's TLS
	// and RemoteAddr if not already set. The connection is
	// automatically closed when the function returns.
	// If TLSNextProto is nil, HTTP/2 support is enabled automatically.
	TLSNextProto map[string]func(*http.Server, *tls.Conn, http.Handler)

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state. See the
	// ConnState type and associated constants for details.
	ConnState func(net.Conn, http.ConnState)

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
	// Usage: iris.Default.Set(iris.OptionCheckForUpdates(true)) or
	//        iris.Default.Config.CheckForUpdates = true or
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

	// DisablePathCorrection corrects and redirects the requested path to the registered path
	// for example, if /home/ path is requested but no handler for this Route found,
	// then the Router checks if /home handler exists, if yes,
	// (permant)redirects the client to the correct path /home
	//
	// Default is false
	DisablePathCorrection bool

	// EnablePathEscape when is true then its escapes the path, the named parameters (if any).
	// Change to false it if you want something like this https://github.com/kataras/iris/issues/135 to work
	//
	// When do you need to Disable(false) it:
	// accepts parameters with slash '/'
	// Request: http://localhost:8080/details/Project%2FDelta
	// ctx.Param("project") returns the raw named parameter: Project%2FDelta
	// which you can escape it manually with net/url:
	// projectName, _ := url.QueryUnescape(c.Param("project").
	// Look here: https://github.com/kataras/iris/issues/135 for more
	//
	// Default is false
	EnablePathEscape bool

	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405) and fires the 405 error instead of 404
	// Default is false
	FireMethodNotAllowed bool

	// DisableBanner outputs the iris banner at startup
	//
	// Default is false
	DisableBanner bool

	// DisableBodyConsumptionOnUnmarshal manages the reading behavior of the context's body readers/binders.
	// If setted to true then it
	// disables the body consumption by the `context.UnmarshalBody/ReadJSON/ReadXML`.
	//
	// By-default io.ReadAll` is used to read the body from the `context.Request.Body which is an `io.ReadCloser`,
	// if this field setted to true then a new buffer will be created to read from and the request body.
	// The body will not be changed and existing data before the context.UnmarshalBody/ReadJSON/ReadXML will be not consumed.
	DisableBodyConsumptionOnUnmarshal bool

	// DisableTemplateEngines set to true to disable loading the default template engine (html/template) and disallow the use of iris.Default.UseEngine
	// Defaults to false
	DisableTemplateEngines bool

	// TimeFormat time format for any kind of datetime parsing
	TimeFormat string

	// Charset character encoding for various rendering
	// used for templates and the rest of the responses
	// Defaults to "UTF-8"
	Charset string

	// Gzip enables gzip compression on your Render actions, this includes any type of render, templates and pure/raw content
	// If you don't want to enable it globally, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
	// Defaults to false
	Gzip bool

	// Other are the custom, dynamic options, can be empty
	// this fill used only by you to set any app's options you want
	// for each of an Iris instance
	Other options.Options
}

// Set implements the OptionSetter
func (c Configuration) Set(main *Configuration) {
	// ignore error
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
	// Note: this is the main's server Host, you can setup unlimited number of net/http servers
	// listening to the $instance.Handler after the manually-called $instance.Build
	//
	// Default comes from iris.Default.Listen/.Serve with iris' listeners (iris.TCP4/UNIX/TLS/LETSENCRYPT)
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
	// Default comes from iris.Default.Listen/.Serve with iris' listeners (TCP4,UNIX,TLS,LETSENCRYPT)
	OptionVScheme = func(val string) OptionSet {
		return func(c *Configuration) {
			c.VScheme = val
		}
	}
	// maximum duration before timing out read of the request
	OptionReadTimeout = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.ReadTimeout = val
		}
	}
	// maximum duration before timing out write of the response
	OptionWriteTimeout = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.WriteTimeout = val
		}
	}

	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and
	// values, including the request line. It does not limit the
	// size of the request body.
	// If zero, DefaultMaxHeaderBytes(8MB) is used.
	OptionMaxHeaderBytes = func(val int) OptionSet {
		return func(c *Configuration) {
			c.MaxHeaderBytes = val
		}
	}

	// TLSNextProto optionally specifies a function to take over
	// ownership of the provided TLS connection when an NPN/ALPN
	// protocol upgrade has occurred. The map key is the protocol
	// name negotiated. The Handler argument should be used to
	// handle HTTP requests and will initialize the Request's TLS
	// and RemoteAddr if not already set. The connection is
	// automatically closed when the function returns.
	// If TLSNextProto is nil, HTTP/2 support is enabled automatically.
	OptionTLSNextProto = func(val map[string]func(*http.Server, *tls.Conn, http.Handler)) OptionSet {
		return func(c *Configuration) {
			c.TLSNextProto = val
		}
	}

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state. See the
	// ConnState type and associated constants for details.
	OptionConnState = func(val func(net.Conn, http.ConnState)) OptionSet {
		return func(c *Configuration) {
			c.ConnState = val
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
	// Usage: iris.Default.Set(iris.OptionCheckForUpdates(true)) or
	//        iris.Default.Config.CheckForUpdates = true or
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

	// OptionDisablePathCorrection corrects and redirects the requested path to the registered path
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

	// OptionEnablePathEscape when is true then its escapes the path, the named path parameters (if any).
	OptionEnablePathEscape = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.EnablePathEscape = val
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

	// OptionDisableBodyConsumptionOnUnmarshal manages the reading behavior of the context's body readers/binders.
	// If setted to true then it
	// disables the body consumption by the `context.UnmarshalBody/ReadJSON/ReadXML`.
	//
	// By-default io.ReadAll` is used to read the body from the `context.Request.Body which is an `io.ReadCloser`,
	// if this field setted to true then a new buffer will be created to read from and the request body.
	// The body will not be changed and existing data before the context.UnmarshalBody/ReadJSON/ReadXML will be not consumed.
	OptionDisableBodyConsumptionOnUnmarshal = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.DisableBodyConsumptionOnUnmarshal = val
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
	// If you don't want to enable it globally, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
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
)

// Default values for base Iris conf
const (
	DefaultDisablePathCorrection = false
	DefaultEnablePathEscape      = false
	DefaultCharset               = "UTF-8"
	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is 8MB
	DefaultMaxHeaderBytes = 8096

	// DefaultReadTimeout no read client timeout
	DefaultReadTimeout = 0
	// DefaultWriteTimeout no serve client timeout
	DefaultWriteTimeout = 0
)

// DefaultConfiguration returns the default configuration for an Iris station, fills the main Configuration
func DefaultConfiguration() Configuration {
	return Configuration{
		VHost:                             "",
		VScheme:                           "",
		ReadTimeout:                       DefaultReadTimeout,
		WriteTimeout:                      DefaultWriteTimeout,
		MaxHeaderBytes:                    DefaultMaxHeaderBytes,
		CheckForUpdates:                   false,
		CheckForUpdatesSync:               false,
		DisablePathCorrection:             DefaultDisablePathCorrection,
		EnablePathEscape:                  DefaultEnablePathEscape,
		FireMethodNotAllowed:              false,
		DisableBanner:                     false,
		DisableBodyConsumptionOnUnmarshal: false,
		TimeFormat:                        DefaultTimeFormat,
		Charset:                           DefaultCharset,
		Gzip:                              false,
		Other:                             options.Options{},
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
