package iris

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"time"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
)

type (
	// OptionSetter sets a configuration field to the main configuration
	// used to help developers to write less and configure only what
	// they really want and nothing else.
	//
	// Usage:
	// iris.New(iris.Configuration{Charset: "UTF-8", Gzip:true})
	// now can be done also by using iris.Option$FIELD:
	// iris.New(iris.OptionCharset("UTF-8"), iris.OptionGzip(true))
	//
	// Benefits:
	// 1. Developers have no worries what option to pass,
	//    they can just type iris.Option and all options should
	//    be visible to their editor's autocomplete-popup window
	// 2. Can be passed with any order
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

// YAML reads Configuration from a file.yml.
//
// Accepts the absolute path of the file.yml.
// An error will be shown to the user via panic with the error message.
// Error may occur when the file.yml doesn't exists or is not formatted correctly.
//
// Usage:
// 1. `app := iris.New(YAML("myfile.yml"))`
// 2. `app.Set(YAML("myfile.yml"))`
func YAML(filename string) Configuration {
	c := DefaultConfiguration()

	// get the abs
	// which will try to find the 'filename' from current workind dir too.
	yamlAbsPath, err := filepath.Abs(filename)
	if err != nil {
		panic("FATAL ERROR .yml.filename to absolute: " + err.Error())
	}

	// read the raw contents of the file
	data, err := ioutil.ReadFile(yamlAbsPath)
	if err != nil {
		panic("FATAL ERROR .yml.ReadFile: " + err.Error())
	}

	// put the file's contents as yaml to the default configuration(c)
	if err := yaml.Unmarshal(data, &c); err != nil {
		panic("FATAL ERROR .yml.Unmarshal: " + err.Error())
	}

	return c
}

// Configuration the whole configuration for an Iris station instance
// these can be passed via options also, look at the top of this file(configuration.go).
// Configuration is a valid OptionSetter.
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
	// Default comes from iris.Default.Listen/.Serve with iris' listeners (iris.TCP4/UNIX/TLS/LETSENCRYPT).
	VHost string `yaml:"vHost"`

	// VScheme is the scheme (http:// or https://) putted at the template function '{{url }}'
	// It's an optional field,
	// When to set VScheme manually:
	// 1. You didn't start the main server using $instance.Listen/ListenTLS/ListenLETSENCRYPT
	//    or $instance.Serve($instance.TCP4()/.TLS...)
	// 2. if you're using something like nginx and have iris listening with
	//   addr only(http://) but the nginx mapper is listening to https://
	//
	// Default comes from iris.Default.Listen/.Serve with iris' listeners (TCP4,UNIX,TLS,LETSENCRYPT).
	VScheme string `yaml:"vScheme"`

	// ReadTimeout is the maximum duration before timing out read of the request.
	ReadTimeout time.Duration `yaml:"readTimeout"`

	// WriteTimeout is the maximum duration before timing out write of the response.
	WriteTimeout time.Duration `yaml:"writeTimeout"`

	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and
	// values, including the request line. It does not limit the
	// size of the request body.
	// If zero, DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int `yaml:"maxHeaderBytes"`

	// CheckForUpdates will try to search for newer version of Iris based on the https://github.com/kataras/iris/releases
	// If a newer version found then the app will ask the he dev/user if want to update the 'x' version
	// if 'y' is pressed then the updater will try to install the latest version
	// the updater, will notify the dev/user that the update is finished and should restart the App manually.
	// Notes:
	// 1. Experimental feature
	// 2. If setted to true, the app will start the server normally and runs the updater in its own goroutine,
	//    in order to no delay the boot time on your development state.
	// 3. If you as developer edited the $GOPATH/src/github/kataras or any other Iris' Go dependencies at the past
	//    then the update process will fail.
	//
	// Usage: app := iris.New(iris.Configuration{CheckForUpdates: true})
	//
	// Defaults to false.
	CheckForUpdates bool `yaml:"checkForUpdates"`

	// DisablePathCorrection corrects and redirects the requested path to the registered path
	// for example, if /home/ path is requested but no handler for this Route found,
	// then the Router checks if /home handler exists, if yes,
	// (permant)redirects the client to the correct path /home
	//
	// Defaults to false.
	DisablePathCorrection bool `yaml:"disablePathCorrection"`

	// EnablePathEscape when is true then its escapes the path, the named parameters (if any).
	// Change to false it if you want something like this https://github.com/kataras/iris/issues/135 to work
	//
	// When do you need to Disable(false) it:
	// accepts parameters with slash '/'
	// Request: http://localhost:8080/details/Project%2FDelta
	// ctx.Param("project") returns the raw named parameter: Project%2FDelta
	// which you can escape it manually with net/url:
	// projectName, _ := url.QueryUnescape(c.Param("project").
	//
	// Defaults to false.
	EnablePathEscape bool `yaml:"enablePathEscape"`

	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405) and
	//  fires the 405 error instead of 404
	// Defaults to false.
	FireMethodNotAllowed bool `yaml:"fireMethodNotAllowed"`

	// DisableBanner outputs the iris banner at startup
	//
	// Defaults to false.
	DisableBanner bool `yaml:"disableBanner"`

	// DisableBodyConsumptionOnUnmarshal manages the reading behavior of the context's body readers/binders.
	// If setted to true then it
	// disables the body consumption by the `context.UnmarshalBody/ReadJSON/ReadXML`.
	//
	// By-default io.ReadAll` is used to read the body from the `context.Request.Body which is an `io.ReadCloser`,
	// if this field setted to true then a new buffer will be created to read from and the request body.
	// The body will not be changed and existing data before the
	// context.UnmarshalBody/ReadJSON/ReadXML will be not consumed.
	DisableBodyConsumptionOnUnmarshal bool `yaml:"disableBodyConsumptionOnUnmarshal"`

	// TimeFormat time format for any kind of datetime parsing
	// Defauls to  "Mon, 02 Jan 2006 15:04:05 GMT".
	TimeFormat string `yaml:"timeFormat"`

	// Charset character encoding for various rendering
	// used for templates and the rest of the responses
	// Defaults to "UTF-8".
	Charset string `yaml:"charset"`

	// Gzip enables gzip compression on your Render actions, this includes any type of render,
	// templates and pure/raw content
	// If you don't want to enable it globally, you could just use the third parameter
	// on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
	// Defaults to false.
	Gzip bool `yaml:"gzip"`

	// Other are the custom, dynamic options, can be empty.
	// This field used only by you to set any app's options you want
	// or by custom adaptors, it's a way to simple communicate between your adaptors (if any)
	// Defaults to a non-nil empty map.
	Other map[string]interface{} `yaml:"other"`
}

