package httptest

import (
	"github.com/gavv/httpexpect"
	"github.com/kataras/iris"
	"net/http"
	"testing"
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

// New Prepares and returns a new test framework based on the api
// is useful when you need to have more than one test framework for the same iris instance
// usage:
// iris.Get("/mypath", func(ctx *iris.Context){ctx.Write("my body")})
// ...
// e := httptest.New(iris.Default, t)
// e.GET("/mypath").Expect().Status(iris.StatusOK).Body().Equal("my body")
//
// You can find example on the https://github.com/kataras/iris/glob/master/context_test.go
func New(api *iris.Framework, t *testing.T, setters ...OptionSetter) *httpexpect.Expect {
	conf := DefaultConfiguration()
	for _, setter := range setters {
		setter.Set(conf)
	}

	api.Set(iris.OptionDisableBanner(true))

	baseURL := ""
	if !api.Plugins.PreBuildFired() {
		api.Build()
	}
	if !conf.ExplicitURL {
		baseURL = api.Config.VScheme + api.Config.VHost
		// if it's still empty then set it to the default server addr
		if baseURL == "" {
			baseURL = iris.SchemeHTTP + iris.DefaultServerAddr
		}

	}

	testConfiguration := httpexpect.Config{
		BaseURL: baseURL,
		Client: &http.Client{
			Transport: httpexpect.NewFastBinder(api.Router),
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
