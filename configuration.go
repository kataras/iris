package iris

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/netutil"

	"github.com/BurntSushi/toml"
	"github.com/kataras/golog"
	"github.com/kataras/sitemap"
	"github.com/kataras/tunnel"
	"gopkg.in/yaml.v3"
)

const globalConfigurationKeyword = "~"

// homeConfigurationFilename returns the physical location of the global configuration(yaml or toml) file.
// This is useful when we run multiple iris servers that share the same
// configuration, even with custom values at its "Other" field.
// It will return a file location
// which targets to $HOME or %HOMEDRIVE%+%HOMEPATH% + "iris" + the given "ext".
func homeConfigurationFilename(ext string) string {
	return filepath.Join(homeDir(), "iris"+ext)
}

func homeDir() (home string) {
	u, err := user.Current()
	if u != nil && err == nil {
		home = u.HomeDir
	}

	if home == "" {
		home = os.Getenv("HOME")
	}

	if home == "" {
		if runtime.GOOS == "plan9" {
			home = os.Getenv("home")
		} else if runtime.GOOS == "windows" {
			home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
		}
	}

	return
}

func parseYAML(filename string) (Configuration, error) {
	c := DefaultConfiguration()
	// get the abs
	// which will try to find the 'filename' from current workind dir too.
	yamlAbsPath, err := filepath.Abs(filename)
	if err != nil {
		return c, fmt.Errorf("parse yaml: %w", err)
	}

	// read the raw contents of the file
	data, err := os.ReadFile(yamlAbsPath)
	if err != nil {
		return c, fmt.Errorf("parse yaml: %w", err)
	}

	// put the file's contents as yaml to the default configuration(c)
	if err := yaml.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("parse yaml: %w", err)
	}
	return c, nil
}

// YAML reads Configuration from a configuration.yml file.
//
// Accepts the absolute path of the cfg.yml.
// An error will be shown to the user via panic with the error message.
// Error may occur when the cfg.yml does not exist or is not formatted correctly.
//
// Note: if the char '~' passed as "filename" then it tries to load and return
// the configuration from the $home_directory + iris.yml,
// see `WithGlobalConfiguration` for more information.
//
// Usage:
// app.Configure(iris.WithConfiguration(iris.YAML("myconfig.yml"))) or
// app.Run([iris.Runner], iris.WithConfiguration(iris.YAML("myconfig.yml"))).
func YAML(filename string) Configuration {
	// check for globe configuration file and use that, otherwise
	// return the default configuration if file doesn't exist.
	if filename == globalConfigurationKeyword {
		filename = homeConfigurationFilename(".yml")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			panic("default configuration file '" + filename + "' does not exist")
		}
	}

	c, err := parseYAML(filename)
	if err != nil {
		panic(err)
	}

	return c
}

// TOML reads Configuration from a toml-compatible document file.
// Read more about toml's implementation at:
// https://github.com/toml-lang/toml
//
// Accepts the absolute path of the configuration file.
// An error will be shown to the user via panic with the error message.
// Error may occur when the file does not exist or is not formatted correctly.
//
// Note: if the char '~' passed as "filename" then it tries to load and return
// the configuration from the $home_directory + iris.tml,
// see `WithGlobalConfiguration` for more information.
//
// Usage:
// app.Configure(iris.WithConfiguration(iris.TOML("myconfig.tml"))) or
// app.Run([iris.Runner], iris.WithConfiguration(iris.TOML("myconfig.tml"))).
func TOML(filename string) Configuration {
	c := DefaultConfiguration()

	// check for globe configuration file and use that, otherwise
	// return the default configuration if file doesn't exist.
	if filename == globalConfigurationKeyword {
		filename = homeConfigurationFilename(".tml")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			panic("default configuration file '" + filename + "' does not exist")
		}
	}

	// get the abs
	// which will try to find the 'filename' from current workind dir too.
	tomlAbsPath, err := filepath.Abs(filename)
	if err != nil {
		panic(fmt.Errorf("toml: %w", err))
	}

	// read the raw contents of the file
	data, err := os.ReadFile(tomlAbsPath)
	if err != nil {
		panic(fmt.Errorf("toml :%w", err))
	}

	// put the file's contents as toml to the default configuration(c)
	if _, err := toml.Decode(string(data), &c); err != nil {
		panic(fmt.Errorf("toml :%w", err))
	}
	// Author's notes:
	// The toml's 'usual thing' for key naming is: the_config_key instead of TheConfigKey
	// but I am always prefer to use the specific programming language's syntax
	// and the original configuration name fields for external configuration files
	// so we do 'toml: "TheConfigKeySameAsTheConfigField" instead.
	return c
}

// Configurator is just an interface which accepts the framework instance.
//
// It can be used to register a custom configuration with `Configure` in order
// to modify the framework instance.
//
// Currently Configurator is being used to describe the configuration's fields values.
type Configurator func(*Application)

// WithGlobalConfiguration will load the global yaml configuration file
// from the home directory and it will set/override the whole app's configuration
// to that file's contents. The global configuration file can be modified by user
// and be used by multiple iris instances.
//
// This is useful when we run multiple iris servers that share the same
// configuration, even with custom values at its "Other" field.
//
// Usage: `app.Configure(iris.WithGlobalConfiguration)` or `app.Run([iris.Runner], iris.WithGlobalConfiguration)`.
var WithGlobalConfiguration = func(app *Application) {
	app.Configure(WithConfiguration(YAML(globalConfigurationKeyword)))
}

// WithLogLevel sets the `Configuration.LogLevel` field.
func WithLogLevel(level string) Configurator {
	return func(app *Application) {
		if app.logger == nil {
			app.logger = golog.Default
		}
		app.logger.SetLevel(level) // can be fired through app.Configure.

		app.config.LogLevel = level
	}
}

// WithSocketSharding sets the `Configuration.SocketSharding` field to true.
func WithSocketSharding(app *Application) {
	// Note(@kataras): It could be a host Configurator but it's an application setting in order
	// to configure it through yaml/toml files as well.
	app.config.SocketSharding = true
}

// WithKeepAlive sets the `Configuration.KeepAlive` field to the given duration.
func WithKeepAlive(keepAliveDur time.Duration) Configurator {
	return func(app *Application) {
		app.config.KeepAlive = keepAliveDur
	}
}

// WithTimeout sets the `Configuration.Timeout` field to the given duration.
func WithTimeout(timeoutDur time.Duration, htmlBody ...string) Configurator {
	return func(app *Application) {
		app.config.Timeout = timeoutDur
		if len(htmlBody) > 0 {
			app.config.TimeoutMessage = htmlBody[0]
		}
	}
}

// NonBlocking sets the `Configuration.NonBlocking` field to true.
func NonBlocking() Configurator {
	return func(app *Application) {
		app.config.NonBlocking = true
	}
}

