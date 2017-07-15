package httptest

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/iris-contrib/httpexpect"
	"github.com/kataras/iris"
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
}

// Set implements the OptionSetter for the Configuration itself
func (c Configuration) Set(main *Configuration) {
	main.URL = c.URL
	main.Debug = c.Debug
}

var (
	// URL if setted then it sets the httptest's BaseURL.
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
)

// DefaultConfiguration returns the default configuration for the httptest.
func DefaultConfiguration() *Configuration {
	return &Configuration{URL: "", Debug: false}
}

// New Prepares and returns a new test framework based on the "app".
// You can find example on the https://github.com/kataras/iris/tree/master/_examples/testing/httptest
func New(t *testing.T, app *iris.Application, setters ...OptionSetter) *httpexpect.Expect {
	conf := DefaultConfiguration()
	for _, setter := range setters {
		setter.Set(conf)
	}

	// disable logger by setting it to the  "Panic" level, iris never uses this
	// so it will never print.
	app.Logger().Level = 0
	app.Build()

	testConfiguration := httpexpect.Config{
		BaseURL: conf.URL,
		Client: &http.Client{
			Transport: httpexpect.NewBinder(app),
			Jar:       httpexpect.NewJar(),
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

// NewInsecure same as New but receives a single host instead of the whole framework.
// Useful for testing running TLS servers.
func NewInsecure(t *testing.T, setters ...OptionSetter) *httpexpect.Expect {
	conf := DefaultConfiguration()
	for _, setter := range setters {
		setter.Set(conf)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	testConfiguration := httpexpect.Config{
		BaseURL: conf.URL,
		Client: &http.Client{
			Transport: transport,
			Jar:       httpexpect.NewJar(),
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
