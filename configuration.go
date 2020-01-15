package iris

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/netutil"

	"github.com/BurntSushi/toml"
	"github.com/kataras/sitemap"
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
	data, err := ioutil.ReadFile(yamlAbsPath)
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
// Error may occur when the cfg.yml doesn't exists or is not formatted correctly.
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
//
// Accepts the absolute path of the configuration file.
// An error will be shown to the user via panic with the error message.
// Error may occur when the file doesn't exists or is not formatted correctly.
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
	data, err := ioutil.ReadFile(tomlAbsPath)
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

// variables for configurators don't need any receivers, functions
// for them that need (helps code editors to recognise as variables without parenthesis completion).

// WithoutServerError will cause to ignore the matched "errors"
// from the main application's `Run` function.
//
// Usage:
// err := app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
// will return `nil` if the server's error was `http/iris#ErrServerClosed`.
//
// See `Configuration#IgnoreServerErrors []string` too.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/http-listening/listen-addr/omit-server-errors
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

// WithoutAutoFireStatusCode disables the AutoFireStatusCode setting.
//
// See `Configuration`.
var WithoutAutoFireStatusCode = func(app *Application) {
	app.config.DisableAutoFireStatusCode = true
}

// WithPathEscape enables the PathEscape setting.
//
// See `Configuration`.
var WithPathEscape = func(app *Application) {
	app.config.EnablePathEscape = true
}

// WithOptimizations can force the application to optimize for the best performance where is possible.
//
// See `Configuration`.
var WithOptimizations = func(app *Application) {
	app.config.EnableOptimizations = true
}