// WithoutServerError will cause to ignore the matched "errors"
// from the main application's `Run/Listen` function.
//
// Usage:
// err := app.Listen(":8080", iris.WithoutServerError(iris.ErrServerClosed))
// will return `nil` if the server's error was `http/iris#ErrServerClosed`.
//
// See `Configuration#IgnoreServerErrors []string` too.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/http-server/listen-addr/omit-server-errors
func WithoutServerError(errors ...error) Configurator {
	return func(app *Application) {
		if len(errors) == 0 {
			return
		}

		errorsAsString := make([]string, len(errors))
		for i, e := range errors {
			errorsAsString[i] = e.Error()
		}

		app.config.IgnoreServerErrors = append(app.config.IgnoreServerErrors, errorsAsString...)
	}
}

// WithoutStartupLog turns off the information send, once, to the terminal when the main server is open.
var WithoutStartupLog = func(app *Application) {
	app.config.DisableStartupLog = true
}

// WithoutBanner is a conversion for the `WithoutStartupLog` option.
//
// Turns off the information send, once, to the terminal when the main server is open.
var WithoutBanner = WithoutStartupLog

// WithoutInterruptHandler disables the automatic graceful server shutdown
// when control/cmd+C pressed.
var WithoutInterruptHandler = func(app *Application) {
	app.config.DisableInterruptHandler = true
}

// WithoutPathCorrection disables the PathCorrection setting.
//
// See `Configuration`.
var WithoutPathCorrection = func(app *Application) {
	app.config.DisablePathCorrection = true
}

// WithPathIntelligence enables the EnablePathIntelligence setting.
//
// See `Configuration`.
var WithPathIntelligence = func(app *Application) {
	app.config.EnablePathIntelligence = true
}

// WithoutPathCorrectionRedirection disables the PathCorrectionRedirection setting.
//
// See `Configuration`.
var WithoutPathCorrectionRedirection = func(app *Application) {
	app.config.DisablePathCorrection = false
	app.config.DisablePathCorrectionRedirection = true
}

// WithoutBodyConsumptionOnUnmarshal disables BodyConsumptionOnUnmarshal setting.
//
// See `Configuration`.
var WithoutBodyConsumptionOnUnmarshal = func(app *Application) {
	app.config.DisableBodyConsumptionOnUnmarshal = true
}

// WithEmptyFormError enables the setting `FireEmptyFormError`.
//
// See `Configuration`.
var WithEmptyFormError = func(app *Application) {
	app.config.FireEmptyFormError = true
}

// WithPathEscape sets the EnablePathEscape setting to true.
//
// See `Configuration`.
var WithPathEscape = func(app *Application) {
	app.config.EnablePathEscape = true
}

// WithLowercaseRouting enables for lowercase routing by
// setting the `ForceLowercaseRoutes` to true.
//
// See `Configuration`.
var WithLowercaseRouting = func(app *Application) {
	app.config.ForceLowercaseRouting = true
}

// WithDynamicHandler enables for dynamic routing by
// setting the `EnableDynamicHandler` to true.
//
// See `Configuration`.
var WithDynamicHandler = func(app *Application) {
	app.config.EnableDynamicHandler = true
}

// WithOptimizations can force the application to optimize for the best performance where is possible.
//
// See `Configuration`.
var WithOptimizations = func(app *Application) {
	app.config.EnableOptimizations = true
}

// WithProtoJSON enables the proto marshaler on Context.JSON method.
//
// See `Configuration` for more.
var WithProtoJSON = func(app *Application) {
	app.config.EnableProtoJSON = true
}

// WithEasyJSON enables the fast easy json marshaler on Context.JSON method.
//
// See `Configuration` for more.
var WithEasyJSON = func(app *Application) {
	app.config.EnableEasyJSON = true
}

// WithFireMethodNotAllowed enables the FireMethodNotAllowed setting.
//
// See `Configuration`.
var WithFireMethodNotAllowed = func(app *Application) {
	app.config.FireMethodNotAllowed = true
}

// WithoutAutoFireStatusCode sets the DisableAutoFireStatusCode setting to true.
//
// See `Configuration`.
var WithoutAutoFireStatusCode = func(app *Application) {
	app.config.DisableAutoFireStatusCode = true
}

// WithResetOnFireErrorCode sets the ResetOnFireErrorCode setting to true.
//
// See `Configuration`.
var WithResetOnFireErrorCode = func(app *Application) {
	app.config.ResetOnFireErrorCode = true
}

// WithURLParamSeparator sets the URLParamSeparator setting to "sep".
//
// See `Configuration`.
var WithURLParamSeparator = func(sep string) Configurator {
	return func(app *Application) {
		app.config.URLParamSeparator = &sep
	}
}

// WithTimeFormat sets the TimeFormat setting.
//
// See `Configuration`.
func WithTimeFormat(timeformat string) Configurator {
	return func(app *Application) {
		app.config.TimeFormat = timeformat
	}
}

// WithCharset sets the Charset setting.
//
// See `Configuration`.
func WithCharset(charset string) Configurator {
	return func(app *Application) {
		app.config.Charset = charset
	}
}

// WithPostMaxMemory sets the maximum post data size
// that a client can send to the server, this differs
// from the overall request body size which can be modified
// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
//
// Defaults to 32MB or 32 << 20 or 32*iris.MB if you prefer.
func WithPostMaxMemory(limit int64) Configurator {
	return func(app *Application) {
		app.config.PostMaxMemory = limit
	}
}

// WithRemoteAddrHeader adds a new request header name
// that can be used to validate the client's real IP.
func WithRemoteAddrHeader(header ...string) Configurator {
	return func(app *Application) {
		for _, h := range header {
			exists := false
			for _, v := range app.config.RemoteAddrHeaders {
				if v == h {
					exists = true
				}
			}

			if !exists {
				app.config.RemoteAddrHeaders = append(app.config.RemoteAddrHeaders, h)
			}
		}
	}
}

// WithoutRemoteAddrHeader removes an existing request header name
// that can be used to validate and parse the client's real IP.
//
// Look `context.RemoteAddr()` for more.
func WithoutRemoteAddrHeader(headerName string) Configurator {
	return func(app *Application) {
		tmp := app.config.RemoteAddrHeaders[:0]
		for _, v := range app.config.RemoteAddrHeaders {
			if v != headerName {
				tmp = append(tmp, v)
			}
		}
		app.config.RemoteAddrHeaders = tmp
	}
}

// WithRemoteAddrPrivateSubnet adds a new private sub-net to be excluded from `context.RemoteAddr`.
// See `WithRemoteAddrHeader` too.
func WithRemoteAddrPrivateSubnet(startIP, endIP string) Configurator {
	return func(app *Application) {
		app.config.RemoteAddrPrivateSubnets = append(app.config.RemoteAddrPrivateSubnets, netutil.IPRange{
			Start: startIP,
			End:   endIP,
		})
	}
}

