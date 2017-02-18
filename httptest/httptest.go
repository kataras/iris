package httptest

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/iris-contrib/httpexpect"
	"gopkg.in/kataras/iris.v6"
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
	// ExplicitURL If true then the url (should) be prepended manually, useful when want to test subdomains
	// Default is false
	ExplicitURL bool
	// Debug if true then debug messages from the httpexpect will be shown when a test runs
	// Default is false
	Debug bool
}

// Set implements the OptionSetter for the Configuration itself
func (c Configuration) Set(main *Configuration) {
	main.ExplicitURL = c.ExplicitURL
	main.Debug = c.Debug
}

var (
	// ExplicitURL If true then the url (should) be prepended manually, useful when want to test subdomains
	// Default is false
	ExplicitURL = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.ExplicitURL = val
		}
	}
	// Debug if true then debug messages from the httpexpect will be shown when a test runs
	// Default is false
	Debug = func(val bool) OptionSet {
		return func(c *Configuration) {
			c.Debug = val
		}
	}
)

// DefaultConfiguration returns the default configuration for the httptest
// all values are defaulted to false for clarity
func DefaultConfiguration() *Configuration {
	return &Configuration{ExplicitURL: false, Debug: false}
}

// New Prepares and returns a new test framework based on the app
// is useful when you need to have more than one test framework for the same iris instance
// usage:
// iris.Default.Get("/mypath", func(ctx *iris.Context){ctx.Write("my body")})
// ...
// e := httptest.New(iris.Default, t)
// e.GET("/mypath").Expect().Status(iris.StatusOK).Body().Equal("my body")
//
// You can find example on the https://github.com/kataras/iris/glob/master/context_test.go
func New(app *iris.Framework, t *testing.T, setters ...OptionSetter) *httpexpect.Expect {
	conf := DefaultConfiguration()
	for _, setter := range setters {
		setter.Set(conf)
	}

	baseURL := ""
	app.Boot()

	if !conf.ExplicitURL {
		baseURL = app.Config.VScheme + app.Config.VHost
		// if it's still empty then set it to the default server addr
		if baseURL == "" {
			baseURL = iris.SchemeHTTP + iris.DefaultServerAddr
		}

	}

	testConfiguration := httpexpect.Config{
		BaseURL: baseURL,
		Client: &http.Client{
			Transport: httpexpect.NewBinder(app.Router),
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

// NewInsecure same as New but receives a single host instead of the whole framework
func NewInsecure(baseURL string, t *testing.T, setters ...OptionSetter) *httpexpect.Expect {
	conf := DefaultConfiguration()
	for _, setter := range setters {
		setter.Set(conf)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	testConfiguration := httpexpect.Config{
		BaseURL: baseURL,
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