// WithFireMethodNotAllowed enables the FireMethodNotAllowed setting.
//
// See `Configuration`.
var WithFireMethodNotAllowed = func(app *Application) {
	app.config.FireMethodNotAllowed = true
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
// from the overral request body size which can be modified
// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
//
// Defaults to 32MB or 32 << 20 if you prefer.
func WithPostMaxMemory(limit int64) Configurator {
	return func(app *Application) {
		app.config.PostMaxMemory = limit
	}
}

// WithRemoteAddrHeader enables or adds a new or existing request header name
// that can be used to validate the client's real IP.
//
// By-default no "X-" header is consired safe to be used for retrieving the
// client's IP address, because those headers can manually change by
// the client. But sometimes are useful e.g., when behind a proxy
// you want to enable the "X-Forwarded-For" or when cloudflare
// you want to enable the "CF-Connecting-IP", inneed you
// can allow the `ctx.RemoteAddr()` to use any header
// that the client may sent.
//
// Defaults to an empty map but an example usage is:
// WithRemoteAddrHeader("X-Forwarded-For")
//
// Look `context.RemoteAddr()` for more.
func WithRemoteAddrHeader(headerName string) Configurator {
	return func(app *Application) {
		if app.config.RemoteAddrHeaders == nil {
			app.config.RemoteAddrHeaders = make(map[string]bool)
		}
		app.config.RemoteAddrHeaders[headerName] = true
	}
}

// WithoutRemoteAddrHeader disables an existing request header name
// that can be used to validate and parse the client's real IP.
//
//
// Keep note that RemoteAddrHeaders is already defaults to an empty map
// so you don't have to call this Configurator if you didn't
// add allowed headers via configuration or via `WithRemoteAddrHeader` before.
//
// Look `context.RemoteAddr()` for more.
func WithoutRemoteAddrHeader(headerName string) Configurator {
	return func(app *Application) {
		if app.config.RemoteAddrHeaders == nil {
			app.config.RemoteAddrHeaders = make(map[string]bool)
		}
		app.config.RemoteAddrHeaders[headerName] = false
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
// Example: https://github.com/kataras/iris/tree/master/_examples/sitemap.
func WithSitemap(startURL string) Configurator {
	sitemaps := sitemap.New(startURL)
	return func(app *Application) {
		var defaultLang string
		if tags := app.I18n.Tags(); len(tags) > 0 {
			defaultLang = tags[0].String()
			sitemaps.DefaultLang(defaultLang)
		}

		for _, r := range app.GetRoutes() {
			if !r.IsStatic() || r.Subdomain != "" {
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
				ctx.Write(contentCopy)
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

// Tunnel is the Tunnels field of the TunnelingConfiguration structure.
type Tunnel struct {
	// Name is the only one required field,
	// it is used to create and close tunnels, e.g. "MyApp".
	// If this field is not empty then ngrok tunnels will be created
	// when the iris app is up and running.
	Name string `json:"name" yaml:"Name" toml:"Name"`
	// Addr is basically optionally as it will be set through
	// Iris built-in Runners, however, if `iris.Raw` is used
	// then this field should be set of form 'hostname:port'
	// because framework cannot be aware
	// of the address you used to run the server on this custom runner.
	Addr string `json:"addr,omitempty" yaml:"Addr" toml:"Addr"`
}

// TunnelingConfiguration contains configuration
// for the optional tunneling through ngrok feature.
// Note that the ngrok should be already installed at the host machine.
type TunnelingConfiguration struct {
	// AuthToken field is optionally and can be used
	// to authenticate the ngrok access.
	// ngrok authtoken <YOUR_AUTHTOKEN>
	AuthToken string `json:"authToken,omitempty" yaml:"AuthToken" toml:"AuthToken"`

	// No...
	// Config is optionally and can be used
	// to load ngrok configuration from file system path.
	//
	// If you don't specify a location for a configuration file,
	// ngrok tries to read one from the default location $HOME/.ngrok2/ngrok.yml.
	// The configuration file is optional; no error is emitted if that path does not exist.
	// Config string `json:"config,omitempty" yaml:"Config" toml:"Config"`

	// Bin is the system binary path of the ngrok executable file.
	// If it's empty then the framework will try to find it through system env variables.
	Bin string `json:"bin,omitempty" yaml:"Bin" toml:"Bin"`

	// WebUIAddr is the web interface address of an already-running ngrok instance.
	// Iris will try to fetch the default web interface address(http://127.0.0.1:4040)
	// to determinate if a ngrok instance is running before try to start it manually.
	// However if a custom web interface address is used,
	// this field must be set e.g. http://127.0.0.1:5050.
	WebInterface string `json:"webInterface,omitempty" yaml:"WebInterface" toml:"WebInterface"`

	// Region is optionally, can be used to set the region which defaults to "us".
	// Available values are:
	// "us" for United States
	// "eu" for Europe
	// "ap" for Asia/Pacific
	// "au" for Australia
	// "sa" for South America
	// "jp" forJapan
	// "in" for India
	Region string `json:"region,omitempty" yaml:"Region" toml:"Region"`

	// Tunnels the collection of the tunnels.
	// One tunnel per Iris Host per Application, usually you only need one.
	Tunnels []Tunnel `json:"tunnels" yaml:"Tunnels" toml:"Tunnels"`
}

func (tc *TunnelingConfiguration) isEnabled() bool {
	return tc != nil && len(tc.Tunnels) > 0
}

func (tc *TunnelingConfiguration) isNgrokRunning() bool {
	_, err := http.Get(tc.WebInterface)
	return err == nil
}

// https://ngrok.com/docs
type ngrokTunnel struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Proto   string `json:"proto"`
	Auth    string `json:"auth"`
	BindTLS bool   `json:"bind_tls"`
}

func (tc TunnelingConfiguration) startTunnel(t Tunnel, publicAddr *string) error {
	tunnelAPIRequest := ngrokTunnel{
		Name:    t.Name,
		Addr:    t.Addr,
		Proto:   "http",
		BindTLS: true,
	}

	if !tc.isNgrokRunning() {
		ngrokBin := "ngrok" // environment binary.

		if tc.Bin == "" {
			_, err := exec.LookPath(ngrokBin)
			if err != nil {
				ngrokEnvVar, found := os.LookupEnv("NGROK")
				if !found {
					return fmt.Errorf(`"ngrok" executable not found, please install it from: https://ngrok.com/download`)
				}

				ngrokBin = ngrokEnvVar
			}
		} else {
			ngrokBin = tc.Bin
		}

		if tc.AuthToken != "" {
			cmd := exec.Command(ngrokBin, "authtoken", tc.AuthToken)
			err := cmd.Run()
			if err != nil {
				return err
			}
		}

		// start -none, start without tunnels.
		//  and finally the -log stdout logs to the stdout otherwise the pipe will never be able to read from, spent a lot of time on this lol.
		cmd := exec.Command(ngrokBin, "start", "-none", "-log", "stdout")

		// if tc.Config != "" {
		// 	cmd.Args = append(cmd.Args, []string{"-config", tc.Config}...)
		// }
		if tc.Region != "" {
			cmd.Args = append(cmd.Args, []string{"-region", tc.Region}...)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}

		if err = cmd.Start(); err != nil {
			return err
		}

		p := make([]byte, 256)
		okText := []byte("client session established")
		for {
			n, err := stdout.Read(p)
			if err != nil {
				return err
			}

			// we need this one:
			// msg="client session established"
			// note that this will block if something terrible happens
			// but ngrok's errors are strong so the error is easy to be resolved without any logs.
			if bytes.Contains(p[:n], okText) {
				break
			}
		}
	}

	return tc.createTunnel(tunnelAPIRequest, publicAddr)
}

func (tc TunnelingConfiguration) stopTunnel(t Tunnel) error {
	url := fmt.Sprintf("%s/api/tunnels/%s", tc.WebInterface, t.Name)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != StatusNoContent {
		return fmt.Errorf("stop return an unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (tc TunnelingConfiguration) createTunnel(tunnelAPIRequest ngrokTunnel, publicAddr *string) error {
	url := fmt.Sprintf("%s/api/tunnels", tc.WebInterface)
	requestData, err := json.Marshal(tunnelAPIRequest)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, context.ContentJSONHeaderValue, bytes.NewBuffer(requestData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	type publicAddrOrErrResp struct {
		PublicAddr string `json:"public_url"`
		Details    struct {
			ErrorText string `json:"err"` // when can't bind more addresses, status code was successful.
		} `json:"details"`
		ErrMsg string `json:"msg"` // when ngrok is not yet ready, status code was unsuccessful.
	}

	var apiResponse publicAddrOrErrResp

	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	if err != nil {
		return err
	}

	if errText := apiResponse.ErrMsg; errText != "" {
		return errors.New(errText)
	}

	if errText := apiResponse.Details.ErrorText; errText != "" {
		return errors.New(errText)
	}

	*publicAddr = apiResponse.PublicAddr
	return nil
}

// Configuration the whole configuration for an iris instance
// these can be passed via options also, look at the top of this file(configuration.go).
// Configuration is a valid OptionSetter.
type Configuration struct {
	// vhost is private and set only with .Run method, it cannot be changed after the first set.
	// It can be retrieved by the context if needed (i.e router for subdomains)
	vhost string

	// Tunneling can be optionally set to enable ngrok http(s) tunneling for this Iris app instance.
	// See the `WithTunneling` Configurator too.
	Tunneling TunnelingConfiguration `json:"tunneling,omitempty" yaml:"Tunneling" toml:"Tunneling"`

	// IgnoreServerErrors will cause to ignore the matched "errors"
	// from the main application's `Run` function.
	// This is a slice of string, not a slice of error
	// users can register these errors using yaml or toml configuration file
	// like the rest of the configuration fields.
	//
	// See `WithoutServerError(...)` function too.
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/http-listening/listen-addr/omit-server-errors
	//
	// Defaults to an empty slice.
	IgnoreServerErrors []string `json:"ignoreServerErrors,omitempty" yaml:"IgnoreServerErrors" toml:"IgnoreServerErrors"`

	// DisableStartupLog if set to true then it turns off the write banner on server startup.
	//
	// Defaults to false.
	DisableStartupLog bool `json:"disableStartupLog,omitempty" yaml:"DisableStartupLog" toml:"DisableStartupLog"`
	// DisableInterruptHandler if set to true then it disables the automatic graceful server shutdown
	// when control/cmd+C pressed.
	// Turn this to true if you're planning to handle this by your own via a custom host.Task.
	//
	// Defaults to false.
	DisableInterruptHandler bool `json:"disableInterruptHandler,omitempty" yaml:"DisableInterruptHandler" toml:"DisableInterruptHandler"`

	// DisablePathCorrection corrects and redirects or executes directly the handler of
	// the requested path to the registered path
	// for example, if /home/ path is requested but no handler for this Route found,
	// then the Router checks if /home handler exists, if yes,
	// (permanent)redirects the client to the correct path /home.
	//
	// See `DisablePathCorrectionRedirection` to enable direct handler execution instead of redirection.
	//
	// Defaults to false.
	DisablePathCorrection bool `json:"disablePathCorrection,omitempty" yaml:"DisablePathCorrection" toml:"DisablePathCorrection"`

	// DisablePathCorrectionRedirection works whenever configuration.DisablePathCorrection is set to false
	// and if DisablePathCorrectionRedirection set to true then it will fire the handler of the matching route without
	// the trailing slash ("/") instead of send a redirection status.
	//
	// Defaults to false.
	DisablePathCorrectionRedirection bool `json:"disablePathCorrectionRedirection,omitempty" yaml:"DisablePathCorrectionRedirection" toml:"DisablePathCorrectionRedirection"`

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
	EnablePathEscape bool `json:"enablePathEscape,omitempty" yaml:"EnablePathEscape" toml:"EnablePathEscape"`

	// EnableOptimization when this field is true
	// then the application tries to optimize for the best performance where is possible.
	//
	// Defaults to false.
	EnableOptimizations bool `json:"enableOptimizations,omitempty" yaml:"EnableOptimizations" toml:"EnableOptimizations"`
	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405) and
	//  fires the 405 error instead of 404
	// Defaults to false.
	FireMethodNotAllowed bool `json:"fireMethodNotAllowed,omitempty" yaml:"FireMethodNotAllowed" toml:"FireMethodNotAllowed"`

	// DisableBodyConsumptionOnUnmarshal manages the reading behavior of the context's body readers/binders.
	// If set to true then it
	// disables the body consumption by the `context.UnmarshalBody/ReadJSON/ReadXML`.
	//
	// By-default io.ReadAll` is used to read the body from the `context.Request.Body which is an `io.ReadCloser`,
	// if this field set to true then a new buffer will be created to read from and the request body.
	// The body will not be changed and existing data before the
	// context.UnmarshalBody/ReadJSON/ReadXML will be not consumed.
	DisableBodyConsumptionOnUnmarshal bool `json:"disableBodyConsumptionOnUnmarshal,omitempty" yaml:"DisableBodyConsumptionOnUnmarshal" toml:"DisableBodyConsumptionOnUnmarshal"`

	// DisableAutoFireStatusCode if true then it turns off the http error status code handler automatic execution
	// from (`context.StatusCodeNotSuccessful`, defaults to < 200 || >= 400).
	// If that is false then for a direct error firing, then call the "context#FireStatusCode(statusCode)" manually.
	//
	// By-default a custom http error handler will be fired when "context.StatusCode(code)" called,
	// code should be equal with the result of the the `context.StatusCodeNotSuccessful` in order to be received as an "http error handler".
	//
	// Developer may want this option to set as true in order to manually call the
	// error handlers when needed via "context#FireStatusCode(< 200 || >= 400)".
	// HTTP Custom error handlers are being registered via app.OnErrorCode(code, handler)".
	//
	// Defaults to false.
	DisableAutoFireStatusCode bool `json:"disableAutoFireStatusCode,omitempty" yaml:"DisableAutoFireStatusCode" toml:"DisableAutoFireStatusCode"`

	// TimeFormat time format for any kind of datetime parsing
	// Defaults to  "Mon, 02 Jan 2006 15:04:05 GMT".
	TimeFormat string `json:"timeFormat,omitempty" yaml:"TimeFormat" toml:"TimeFormat"`

	// Charset character encoding for various rendering
	// used for templates and the rest of the responses
	// Defaults to "UTF-8".
	Charset string `json:"charset,omitempty" yaml:"Charset" toml:"Charset"`

	// PostMaxMemory sets the maximum post data size
	// that a client can send to the server, this differs
	// from the overral request body size which can be modified
	// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
	//
	// Defaults to 32MB or 32 << 20 if you prefer.
	PostMaxMemory int64 `json:"postMaxMemory" yaml:"PostMaxMemory" toml:"PostMaxMemory"`
	//  +----------------------------------------------------+
	//  | Context's keys for values used on various featuers |
	//  +----------------------------------------------------+

	// Context values' keys for various features.
	//
	// LocaleContextKey is used by i18n to get the current request's locale, which contains a translate function too.
	//
	// Defaults to "iris.locale".
	LocaleContextKey string `json:"localeContextKey,omitempty" yaml:"LocaleContextKey" toml:"LocaleContextKey"`

	// GetViewLayoutContextKey is the key of the context's user values' key
	// which is being used to set the template
	// layout from a middleware or the main handler.
	// Overrides the parent's or the configuration's.
	//
	// Defaults to "iris.ViewLayout"
	ViewLayoutContextKey string `json:"viewLayoutContextKey,omitempty" yaml:"ViewLayoutContextKey" toml:"ViewLayoutContextKey"`
	// GetViewDataContextKey is the key of the context's user values' key
	// which is being used to set the template
	// binding data from a middleware or the main handler.
	//
	// Defaults to "iris.viewData"
	ViewDataContextKey string `json:"viewDataContextKey,omitempty" yaml:"ViewDataContextKey" toml:"ViewDataContextKey"`
	// RemoteAddrHeaders are the allowed request headers names
	// that can be valid to parse the client's IP based on.
	// By-default no "X-" header is consired safe to be used for retrieving the
	// client's IP address, because those headers can manually change by
	// the client. But sometimes are useful e.g., when behind a proxy
	// you want to enable the "X-Forwarded-For" or when cloudflare
	// you want to enable the "CF-Connecting-IP", inneed you
	// can allow the `ctx.RemoteAddr()` to use any header
	// that the client may sent.
	//
	// Defaults to an empty map but an example usage is:
	// RemoteAddrHeaders {
	//	"X-Real-Ip":             true,
	//  "X-Forwarded-For":       true,
	// 	"CF-Connecting-IP": 	 true,
	//	}
	//
	// Look `context.RemoteAddr()` for more.
	RemoteAddrHeaders map[string]bool `json:"remoteAddrHeaders,omitempty" yaml:"RemoteAddrHeaders" toml:"RemoteAddrHeaders"`

	// Other are the custom, dynamic options, can be empty.
	// This field used only by you to set any app's options you want.
	//
	// Defaults to a non-nil empty map.
	Other map[string]interface{} `json:"other,omitempty" yaml:"Other" toml:"Other"`
}

var _ context.ConfigurationReadOnly = &Configuration{}

// GetVHost returns the non-exported vhost config field.
//
// If original addr ended with :443 or :80, it will return the host without the port.
// If original addr was :https or :http, it will return localhost.
// If original addr was 0.0.0.0, it will return localhost.
func (c Configuration) GetVHost() string {
	return c.vhost
}

// GetDisablePathCorrection returns the Configuration#DisablePathCorrection,
// DisablePathCorrection corrects and redirects the requested path to the registered path
// for example, if /home/ path is requested but no handler for this Route found,
// then the Router checks if /home handler exists, if yes,
// (permanent)redirects the client to the correct path /home.
func (c Configuration) GetDisablePathCorrection() bool {
	return c.DisablePathCorrection
}

// GetDisablePathCorrectionRedirection returns the Configuration#DisablePathCorrectionRedirection field.
// If DisablePathCorrectionRedirection set to true then it will fire the handler of the matching route without
// the last slash ("/") instead of send a redirection status.
func (c Configuration) GetDisablePathCorrectionRedirection() bool {
	return c.DisablePathCorrectionRedirection
}

// GetEnablePathEscape is the Configuration#EnablePathEscape,
// returns true when its escapes the path, the named parameters (if any).
func (c Configuration) GetEnablePathEscape() bool {
	return c.EnablePathEscape
}

// GetEnableOptimizations returns whether
// the application has performance optimizations enabled.
func (c Configuration) GetEnableOptimizations() bool {
	return c.EnableOptimizations
}

// GetFireMethodNotAllowed returns the Configuration#FireMethodNotAllowed.
func (c Configuration) GetFireMethodNotAllowed() bool {
	return c.FireMethodNotAllowed
}

// GetDisableBodyConsumptionOnUnmarshal returns the Configuration#GetDisableBodyConsumptionOnUnmarshal,
// manages the reading behavior of the context's body readers/binders.
// If returns true then the body consumption by the `context.UnmarshalBody/ReadJSON/ReadXML`
// is disabled.
//
// By-default io.ReadAll` is used to read the body from the `context.Request.Body which is an `io.ReadCloser`,
// if this field set to true then a new buffer will be created to read from and the request body.
// The body will not be changed and existing data before the
// context.UnmarshalBody/ReadJSON/ReadXML will be not consumed.
func (c Configuration) GetDisableBodyConsumptionOnUnmarshal() bool {
	return c.DisableBodyConsumptionOnUnmarshal
}

// GetDisableAutoFireStatusCode returns the Configuration#DisableAutoFireStatusCode.
// Returns true when the http error status code handler automatic execution turned off.
func (c Configuration) GetDisableAutoFireStatusCode() bool {
	return c.DisableAutoFireStatusCode
}

// GetTimeFormat returns the Configuration#TimeFormat,
// format for any kind of datetime parsing.
func (c Configuration) GetTimeFormat() string {
	return c.TimeFormat
}

// GetCharset returns the Configuration#Charset,
// the character encoding for various rendering
// used for templates and the rest of the responses.
func (c Configuration) GetCharset() string {
	return c.Charset
}

// GetPostMaxMemory returns the maximum configured post data size
// that a client can send to the server, this differs
// from the overral request body size which can be modified
// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
//
// Defaults to 32MB or 32 << 20 if you prefer.
func (c Configuration) GetPostMaxMemory() int64 {
	return c.PostMaxMemory
}

// GetLocaleContextKey returns the configuration's LocaleContextKey value,
// used for i18n.
func (c Configuration) GetLocaleContextKey() string {
	return c.LocaleContextKey
}

// GetViewLayoutContextKey returns the key of the context's user values' key
// which is being used to set the template
// layout from a middleware or the main handler.
// Overrides the parent's or the configuration's.
func (c Configuration) GetViewLayoutContextKey() string {
	return c.ViewLayoutContextKey
}

// GetViewDataContextKey returns the key of the context's user values' key
// which is being used to set the template
// binding data from a middleware or the main handler.
func (c Configuration) GetViewDataContextKey() string {
	return c.ViewDataContextKey
}

// GetRemoteAddrHeaders returns the allowed request headers names
// that can be valid to parse the client's IP based on.
// By-default no "X-" header is consired safe to be used for retrieving the
// client's IP address, because those headers can manually change by
// the client. But sometimes are useful e.g., when behind a proxy
// you want to enable the "X-Forwarded-For" or when cloudflare
// you want to enable the "CF-Connecting-IP", inneed you
// can allow the `ctx.RemoteAddr()` to use any header
// that the client may sent.
//
// Defaults to an empty map but an example usage is:
// RemoteAddrHeaders {
//	"X-Real-Ip":             true,
//  "X-Forwarded-For":       true,
// 	"CF-Connecting-IP": 	 true,
//	}
//
// Look `context.RemoteAddr()` for more.
func (c Configuration) GetRemoteAddrHeaders() map[string]bool {
	return c.RemoteAddrHeaders
}

// GetOther returns the Configuration#Other map.
func (c Configuration) GetOther() map[string]interface{} {
	return c.Other
}

// WithConfiguration sets the "c" values to the framework's configurations.
//
// Usage:
// app.Run(iris.Addr(":8080"), iris.WithConfiguration(iris.Configuration{/* fields here */ }))
// or
// iris.WithConfiguration(iris.YAML("./cfg/iris.yml"))
// or
// iris.WithConfiguration(iris.TOML("./cfg/iris.tml"))
func WithConfiguration(c Configuration) Configurator {
	return func(app *Application) {
		main := app.config

		if c.Tunneling.isEnabled() {
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

		if v := c.EnablePathEscape; v {
			main.EnablePathEscape = v
		}

		if v := c.EnableOptimizations; v {
			main.EnableOptimizations = v
		}

		if v := c.FireMethodNotAllowed; v {
			main.FireMethodNotAllowed = v
		}

		if v := c.DisableBodyConsumptionOnUnmarshal; v {
			main.DisableBodyConsumptionOnUnmarshal = v
		}

		if v := c.DisableAutoFireStatusCode; v {
			main.DisableAutoFireStatusCode = v
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

		if v := c.ViewLayoutContextKey; v != "" {
			main.ViewLayoutContextKey = v
		}

		if v := c.ViewDataContextKey; v != "" {
			main.ViewDataContextKey = v
		}

		if v := c.RemoteAddrHeaders; len(v) > 0 {
			if main.RemoteAddrHeaders == nil {
				main.RemoteAddrHeaders = make(map[string]bool, len(v))
			}
			for key, value := range v {
				main.RemoteAddrHeaders[key] = value
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

// DefaultConfiguration returns the default configuration for an iris station, fills the main Configuration
func DefaultConfiguration() Configuration {
	return Configuration{
		DisableStartupLog:                 false,
		DisableInterruptHandler:           false,
		DisablePathCorrection:             false,
		EnablePathEscape:                  false,
		FireMethodNotAllowed:              false,
		DisableBodyConsumptionOnUnmarshal: false,
		DisableAutoFireStatusCode:         false,
		TimeFormat:                        "Mon, 02 Jan 2006 15:04:05 GMT",
		Charset:                           "UTF-8",

		// PostMaxMemory is for post body max memory.
		//
		// The request body the size limit
		// can be set by the middleware `LimitRequestBodySize`
		// or `context#SetMaxRequestBodySize`.
		PostMaxMemory:        32 << 20, // 32MB
		LocaleContextKey:     "iris.locale",
		ViewLayoutContextKey: "iris.viewLayout",
		ViewDataContextKey:   "iris.viewData",
		RemoteAddrHeaders:    make(map[string]bool),
		EnableOptimizations:  false,
		Other:                make(map[string]interface{}),
	}
}