// WithSSLProxyHeader sets a SSLProxyHeaders key value pair.
// Example: WithSSLProxyHeader("X-Forwarded-Proto", "https").
// See `Context.IsSSL` for more.
func WithSSLProxyHeader(headerKey, headerValue string) Configurator {
	return func(app *Application) {
		if app.config.SSLProxyHeaders == nil {
			app.config.SSLProxyHeaders = make(map[string]string)
		}

		app.config.SSLProxyHeaders[headerKey] = headerValue
	}
}

// WithHostProxyHeader sets a HostProxyHeaders key value pair.
// Example: WithHostProxyHeader("X-Host").
// See `Context.Host` for more.
func WithHostProxyHeader(headers ...string) Configurator {
	return func(app *Application) {
		if app.config.HostProxyHeaders == nil {
			app.config.HostProxyHeaders = make(map[string]bool)
		}
		for _, k := range headers {
			app.config.HostProxyHeaders[k] = true
		}
	}
}

// WithOtherValue adds a value based on a key to the Other setting.
//
// See `Configuration.Other`.
func WithOtherValue(key string, val interface{}) Configurator {
	return func(app *Application) {
		if app.config.Other == nil {
			app.config.Other = make(map[string]interface{})
		}
		app.config.Other[key] = val
	}
}

// WithSitemap enables the sitemap generator.
// Use the Route's `SetLastMod`, `SetChangeFreq` and `SetPriority` to modify
// the sitemap's URL child element properties.
// Excluded routes:
// - dynamic
// - subdomain
// - offline
// - ExcludeSitemap method called
//
// It accepts a "startURL" input argument which
// is the prefix for the registered routes that will be included in the sitemap.
//
// If more than 50,000 static routes are registered then sitemaps will be splitted and a sitemap index will be served in
// /sitemap.xml.
//
// If `Application.I18n.Load/LoadAssets` is called then the sitemap will contain translated links for each static route.
//
// If the result does not complete your needs you can take control
// and use the github.com/kataras/sitemap package to generate a customized one instead.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/sitemap.
func WithSitemap(startURL string) Configurator {
	sitemaps := sitemap.New(startURL)
	return func(app *Application) {
		var defaultLang string
		if tags := app.I18n.Tags(); len(tags) > 0 {
			defaultLang = tags[0].String()
			sitemaps.DefaultLang(defaultLang)
		}

		for _, r := range app.GetRoutes() {
			if !r.IsStatic() || r.Subdomain != "" || !r.IsOnline() || r.NoSitemap {
				continue
			}

			loc := r.StaticPath()
			var translatedLinks []sitemap.Link

			for _, tag := range app.I18n.Tags() {
				lang := tag.String()
				langPath := lang
				href := ""
				if lang == defaultLang {
					// http://domain.com/en-US/path to just http://domain.com/path if en-US is the default language.
					langPath = ""
				}

				if app.I18n.PathRedirect {
					// then use the path prefix.
					// e.g. http://domain.com/el-GR/path
					if langPath == "" { // fix double slashes http://domain.com// when self-included default language.
						href = loc
					} else {
						href = "/" + langPath + loc
					}
				} else if app.I18n.Subdomain {
					// then use the subdomain.
					// e.g. http://el.domain.com/path
					scheme := netutil.ResolveSchemeFromVHost(startURL)
					host := strings.TrimLeft(startURL, scheme)
					if langPath != "" {
						href = scheme + strings.Split(langPath, "-")[0] + "." + host + loc
					} else {
						href = loc
					}

				} else if p := app.I18n.URLParameter; p != "" {
					// then use the URL parameter.
					// e.g. http://domain.com/path?lang=el-GR
					href = loc + "?" + p + "=" + lang
				} else {
					// then skip it, we can't generate the link at this state.
					continue
				}

				translatedLinks = append(translatedLinks, sitemap.Link{
					Rel:      "alternate",
					Hreflang: lang,
					Href:     href,
				})
			}

			sitemaps.URL(sitemap.URL{
				Loc:        loc,
				LastMod:    r.LastMod,
				ChangeFreq: r.ChangeFreq,
				Priority:   r.Priority,
				Links:      translatedLinks,
			})
		}

		for _, s := range sitemaps.Build() {
			contentCopy := make([]byte, len(s.Content))
			copy(contentCopy, s.Content)

			handler := func(ctx Context) {
				ctx.ContentType(context.ContentXMLHeaderValue)
				ctx.Write(contentCopy) // nolint:errcheck
			}
			if app.builded {
				routes := app.CreateRoutes([]string{MethodGet, MethodHead, MethodOptions}, s.Path, handler)

				for _, r := range routes {
					if err := app.Router.AddRouteUnsafe(r); err != nil {
						app.Logger().Errorf("sitemap route: %v", err)
					}
				}
			} else {
				app.HandleMany("GET HEAD OPTIONS", s.Path, handler)
			}

		}
	}
}

// WithTunneling is the `iris.Configurator` for the `iris.Configuration.Tunneling` field.
// It's used to enable http tunneling for an Iris Application, per registered host
//
// Alternatively use the `iris.WithConfiguration(iris.Configuration{Tunneling: iris.TunnelingConfiguration{ ...}}}`.
var WithTunneling = func(app *Application) {
	conf := TunnelingConfiguration{
		Tunnels: []Tunnel{{}}, // create empty tunnel, its addr and name are set right before host serve.
	}

	app.config.Tunneling = conf
}

type (
	// TunnelingConfiguration contains configuration
	// for the optional tunneling through ngrok feature.
	// Note that the ngrok should be already installed at the host machine.
	TunnelingConfiguration = tunnel.Configuration
	// Tunnel is the Tunnels field of the TunnelingConfiguration structure.
	Tunnel = tunnel.Tunnel
)