// Set implements the OptionSetter
func (c Configuration) Set(main *Configuration) {
	if err := mergo.MergeWithOverwrite(main, c); err != nil {
		panic("FATAL ERROR .Configuration as OptionSetter: " + err.Error())
	}
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
	// Default comes from iris.Default.Listen/.Serve with iris' listeners (iris.TCP4/UNIX/TLS/LETSENCRYPT).
	OptionVHost = func(val string) OptionSet {
		return func(c *Configuration) {
			c.VHost = val
		}
	}

	// OptionVScheme is the scheme (http:// or https://) putted at the template function '{{url }}'
	// It's an optional field,
	// When to set Scheme manually:
	// 1. You didn't start the main server using $instance.Listen/ListenTLS/ListenLETSENCRYPT
	//     or $instance.Serve($instance.TCP4()/.TLS...)
	// 2. if you're using something like nginx and have iris listening with
	//    addr only(http://) but the nginx mapper is listening to https://
	//
	// Default comes from iris.Default.Listen/.Serve with iris' listeners (TCP4,UNIX,TLS,LETSENCRYPT).
	OptionVScheme = func(val string) OptionSet {
		return func(c *Configuration) {
			c.VScheme = val
		}
	}

	// OptionReadTimeout sets the Maximum duration before timing out read of the request.
	OptionReadTimeout = func(val time.Duration) OptionSet {
		return func(c *Configuration) {
			c.ReadTimeout = val
		}
	}

	// OptionWriteTimeout sets the Maximum duration before timing out write of the response.
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
	// Defaults to false.
	OptionCheckForUpdates = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.CheckForUpdates = val
		}
	}

	// OptionDisablePathCorrection corrects and redirects the requested path to the registered path
	// for example, if /home/ path is requested but no handler for this Route found,
	// then the Router checks if /home handler exists, if yes,
	// (permant)redirects the client to the correct path /home
	//
	// Defaults to false.
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

	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405)
	// and fires the 405 error instead of 404
	// Defaults to false.
	OptionFireMethodNotAllowed = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.FireMethodNotAllowed = val
		}
	}

	// OptionDisableBanner outputs the iris banner at startup.
	//
	// Defaults to false.
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

	// OptionTimeFormat time format for any kind of datetime parsing.
	// Defauls to  "Mon, 02 Jan 2006 15:04:05 GMT".
	OptionTimeFormat = func(val string) OptionSet {
		return func(c *Configuration) {
			c.TimeFormat = val
		}
	}

	// OptionCharset character encoding for various rendering
	// used for templates and the rest of the responses
	// Defaults to "UTF-8".
	OptionCharset = func(val string) OptionSet {
		return func(c *Configuration) {
			c.Charset = val
		}
	}

	// OptionGzip enables gzip compression on your Render actions, this includes any type of render, templates and pure/raw content
	// If you don't want to enable it globally, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
	// Defaults to false.
	OptionGzip = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Gzip = val
		}
	}

	// Other are the custom, dynamic options, can be empty.
	// This field used only by you to set any app's options you want
	// or by custom adaptors, it's a way to simple communicate between your adaptors (if any)
	// Defaults to a non-nil empty map.
	OptionOther = func(key string, val interface{}) OptionSet {
		return func(c *Configuration) {
			if c.Other == nil {
				c.Other = make(map[string]interface{}, 0)
			}
			c.Other[key] = val
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
		DisablePathCorrection:             DefaultDisablePathCorrection,
		EnablePathEscape:                  DefaultEnablePathEscape,
		FireMethodNotAllowed:              false,
		DisableBanner:                     false,
		DisableBodyConsumptionOnUnmarshal: false,
		TimeFormat:                        DefaultTimeFormat,
		Charset:                           DefaultCharset,
		Gzip:                              false,
		Other:                             make(map[string]interface{}, 0),
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
