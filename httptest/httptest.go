package httptest

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/i18n"

	"github.com/iris-contrib/httpexpect/v2"
)

type (
	// OptionSetter sets a configuration field to the configuration
	OptionSetter interface {
		// Set receives a pointer to the Configuration type and does the job of filling it
		Set(c *Configuration)
	}
	// OptionSet implements the OptionSetter
	OptionSet func(c *Configuration)
)

// Set is the func which makes the OptionSet an OptionSetter, this is used mostly
func (o OptionSet) Set(c *Configuration) {
	o(c)
}

// Configuration httptest configuration
type Configuration struct {
	// URL the base url.
	// Defaults to empty string "".
	URL string
	// Debug if true then debug messages from the httpexpect will be shown when a test runs
	// Defaults to false.
	Debug bool
	// LogLevel sets the application's log level.
	// Defaults to "disable" when testing.
	LogLevel string

	// If true then the underline httpexpect report will be acquired by the NewRequireReporter
	// call instead of the default NewAssertReporter.
	// Defaults to false.
	Strict bool // Note: if more reports are available in the future then add a Reporter interface as a field.
}

// Set implements the OptionSetter for the Configuration itself
func (c Configuration) Set(main *Configuration) {
	main.URL = c.URL
	main.Debug = c.Debug
	if c.LogLevel != "" {
		main.LogLevel = c.LogLevel
	}
	main.Strict = c.Strict
}

var (
	// URL if set then it sets the httptest's BaseURL.
	// Defaults to empty string "".
	URL = func(schemeAndHost string) OptionSet {
		return func(c *Configuration) {
			c.URL = schemeAndHost
		}
	}
	// Debug if true then debug messages from the httpexpect will be shown when a test runs
	// Defaults to false.
	Debug = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Debug = val
		}
	}

	// LogLevel sets the application's log level.
	// Defaults to disabled when testing.
	LogLevel = func(level string) OptionSet {
		return func(c *Configuration) {
			c.LogLevel = level
		}
	}

	// Strict sets the Strict configuration field to "val".
	// Applies the NewRequireReporter instead of the default one.
	// Use this if you want the test to fail on first error, before all checks have been done.
	Strict = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Strict = val
		}
	}
)

// DefaultConfiguration returns the default configuration for the httptest.
func DefaultConfiguration() *Configuration {
	return &Configuration{URL: "", Debug: false, LogLevel: "disable"}
}

// New Prepares and returns a new test framework based on the "app".
// Usage:
//
//	httptest.New(t, app)
//
// With options:
//
//	httptest.New(t, app, httptest.URL(...), httptest.Debug(true), httptest.LogLevel("debug"), httptest.Strict(true))
//
// Examples at: https://github.com/kataras/iris/tree/main/_examples/testing/httptest and
// https://github.com/kataras/iris/tree/main/_examples/testing/ginkgotest.
func New(t IrisTesty, app *iris.Application, setters ...OptionSetter) *httpexpect.Expect {
	conf := DefaultConfiguration()
	for _, setter := range setters {
		setter.Set(conf)
	}

	// set the logger or disable it (default).
	app.Logger().SetLevel(conf.LogLevel)

	if err := app.Build(); err != nil {
		if conf.LogLevel != "disable" {
			app.Logger().Println(err.Error())
			return nil
		}
	}

	var reporter httpexpect.Reporter

	if conf.Strict {
		reporter = httpexpect.NewRequireReporter(t)
	} else {
		reporter = httpexpect.NewAssertReporter(t)
	}

	testConfiguration := httpexpect.Config{
		BaseURL: conf.URL,
		Client: &http.Client{
			Transport: httpexpect.NewBinder(app),
			Jar:       httpexpect.NewCookieJar(),
		},
		Reporter: reporter,
	}

	if conf.Debug {
		testConfiguration.Printers = []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		}
	}

	return httpexpect.WithConfig(testConfiguration)
}

// NewInsecure same as New but receives a single host instead of the whole framework.
// Useful for testing running TLS servers.
func NewInsecure(t IrisTesty, setters ...OptionSetter) *httpexpect.Expect {
	conf := DefaultConfiguration()
	for _, setter := range setters {
		setter.Set(conf)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS13}, // lint:ignore
	}

	testConfiguration := httpexpect.Config{
		BaseURL: conf.URL,
		Client: &http.Client{
			Transport: transport,
			Jar:       httpexpect.NewCookieJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	}

	if conf.Debug {
		testConfiguration.Printers = []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		}
	}

	return httpexpect.WithConfig(testConfiguration)
}

// Aliases for "net/http/httptest" package. See `Do` package-level function.
var (
	NewRecorder = httptest.NewRecorder
	NewRequest  = httptest.NewRequest
)

// Do is a simple helper which can be used to test handlers individually
// with the "net/http/httptest" package.
// This package contains aliases for `NewRequest` and `NewRecorder` too.
//
// For a more efficient testing please use the `New` function instead.
func Do(w http.ResponseWriter, r *http.Request, handler iris.Handler, irisConfigurators ...iris.Configurator) {
	app := new(iris.Application)
	app.I18n = i18n.New()
	app.Configure(iris.WithConfiguration(iris.DefaultConfiguration()), iris.WithLogLevel("disable"))
	app.Configure(irisConfigurators...)

	app.HTTPErrorHandler = router.NewDefaultHandler(app.ConfigurationReadOnly(), app.Logger())
	app.ContextPool = context.New(func() interface{} {
		return context.NewContext(app)
	})

	ctx := app.ContextPool.Acquire(w, r)
	handler(ctx)
	app.ContextPool.Release(ctx)
}

// IrisTesty is an interface which all testing package should implement.
// The `httptest` standard package and `ginkgo` third-party module do implement this interface indeed.
//
// See the `New` package-level function for more.
type IrisTesty interface {
	Cleanup(func())
	Error(args ...any)
	Errorf(format string, args ...any)
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Helper()
	Log(args ...any)
	Logf(format string, args ...any)
	Name() string
	Setenv(key, value string)
	Skip(args ...any)
	SkipNow()
	Skipf(format string, args ...any)
	Skipped() bool
	TempDir() string
}