// Configuration holds the necessary settings for an Iris Application instance.
// All fields are optionally, the default values will work for a common web application.
//
// A Configuration value can be passed through `WithConfiguration` Configurator.
// Usage:
// conf := iris.Configuration{ ... }
// app := iris.New()
// app.Configure(iris.WithConfiguration(conf)) OR
// app.Run/Listen(..., iris.WithConfiguration(conf)).
type Configuration struct {
	// VHost lets you customize the trusted domain this server should run on.
	// Its value will be used as the return value of Context.Domain() too.
	// It can be retrieved by the context if needed (i.e router for subdomains)
	VHost string `ini:"v_host" json:"vHost" yaml:"VHost" toml:"VHost" env:"V_HOST"`

	// LogLevel is the log level the application should use to output messages.
	// Logger, by default, is mostly used on Build state but it is also possible
	// that debug error messages could be thrown when the app is running, e.g.
	// when malformed data structures try to be sent on Client (i.e Context.JSON/JSONP/XML...).
	//
	// Defaults to "info". Possible values are:
	// * "disable"
	// * "fatal"
	// * "error"
	// * "warn"
	// * "info"
	// * "debug"
	LogLevel string `ini:"log_level" json:"logLevel" yaml:"LogLevel" toml:"LogLevel" env:"LOG_LEVEL"`

	// SocketSharding enables SO_REUSEPORT (or SO_REUSEADDR for windows)
	// on all registered Hosts.
	// This option allows linear scaling server performance on multi-CPU servers.
	//
	// Please read the following:
	// 1. https://stackoverflow.com/a/14388707
	// 2. https://stackoverflow.com/a/59692868
	// 3. https://www.nginx.com/blog/socket-sharding-nginx-release-1-9-1/
	// 4. (BOOK) Learning HTTP/2: A Practical Guide for Beginners:
	//	  Page 37, To Shard or Not to Shard?
	//
	// Defaults to false.
	SocketSharding bool `ini:"socket_sharding" json:"socketSharding" yaml:"SocketSharding" toml:"SocketSharding" env:"SOCKET_SHARDING"`
	// KeepAlive sets the TCP connection's keep-alive duration.
	// If set to greater than zero then a tcp listener featured keep alive
	// will be used instead of the simple tcp one.
	//
	// Defaults to 0.
	KeepAlive time.Duration `ini:"keepalive" json:"keepAlive" yaml:"KeepAlive" toml:"KeepAlive" env:"KEEP_ALIVE"`
	// Timeout wraps the application's router with an http timeout handler
	// if the value is greater than zero.
	//
	// The underline response writer supports the Pusher interface but does not support
	// the Hijacker or Flusher interfaces when Timeout handler is registered.
	//
	// Read more at: https://pkg.go.dev/net/http#TimeoutHandler.
	Timeout time.Duration `ini:"timeout" json:"timeout" yaml:"Timeout" toml:"Timeout"`
	// TimeoutMessage specifies the HTML body when a handler hits its life time based
	// on the Timeout configuration field.
	TimeoutMessage string `ini:"timeout_message" json:"timeoutMessage" yaml:"TimeoutMessage" toml:"TimeoutMessage"`
	// NonBlocking, if set to true then the server will start listening for incoming connections
	// without blocking the main goroutine. Use the Application.Wait method to block and wait for the server to be up and running.
	NonBlocking bool `ini:"non_blocking" json:"nonBlocking" yaml:"NonBlocking" toml:"NonBlocking"`

	// Tunneling can be optionally set to enable ngrok http(s) tunneling for this Iris app instance.
	// See the `WithTunneling` Configurator too.
	Tunneling TunnelingConfiguration `ini:"tunneling" json:"tunneling,omitempty" yaml:"Tunneling" toml:"Tunneling"`
	// IgnoreServerErrors will cause to ignore the matched "errors"
	// from the main application's `Run` function.
	// This is a slice of string, not a slice of error
	// users can register these errors using yaml or toml configuration file
	// like the rest of the configuration fields.
	//
	// See `WithoutServerError(...)` function too.
	//
	// Example: https://github.com/kataras/iris/tree/main/_examples/http-server/listen-addr/omit-server-errors
	//
	// Defaults to an empty slice.
	IgnoreServerErrors []string `ini:"ignore_server_errors" json:"ignoreServerErrors,omitempty" yaml:"IgnoreServerErrors" toml:"IgnoreServerErrors"`

	// DisableStartupLog if set to true then it turns off the write banner on server startup.
	//
	// Defaults to false.
	DisableStartupLog bool `ini:"disable_startup_log" json:"disableStartupLog,omitempty" yaml:"DisableStartupLog" toml:"DisableStartupLog"`
	// DisableInterruptHandler if set to true then it disables the automatic graceful server shutdown
	// when control/cmd+C pressed.
	// Turn this to true if you're planning to handle this by your own via a custom host.Task.
	//
	// Defaults to false.
	DisableInterruptHandler bool `ini:"disable_interrupt_handler" json:"disableInterruptHandler,omitempty" yaml:"DisableInterruptHandler" toml:"DisableInterruptHandler"`

	// DisablePathCorrection disables the correcting
	// and redirecting or executing directly the handler of
	// the requested path to the registered path
	// for example, if /home/ path is requested but no handler for this Route found,
	// then the Router checks if /home handler exists, if yes,
	// (permanent)redirects the client to the correct path /home.
	//
	// See `DisablePathCorrectionRedirection` to enable direct handler execution instead of redirection.
	//
	// Defaults to false.
	DisablePathCorrection bool `ini:"disable_path_correction" json:"disablePathCorrection,omitempty" yaml:"DisablePathCorrection" toml:"DisablePathCorrection"`
	// DisablePathCorrectionRedirection works whenever configuration.DisablePathCorrection is set to false
	// and if DisablePathCorrectionRedirection set to true then it will fire the handler of the matching route without
	// the trailing slash ("/") instead of send a redirection status.
	//
	// Defaults to false.
	DisablePathCorrectionRedirection bool `ini:"disable_path_correction_redirection" json:"disablePathCorrectionRedirection,omitempty" yaml:"DisablePathCorrectionRedirection" toml:"DisablePathCorrectionRedirection"`
	// EnablePathIntelligence if set to true,
	// the router will redirect HTTP "GET" not found pages to the most closest one path(if any). For example
	// you register a route at "/contact" path -
	// a client tries to reach it by "/cont", the path will be automatic fixed
	// and the client will be redirected to the "/contact" path
	// instead of getting a 404 not found response back.
	//
	// Defaults to false.
	EnablePathIntelligence bool `ini:"enable_path_intelligence" json:"enablePathIntelligence,omitempty" yaml:"EnablePathIntelligence" toml:"EnablePathIntelligence"`
	// EnablePathEscape when is true then its escapes the path and the named parameters (if any).
	// When do you need to Disable(false) it:
	// accepts parameters with slash '/'
	// Request: http://localhost:8080/details/Project%2FDelta
	// ctx.Param("project") returns the raw named parameter: Project%2FDelta
	// which you can escape it manually with net/url:
	// projectName, _ := url.QueryUnescape(c.Param("project").
	//
	// Defaults to false.
	EnablePathEscape bool `ini:"enable_path_escape" json:"enablePathEscape,omitempty" yaml:"EnablePathEscape" toml:"EnablePathEscape"`
	// ForceLowercaseRouting if enabled, converts all registered routes paths to lowercase
	// and it does lowercase the request path too for matching.
	//
	// Defaults to false.
	ForceLowercaseRouting bool `ini:"force_lowercase_routing" json:"forceLowercaseRouting,omitempty" yaml:"ForceLowercaseRouting" toml:"ForceLowercaseRouting"`
	// EnableOptimizations enables dynamic request handler.
	// It gives the router the feature to add routes while in serve-time,
	// when `RefreshRouter` is called.
	// If this setting is set to true, the request handler will use a mutex for data(trie routing) protection,
	// hence the performance cost.
	//
	// Defaults to false.
	EnableDynamicHandler bool `ini:"enable_dynamic_handler" json:"enableDynamicHandler,omitempty" yaml:"EnableDynamicHandler" toml:"EnableDynamicHandler"`
	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405) and
	//  fires the 405 error instead of 404
	// Defaults to false.
	FireMethodNotAllowed bool `ini:"fire_method_not_allowed" json:"fireMethodNotAllowed,omitempty" yaml:"FireMethodNotAllowed" toml:"FireMethodNotAllowed"`
	// DisableAutoFireStatusCode if true then it turns off the http error status code
	// handler automatic execution on error code from a `Context.StatusCode` call.
	// By-default a custom http error handler will be fired when "Context.StatusCode(errorCode)" called.
	//
	// Defaults to false.
	DisableAutoFireStatusCode bool `ini:"disable_auto_fire_status_code" json:"disableAutoFireStatusCode,omitempty" yaml:"DisableAutoFireStatusCode" toml:"DisableAutoFireStatusCode"`
	// ResetOnFireErrorCode if true then any previously response body or headers through
	// response recorder will be ignored and the router
	// will fire the registered (or default) HTTP error handler instead.
	// See `core/router/handler#FireErrorCode` and `Context.EndRequest` for more details.
	//
	// Read more at: https://github.com/kataras/iris/issues/1531
	//
	// Defaults to false.
	ResetOnFireErrorCode bool `ini:"reset_on_fire_error_code" json:"resetOnFireErrorCode,omitempty" yaml:"ResetOnFireErrorCode" toml:"ResetOnFireErrorCode"`

	// URLParamSeparator defines the character(s) separator for Context.URLParamSlice.
	// If empty or null then request url parameters with comma separated values will be retrieved as one.
	//
	// Defaults to comma ",".
	URLParamSeparator *string `ini:"url_param_separator" json:"urlParamSeparator,omitempty" yaml:"URLParamSeparator" toml:"URLParamSeparator"`
	// EnableOptimization when this field is true
	// then the application tries to optimize for the best performance where is possible.
	//
	// Defaults to false.
	// Deprecated. As of version 12.2.x this field does nothing.
	EnableOptimizations bool `ini:"enable_optimizations" json:"enableOptimizations,omitempty" yaml:"EnableOptimizations" toml:"EnableOptimizations"`
	// EnableProtoJSON when this field is true
	// enables the proto marshaler on given proto messages when calling the Context.JSON method.
	//
	// Defaults to false.
	EnableProtoJSON bool `ini:"enable_proto_json" json:"enableProtoJSON,omitempty" yaml:"EnableProtoJSON" toml:"EnableProtoJSON"`
	// EnableEasyJSON when this field is true
	// enables the fast easy json marshaler on compatible struct values when calling the Context.JSON method.
	//
	// Defaults to false.
	EnableEasyJSON bool `ini:"enable_easy_json" json:"enableEasyJSON,omitempty" yaml:"EnableEasyJSON" toml:"EnableEasyJSON"`

	// DisableBodyConsumptionOnUnmarshal manages the reading behavior of the context's body readers/binders.
	// If set to true then it
	// disables the body consumption by the `context.UnmarshalBody/ReadJSON/ReadXML`.
	//
	// By-default io.ReadAll` is used to read the body from the `context.Request.Body which is an `io.ReadCloser`,
	// if this field set to true then a new buffer will be created to read from and the request body.
	// The body will not be changed and existing data before the
	// context.UnmarshalBody/ReadJSON/ReadXML will be not consumed.
	//
	// See `Context.RecordRequestBody` method for the same feature, per-request.
	DisableBodyConsumptionOnUnmarshal bool `ini:"disable_body_consumption" json:"disableBodyConsumptionOnUnmarshal,omitempty" yaml:"DisableBodyConsumptionOnUnmarshal" toml:"DisableBodyConsumptionOnUnmarshal"`
	// FireEmptyFormError returns if set to tue true then the `context.ReadForm/ReadQuery/ReadBody`
	// will return an `iris.ErrEmptyForm` on empty request form data.
	FireEmptyFormError bool `ini:"fire_empty_form_error" json:"fireEmptyFormError,omitempty" yaml:"FireEmptyFormError" toml:"FireEmptyFormError"`

	// TimeFormat time format for any kind of datetime parsing
	// Defaults to  "Mon, 02 Jan 2006 15:04:05 GMT".
	TimeFormat string `ini:"time_format" json:"timeFormat,omitempty" yaml:"TimeFormat" toml:"TimeFormat"`

	// Charset character encoding for various rendering
	// used for templates and the rest of the responses
	// Defaults to "utf-8".
	Charset string `ini:"charset" json:"charset,omitempty" yaml:"Charset" toml:"Charset"`

	// PostMaxMemory sets the maximum post data size
	// that a client can send to the server, this differs
	// from the overall request body size which can be modified
	// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
	//
	// Defaults to 32MB or 32 << 20 if you prefer.
	PostMaxMemory int64 `ini:"post_max_memory" json:"postMaxMemory" yaml:"PostMaxMemory" toml:"PostMaxMemory"`
	//  +----------------------------------------------------+
	//  | Context's keys for values used on various featuers |
	//  +----------------------------------------------------+

	// Context values' keys for various features.
	//
	// LocaleContextKey is used by i18n to get the current request's locale, which contains a translate function too.
	//
	// Defaults to "iris.locale".
	LocaleContextKey string `ini:"locale_context_key" json:"localeContextKey,omitempty" yaml:"LocaleContextKey" toml:"LocaleContextKey"`
	// LanguageContextKey is the context key which a language can be modified by a middleware.
	// It has the highest priority over the rest and if it is empty then it is ignored,
	// if it set to a static string of "default" or to the default language's code
	// then the rest of the language extractors will not be called at all and
	// the default language will be set instead.
	//
	// Use with `Context.SetLanguage("el-GR")`.
	//
	// See `i18n.ExtractFunc` for a more organised way of the same feature.
	// Defaults to "iris.locale.language".
	LanguageContextKey string `ini:"language_context_key" json:"languageContextKey,omitempty" yaml:"LanguageContextKey" toml:"LanguageContextKey"`
	// LanguageInputContextKey is the context key of a language that is given by the end-user.
	// It's the real user input of the language string, matched or not.
	//
	// Defaults to "iris.locale.language.input".
	LanguageInputContextKey string `ini:"language_input_context_key" json:"languageInputContextKey,omitempty" yaml:"LanguageInputContextKey" toml:"LanguageInputContextKey"`
	// VersionContextKey is the context key which an API Version can be modified
	// via a middleware through `SetVersion` method, e.g. `versioning.SetVersion(ctx, ">=1.0.0 <2.0.0")`.
	// Defaults to "iris.api.version".
	VersionContextKey string `ini:"version_context_key" json:"versionContextKey" yaml:"VersionContextKey" toml:"VersionContextKey"`
	// VersionAliasesContextKey is the context key which the versioning feature
	// can look up for alternative values of a version and fallback to that.
	// Head over to the versioning package for more.
	// Defaults to "iris.api.version.aliases"
	VersionAliasesContextKey string `ini:"version_aliases_context_key" json:"versionAliasesContextKey" yaml:"VersionAliasesContextKey" toml:"VersionAliasesContextKey"`
	// ViewEngineContextKey is the context's values key
	// responsible to store and retrieve(view.Engine) the current view engine.
	// A middleware or a Party can modify its associated value to change
	// a view engine that `ctx.View` will render through.
	// If not an engine is registered by the end-developer
	// then its associated value is always nil,
	// meaning that the default value is nil.
	// See `Party.RegisterView` and `Context.ViewEngine` methods as well.
	//
	// Defaults to "iris.view.engine".
	ViewEngineContextKey string `ini:"view_engine_context_key" json:"viewEngineContextKey,omitempty" yaml:"ViewEngineContextKey" toml:"ViewEngineContextKey"`
	// ViewLayoutContextKey is the context's values key
	// responsible to store and retrieve(string) the current view layout.
	// A middleware can modify its associated value to change
	// the layout that `ctx.View` will use to render a template.
	//
	// Defaults to "iris.view.layout".
	ViewLayoutContextKey string `ini:"view_layout_context_key" json:"viewLayoutContextKey,omitempty" yaml:"ViewLayoutContextKey" toml:"ViewLayoutContextKey"`
	// ViewDataContextKey is the context's values key
	// responsible to store and retrieve(interface{}) the current view binding data.
	// A middleware can modify its associated value to change
	// the template's data on-fly.
	//
	// Defaults to "iris.view.data".
	ViewDataContextKey string `ini:"view_data_context_key" json:"viewDataContextKey,omitempty" yaml:"ViewDataContextKey" toml:"ViewDataContextKey"`
	// FallbackViewContextKey is the context's values key
	// responsible to store the view fallback information.
	//
	// Defaults to "iris.view.fallback".
	FallbackViewContextKey string `ini:"fallback_view_context_key" json:"fallbackViewContextKey,omitempty" yaml:"FallbackViewContextKey" toml:"FallbackViewContextKey"`
	// RemoteAddrHeaders are the allowed request headers names
	// that can be valid to parse the client's IP based on.
	// By-default no "X-" header is consired safe to be used for retrieving the
	// client's IP address, because those headers can manually change by
	// the client. But sometimes are useful e.g. when behind a proxy
	// you want to enable the "X-Forwarded-For" or when cloudflare
	// you want to enable the "CF-Connecting-IP", indeed you
	// can allow the `ctx.RemoteAddr()` to use any header
	// that the client may sent.
	//
	// Defaults to an empty slice but an example usage is:
	// RemoteAddrHeaders {
	//    "X-Real-Ip",
	//    "X-Forwarded-For",
	//    "CF-Connecting-IP",
	//    "True-Client-Ip",
	//    "X-Appengine-Remote-Addr",
	//	}
	//
	// Look `context.RemoteAddr()` for more.
	RemoteAddrHeaders []string `ini:"remote_addr_headers" json:"remoteAddrHeaders,omitempty" yaml:"RemoteAddrHeaders" toml:"RemoteAddrHeaders"`
	// RemoteAddrHeadersForce forces the `Context.RemoteAddr()` method
	// to return the first entry of a request header as a fallback,
	// even if that IP is a part of the `RemoteAddrPrivateSubnets` list.
	// The default behavior, if a remote address is part of the `RemoteAddrPrivateSubnets`,
	// is to retrieve the IP from the `Request.RemoteAddr` field instead.
	RemoteAddrHeadersForce bool `ini:"remote_addr_headers_force" json:"remoteAddrHeadersForce,omitempty" yaml:"RemoteAddrHeadersForce" toml:"RemoteAddrHeadersForce"`
	// RemoteAddrPrivateSubnets defines the private sub-networks.
	// They are used to be compared against
	// IP Addresses fetched through `RemoteAddrHeaders` or `Context.Request.RemoteAddr`.
	// For details please navigate through: https://github.com/kataras/iris/issues/1453
	// Defaults to:
	// {
	// 	Start: "10.0.0.0",
	// 	End:   "10.255.255.255",
	// },
	// {
	// 	Start: "100.64.0.0",
	// 	End:   "100.127.255.255",
	// },
	// {
	// 	Start: "172.16.0.0",
	// 	End:   "172.31.255.255",
	// },
	// {
	// 	Start: "192.0.0.0",
	// 	End:   "192.0.0.255",
	// },
	// {
	// 	Start: "192.168.0.0",
	// 	End:   "192.168.255.255",
	// },
	// {
	// 	Start: "198.18.0.0",
	// 	End:   "198.19.255.255",
	// }
	//
	// Look `Context.RemoteAddr()` for more.
	RemoteAddrPrivateSubnets []netutil.IPRange `ini:"remote_addr_private_subnets" json:"remoteAddrPrivateSubnets" yaml:"RemoteAddrPrivateSubnets" toml:"RemoteAddrPrivateSubnets"`
	// SSLProxyHeaders defines the set of header key values
	// that would indicate a valid https Request (look `Context.IsSSL()`).
	// Example: `map[string]string{"X-Forwarded-Proto": "https"}`.
	//
	// Defaults to empty map.
	SSLProxyHeaders map[string]string `ini:"ssl_proxy_headers" json:"sslProxyHeaders" yaml:"SSLProxyHeaders" toml:"SSLProxyHeaders"`
	// HostProxyHeaders defines the set of headers that may hold a proxied hostname value for the clients.
	// Look `Context.Host()` for more.
	// Defaults to empty map.
	HostProxyHeaders map[string]bool `ini:"host_proxy_headers" json:"hostProxyHeaders" yaml:"HostProxyHeaders" toml:"HostProxyHeaders"`
	// Other are the custom, dynamic options, can be empty.
	// This field used only by you to set any app's options you want.
	//
	// Defaults to empty map.
	Other map[string]interface{} `ini:"other" json:"other,omitempty" yaml:"Other" toml:"Other"`
}

