package httptest

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/valyala/fasthttp"
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

// Configuration is the httptest main configuration
type Configuration struct {
	// Handler is the net/http handler, if != nil then httptest is testing via net/http's options
	Handler http.Handler
	// RequestHandler is the valyala/fasthttp.RequestHandler, if != nil then httptest is testing via valyala/fasthttp's options
	RequestHandler fasthttp.RequestHandler
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
	if c.Handler != nil {
		main.Handler = c.Handler
	} else if c.RequestHandler != nil {
		main.RequestHandler = c.RequestHandler
	}
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
	// Handler sets the http handler to the httptest , use this function if you test your net/http api
	Handler = func(val http.Handler) OptionSet {
		return func(c *Configuration) {
			c.Handler = val
		}
	}
	// RequestHandler sets the fasthttp handler to the httptest , use this function if you test your valyala/fasthttp api
	RequestHandler = func(val fasthttp.RequestHandler) OptionSet {
		return func(c *Configuration) {
			c.RequestHandler = val
		}
	}
)

// DefaultConfiguration returns the default configuration for the httptest
// all values are defaulted to false for clarity
func DefaultConfiguration() *Configuration {
	return &Configuration{ExplicitURL: false, Debug: false}
}

// New Prepares and returns a new test framework based on a handler
// mux := http.NewServeMux()
// mux.Handle("/",http.HandlerFunc(...))
// ...
// e := httptest.New(t, httptest.Handler(mux))
// e.GET("/mypath").Expect().Status(http.StatusOK).Body().Equal("my body")
func New(t *testing.T, setters ...OptionSetter) *httpexpect.Expect {
	conf := DefaultConfiguration()
	for _, setter := range setters {
		setter.Set(conf)
	}

	baseURL := ""
	if !conf.ExplicitURL {
		baseURL = "http://localhost:9999"
	}

	client := &http.Client{Jar: httpexpect.NewJar()}

	// check if net/http or valyala/fasthttp
	if conf.Handler != nil {
		client.Transport = httpexpect.NewBinder(conf.Handler)
	} else if conf.RequestHandler != nil {
		client.Transport = httpexpect.NewFastBinder(conf.RequestHandler)
	}

	testConfiguration := httpexpect.Config{
		BaseURL:  baseURL,
		Client:   client,
		Reporter: httpexpect.NewAssertReporter(t),
	}

	if conf.Debug {
		testConfiguration.Printers = []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		}
	}

	return httpexpect.WithConfig(testConfiguration)
}