var _ context.ConfigurationReadOnly = (*Configuration)(nil)

// GetVHost returns the VHost config field.
func (c *Configuration) GetVHost() string {
	vhost := c.VHost
	return vhost
}

// SetVHost sets the VHost config field.
func (c *Configuration) SetVHost(s string) {
	c.VHost = s
}

// GetLogLevel returns the LogLevel field.
func (c *Configuration) GetLogLevel() string {
	return c.LogLevel
}

// GetSocketSharding returns the SocketSharding field.
func (c *Configuration) GetSocketSharding() bool {
	return c.SocketSharding
}

// GetKeepAlive returns the KeepAlive field.
func (c *Configuration) GetKeepAlive() time.Duration {
	return c.KeepAlive
}

// GetTimeout returns the Timeout field.
func (c *Configuration) GetTimeout() time.Duration {
	return c.Timeout
}

// GetNonBlocking returns the NonBlocking field.
func (c *Configuration) GetNonBlocking() bool {
	return c.NonBlocking
}

// GetTimeoutMessage returns the TimeoutMessage field.
func (c *Configuration) GetTimeoutMessage() string {
	return c.TimeoutMessage
}

// GetDisablePathCorrection returns the DisablePathCorrection field.
func (c *Configuration) GetDisablePathCorrection() bool {
	return c.DisablePathCorrection
}

// GetDisablePathCorrectionRedirection returns the DisablePathCorrectionRedirection field.
func (c *Configuration) GetDisablePathCorrectionRedirection() bool {
	return c.DisablePathCorrectionRedirection
}

// GetEnablePathIntelligence returns the EnablePathIntelligence field.
func (c *Configuration) GetEnablePathIntelligence() bool {
	return c.EnablePathIntelligence
}

// GetEnablePathEscape returns the EnablePathEscape field.
func (c *Configuration) GetEnablePathEscape() bool {
	return c.EnablePathEscape
}

// GetForceLowercaseRouting returns the ForceLowercaseRouting field.
func (c *Configuration) GetForceLowercaseRouting() bool {
	return c.ForceLowercaseRouting
}

// GetEnableDynamicHandler returns the EnableDynamicHandler field.
func (c *Configuration) GetEnableDynamicHandler() bool {
	return c.EnableDynamicHandler
}

// GetFireMethodNotAllowed returns the FireMethodNotAllowed field.
func (c *Configuration) GetFireMethodNotAllowed() bool {
	return c.FireMethodNotAllowed
}

// GetEnableOptimizations returns the EnableOptimizations.
func (c *Configuration) GetEnableOptimizations() bool {
	return c.EnableOptimizations
}

// GetEnableProtoJSON returns the EnableProtoJSON field.
func (c *Configuration) GetEnableProtoJSON() bool {
	return c.EnableProtoJSON
}

// GetEnableEasyJSON returns the EnableEasyJSON field.
func (c *Configuration) GetEnableEasyJSON() bool {
	return c.EnableEasyJSON
}

// GetDisableBodyConsumptionOnUnmarshal returns the DisableBodyConsumptionOnUnmarshal field.
func (c *Configuration) GetDisableBodyConsumptionOnUnmarshal() bool {
	return c.DisableBodyConsumptionOnUnmarshal
}

// GetFireEmptyFormError returns the DisableBodyConsumptionOnUnmarshal field.
func (c *Configuration) GetFireEmptyFormError() bool {
	return c.FireEmptyFormError
}

// GetDisableAutoFireStatusCode returns the DisableAutoFireStatusCode field.
func (c *Configuration) GetDisableAutoFireStatusCode() bool {
	return c.DisableAutoFireStatusCode
}

// GetResetOnFireErrorCode returns ResetOnFireErrorCode field.
func (c *Configuration) GetResetOnFireErrorCode() bool {
	return c.ResetOnFireErrorCode
}

// GetURLParamSeparator returns URLParamSeparator field.
func (c *Configuration) GetURLParamSeparator() *string {
	return c.URLParamSeparator
}

// GetTimeFormat returns the TimeFormat field.
func (c *Configuration) GetTimeFormat() string {
	return c.TimeFormat
}

// GetCharset returns the Charset field.
func (c *Configuration) GetCharset() string {
	return c.Charset
}

// GetPostMaxMemory returns the PostMaxMemory field.
func (c *Configuration) GetPostMaxMemory() int64 {
	return c.PostMaxMemory
}

// GetLocaleContextKey returns the LocaleContextKey field.
func (c *Configuration) GetLocaleContextKey() string {
	return c.LocaleContextKey
}

// GetLanguageContextKey returns the LanguageContextKey field.
func (c *Configuration) GetLanguageContextKey() string {
	return c.LanguageContextKey
}

// GetLanguageInputContextKey returns the LanguageInputContextKey field.
func (c *Configuration) GetLanguageInputContextKey() string {
	return c.LanguageInputContextKey
}

// GetVersionContextKey returns the VersionContextKey field.
func (c *Configuration) GetVersionContextKey() string {
	return c.VersionContextKey
}

// GetVersionAliasesContextKey returns the VersionAliasesContextKey field.
func (c *Configuration) GetVersionAliasesContextKey() string {
	return c.VersionAliasesContextKey
}

// GetViewEngineContextKey returns the ViewEngineContextKey field.
func (c *Configuration) GetViewEngineContextKey() string {
	return c.ViewEngineContextKey
}

// GetViewLayoutContextKey returns the ViewLayoutContextKey field.
func (c *Configuration) GetViewLayoutContextKey() string {
	return c.ViewLayoutContextKey
}

// GetViewDataContextKey returns the ViewDataContextKey field.
func (c *Configuration) GetViewDataContextKey() string {
	return c.ViewDataContextKey
}

// GetFallbackViewContextKey returns the FallbackViewContextKey field.
func (c *Configuration) GetFallbackViewContextKey() string {
	return c.FallbackViewContextKey
}

// GetRemoteAddrHeaders returns the RemoteAddrHeaders field.
func (c *Configuration) GetRemoteAddrHeaders() []string {
	return c.RemoteAddrHeaders
}

// GetRemoteAddrHeadersForce returns RemoteAddrHeadersForce field.
func (c *Configuration) GetRemoteAddrHeadersForce() bool {
	return c.RemoteAddrHeadersForce
}

// GetSSLProxyHeaders returns the SSLProxyHeaders field.
func (c *Configuration) GetSSLProxyHeaders() map[string]string {
	return c.SSLProxyHeaders
}

// GetRemoteAddrPrivateSubnets returns the RemoteAddrPrivateSubnets field.
func (c *Configuration) GetRemoteAddrPrivateSubnets() []netutil.IPRange {
	return c.RemoteAddrPrivateSubnets
}

// GetHostProxyHeaders returns the HostProxyHeaders field.
func (c *Configuration) GetHostProxyHeaders() map[string]bool {
	return c.HostProxyHeaders
}

// GetOther returns the Other field.
func (c *Configuration) GetOther() map[string]interface{} {
	return c.Other
}

// WithConfiguration sets the "c" values to the framework's configurations.
//
// Usage:
// app.Listen(":8080", iris.WithConfiguration(iris.Configuration{/* fields here */ }))
// or
// iris.WithConfiguration(iris.YAML("./cfg/iris.yml"))
// or
// iris.WithConfiguration(iris.TOML("./cfg/iris.tml"))
func WithConfiguration(c Configuration) Configurator {
	return func(app *Application) {
		main := app.config

		if main == nil {
			app.config = &c
			return
		}

		if v := c.LogLevel; v != "" {
			main.LogLevel = v
		}

		if v := c.SocketSharding; v {
			main.SocketSharding = v
		}

		if v := c.KeepAlive; v > 0 {
			main.KeepAlive = v
		}

		if v := c.Timeout; v > 0 {
			main.Timeout = v
		}

		if v := c.TimeoutMessage; v != "" {
			main.TimeoutMessage = v
		}

		if v := c.NonBlocking; v {
			main.NonBlocking = v
		}

		if len(c.Tunneling.Tunnels) > 0 {
			main.Tunneling = c.Tunneling
		}

		if v := c.IgnoreServerErrors; len(v) > 0 {
			main.IgnoreServerErrors = append(main.IgnoreServerErrors, v...)
		}

		if v := c.DisableStartupLog; v {
			main.DisableStartupLog = v
		}

		if v := c.DisableInterruptHandler; v {
			main.DisableInterruptHandler = v
		}

		if v := c.DisablePathCorrection; v {
			main.DisablePathCorrection = v
		}

		if v := c.DisablePathCorrectionRedirection; v {
			main.DisablePathCorrectionRedirection = v
		}

		if v := c.EnablePathIntelligence; v {
			main.EnablePathIntelligence = v
		}

		if v := c.EnablePathEscape; v {
			main.EnablePathEscape = v
		}

		if v := c.ForceLowercaseRouting; v {
			main.ForceLowercaseRouting = v
		}

		if v := c.EnableOptimizations; v {
			main.EnableOptimizations = v
		}

		if v := c.EnableProtoJSON; v {
			main.EnableProtoJSON = v
		}

		if v := c.EnableEasyJSON; v {
			main.EnableEasyJSON = v
		}

		if v := c.FireMethodNotAllowed; v {
			main.FireMethodNotAllowed = v
		}

		if v := c.DisableAutoFireStatusCode; v {
			main.DisableAutoFireStatusCode = v
		}

		if v := c.ResetOnFireErrorCode; v {
			main.ResetOnFireErrorCode = v
		}

		if v := c.URLParamSeparator; v != nil {
			main.URLParamSeparator = v
		}

		if v := c.DisableBodyConsumptionOnUnmarshal; v {
			main.DisableBodyConsumptionOnUnmarshal = v
		}

		if v := c.FireEmptyFormError; v {
			main.FireEmptyFormError = v
		}

		if v := c.TimeFormat; v != "" {
			main.TimeFormat = v
		}

		if v := c.Charset; v != "" {
			main.Charset = v
		}

		if v := c.PostMaxMemory; v > 0 {
			main.PostMaxMemory = v
		}

		if v := c.LocaleContextKey; v != "" {
			main.LocaleContextKey = v
		}

		if v := c.LanguageContextKey; v != "" {
			main.LanguageContextKey = v
		}

		if v := c.LanguageInputContextKey; v != "" {
			main.LanguageInputContextKey = v
		}

		if v := c.VersionContextKey; v != "" {
			main.VersionContextKey = v
		}

		if v := c.VersionAliasesContextKey; v != "" {
			main.VersionAliasesContextKey = v
		}

		if v := c.ViewEngineContextKey; v != "" {
			main.ViewEngineContextKey = v
		}
		if v := c.ViewLayoutContextKey; v != "" {
			main.ViewLayoutContextKey = v
		}
		if v := c.ViewDataContextKey; v != "" {
			main.ViewDataContextKey = v
		}
		if v := c.FallbackViewContextKey; v != "" {
			main.FallbackViewContextKey = v
		}

		if v := c.RemoteAddrHeaders; len(v) > 0 {
			main.RemoteAddrHeaders = v
		}

		if v := c.RemoteAddrHeadersForce; v {
			main.RemoteAddrHeadersForce = v
		}

		if v := c.RemoteAddrPrivateSubnets; len(v) > 0 {
			main.RemoteAddrPrivateSubnets = v
		}

		if v := c.SSLProxyHeaders; len(v) > 0 {
			if main.SSLProxyHeaders == nil {
				main.SSLProxyHeaders = make(map[string]string, len(v))
			}
			for key, value := range v {
				main.SSLProxyHeaders[key] = value
			}
		}

		if v := c.HostProxyHeaders; len(v) > 0 {
			if main.HostProxyHeaders == nil {
				main.HostProxyHeaders = make(map[string]bool, len(v))
			}
			for key, value := range v {
				main.HostProxyHeaders[key] = value
			}
		}

		if v := c.Other; len(v) > 0 {
			if main.Other == nil {
				main.Other = make(map[string]interface{}, len(v))
			}
			for key, value := range v {
				main.Other[key] = value
			}
		}
	}
}

// DefaultTimeoutMessage is the default timeout message which is rendered
// on expired handlers when timeout handler is registered (see Timeout configuration field).
var DefaultTimeoutMessage = `<html><head><title>Timeout</title></head><body><h1>Timeout</h1>Looks like the server is taking too long to respond, this can be caused by either poor connectivity or an error with our servers. Please try again in a while.</body></html>`

func toStringPtr(s string) *string {
	return &s
}

// DefaultConfiguration returns the default configuration for an iris station, fills the main Configuration
func DefaultConfiguration() Configuration {
	return Configuration{
		LogLevel:                          "info",
		SocketSharding:                    false,
		KeepAlive:                         0,
		Timeout:                           0,
		TimeoutMessage:                    DefaultTimeoutMessage,
		NonBlocking:                       false,
		DisableStartupLog:                 false,
		DisableInterruptHandler:           false,
		DisablePathCorrection:             false,
		EnablePathEscape:                  false,
		ForceLowercaseRouting:             false,
		FireMethodNotAllowed:              false,
		DisableBodyConsumptionOnUnmarshal: false,
		FireEmptyFormError:                false,
		DisableAutoFireStatusCode:         false,
		ResetOnFireErrorCode:              false,
		URLParamSeparator:                 toStringPtr(","),
		TimeFormat:                        "Mon, 02 Jan 2006 15:04:05 GMT",
		Charset:                           "utf-8",

		// PostMaxMemory is for post body max memory.
		//
		// The request body the size limit
		// can be set by the middleware `LimitRequestBodySize`
		// or `context#SetMaxRequestBodySize`.
		PostMaxMemory:            32 << 20, // 32MB
		LocaleContextKey:         "iris.locale",
		LanguageContextKey:       "iris.locale.language",
		LanguageInputContextKey:  "iris.locale.language.input",
		VersionContextKey:        "iris.api.version",
		VersionAliasesContextKey: "iris.api.version.aliases",
		ViewEngineContextKey:     "iris.view.engine",
		ViewLayoutContextKey:     "iris.view.layout",
		ViewDataContextKey:       "iris.view.data",
		FallbackViewContextKey:   "iris.view.fallback",
		RemoteAddrHeaders:        nil,
		RemoteAddrHeadersForce:   false,
		RemoteAddrPrivateSubnets: []netutil.IPRange{
			{
				Start: "10.0.0.0",
				End:   "10.255.255.255",
			},
			{
				Start: "100.64.0.0",
				End:   "100.127.255.255",
			},
			{
				Start: "172.16.0.0",
				End:   "172.31.255.255",
			},
			{
				Start: "192.0.0.0",
				End:   "192.0.0.255",
			},
			{
				Start: "192.168.0.0",
				End:   "192.168.255.255",
			},
			{
				Start: "198.18.0.0",
				End:   "198.19.255.255",
			},
		},
		SSLProxyHeaders:     make(map[string]string),
		HostProxyHeaders:    make(map[string]bool),
		EnableOptimizations: false,
		EnableProtoJSON:     false,
		EnableEasyJSON:      false,
		Other:               make(map[string]interface{}),
	}
}
