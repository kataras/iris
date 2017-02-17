// Package httpexpect helps with end-to-end HTTP and REST API testing.
//
// Usage examples
//
// See example directory:
//  - https://godoc.org/github.com/iris-contrib/httpexpect/_examples
//  - https://github.com/iris-contrib/httpexpect/tree/master/_examples
//
// Communication mode
//
// There are two common ways to test API with httpexpect:
//  - start HTTP server and instruct httpexpect to use HTTP client for communication
//  - don't start server and instruct httpexpect to invoke http handler directly
//
// The second approach works only if the server is a Go module and its handler can
// be imported in tests.
//
// Concrete behaviour is determined by Client implementation passed to Config struct.
// If you're using http.Client, set its Transport field (http.RoundTriper) to one of
// the following:
//  1. default (nil) - use HTTP transport from net/http (you should start server)
//  2. httpexpect.Binder - invoke given http.Handler directly
//
// Note that http handler can be usually obtained from http framework you're using.
// E.g., echo framework provides either http.Handler.
//
// You can also provide your own implementation of RequestFactory (creates http.Request),
// or Client (gets http.Request and returns http.Response).
//
// If you're starting server from tests, it's very handy to use net/http/httptest.
//
// Value equality
//
// Whenever values are checked for equality in httpexpect, they are converted
// to "canonical form":
//  - structs are converted to map[string]interface{}
//  - type aliases are removed
//  - numeric types are converted to float64
//  - non-nil interfaces pointing to nil slices and maps are replaced with
//    nil interfaces
//
// This is equivalent to subsequently json.Marshal() and json.Unmarshal() the value
// and currently is implemented so.
//
// Failure handling
//
// When some check fails, failure is reported. If non-fatal failures are used
// (see Reporter interface), execution is continued and instance that was checked
// is marked as failed.
//
// If specific instance is marked as failed, all subsequent checks are ignored
// for this instance and for any child instances retrieved after failure.
//
// Example:
//  array := NewArray(NewAssertReporter(t), []interface{}{"foo", 123})
//
//  e0 := array.Element(0)  // success
//  e1 := array.Element(1)  // success
//
//  s0 := e0.String()  // success
//  s1 := e1.String()  // failure; e1 and s1 are marked as failed, e0 and s0 are not
//
//  s0.Equal("foo")    // success
//  s1.Equal("bar")    // this check is ignored because s1 is marked as failed
package httpexpect

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"golang.org/x/net/publicsuffix"
)

// Expect is a toplevel object that contains user Config and allows
// to construct Request objects.
type Expect struct {
	config   Config
	builders []func(*Request)
}

// Config contains various settings.
type Config struct {
	// BaseURL is a URL to prepended to all request. My be empty. If
	// non-empty, trailing slash is allowed but not required and is
	// appended automatically.
	BaseURL string

	// RequestFactory is used to pass in a custom *http.Request generation func.
	// May be nil.
	//
	// You can use DefaultRequestFactory, or provide custom implementation.
	// Useful for Google App Engine testing for example.
	RequestFactory RequestFactory

	// Client is used to send http.Request and receive http.Response.
	// Should not be nil.
	//
	// You can use http.DefaultClient or http.Client, or provide
	// custom implementation.
	Client Client

	// Reporter is used to report failures.
	// Should not be nil.
	//
	// You can use AssertReporter, RequireReporter (they use testify),
	// or testing.TB, or provide custom implementation.
	Reporter Reporter

	// Printers are used to print requests and responses.
	// May be nil.
	//
	// You can use CompactPrinter, DebugPrinter, CurlPrinter, or provide
	// custom implementation.
	//
	// You can also use builtin printers with alternative Logger if
	// you're happy with their format, but want to send logs somewhere
	// else instead of testing.TB.
	Printers []Printer
}

// RequestFactory is used to create all http.Request objects.
// aetest.Instance from the Google App Engine implements this interface.
type RequestFactory interface {
	NewRequest(method, urlStr string, body io.Reader) (*http.Request, error)
}

// Client is used to send http.Request and receive http.Response.
// http.Client, Binder, and FastBinder implement this interface.
type Client interface {
	// Do sends request and returns response.
	Do(*http.Request) (*http.Response, error)
}

// Printer is used to print requests and responses.
// CompactPrinter, DebugPrinter, and CurlPrinter implement this interface.
type Printer interface {
	// Request is called before request is sent.
	Request(*http.Request)

	// Response is called after response is received.
	Response(*http.Response, time.Duration)
}

// Logger is used as output backend for Printer.
// testing.TB implements this interface.
type Logger interface {
	// Logf writes message to log.
	Logf(fmt string, args ...interface{})
}

// Reporter is used to report failures.
// testing.TB, AssertReporter, and RequireReporter implement this interface.
type Reporter interface {
	// Errorf reports failure.
	// Allowed to return normally or terminate test using t.FailNow().
	Errorf(message string, args ...interface{})
}

// LoggerReporter combines Logger and Reporter interfaces.
type LoggerReporter interface {
	Logger
	Reporter
}

// DefaultRequestFactory is the default RequestFactory implementation which just
// calls http.NewRequest.
type DefaultRequestFactory struct{}

// NewRequest implements RequestFactory.NewRequest.
func (DefaultRequestFactory) NewRequest(
	method, urlStr string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, urlStr, body)
}

// New returns a new Expect object.
//
// baseURL specifies URL to prepended to all request. My be empty. If non-empty,
// trailing slash is allowed but not required and is appended automatically.
//
// New is a shorthand for WithConfig. It uses:
//  - CompactPrinter as Printer, with testing.TB as Logger
//  - AssertReporter as Reporter
//  - DefaultRequestFactory as RequestFactory
//
// Client is set to a default client with a non-nil Jar:
//  &http.Client{
//      Jar: httpexpect.NewJar(),
//  }
//
// Example:
//  func TestSomething(t *testing.T) {
//      e := httpexpect.New(t, "http://example.com/")
//
//      e.GET("/path").
//          Expect().
//          Status(http.StatusOK)
//  }
func New(t LoggerReporter, baseURL string) *Expect {
	return WithConfig(Config{
		BaseURL:  baseURL,
		Reporter: NewAssertReporter(t),
		Printers: []Printer{
			NewCompactPrinter(t),
		},
	})
}

// WithConfig returns a new Expect object with given config.
//
// Reporter should not be nil.
//
// If RequestFactory is nil, it's set to a DefaultRequestFactory instance.
//
// If Client is nil, it's set to a default client with a non-nil Jar:
//  &http.Client{
//      Jar: httpexpect.NewJar(),
//  }
//
// Example:
//  func TestSomething(t *testing.T) {
//      e := httpexpect.WithConfig(httpexpect.Config{
//          BaseURL:  "http://example.com/",
//          Client:   &http.Client{
//              Transport: httpexpect.NewBinder(myHandler()),
//              Jar:       httpexpect.NewJar(),
//          },
//          Reporter: httpexpect.NewAssertReporter(t),
//          Printers: []httpexpect.Printer{
//              httpexpect.NewCurlPrinter(t),
//              httpexpect.NewDebugPrinter(t, true)
//          },
//      })
//
//      e.GET("/path").
//          Expect().
//          Status(http.StatusOK)
//  }
func WithConfig(config Config) *Expect {
	if config.Reporter == nil {
		panic("config.Reporter is nil")
	}
	if config.RequestFactory == nil {
		config.RequestFactory = DefaultRequestFactory{}
	}
	if config.Client == nil {
		config.Client = &http.Client{
			Jar: NewJar(),
		}
	}
	return &Expect{
		config:   config,
		builders: nil,
	}
}

// NewJar returns a new http.CookieJar.
//
// Returned jar is implemented in net/http/cookiejar. PublicSuffixList is
// implemented in golang.org/x/net/publicsuffix.
//
// Note that this jar ignores cookies when request url is empty.
func NewJar() http.CookieJar {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		panic(err)
	}
	return jar
}

// Builder returns a copy of Expect instance with given builder attached to it.
// Returned copy contains all previously attached builders plus a new one.
// Builders are invoked from Request method, after constructing every new request.
//
// Example:
//  e := httpexpect.New(t, "http://example.com")
//
//  token := e.POST("/login").WithForm(Login{"ford", "betelgeuse7"}).
//      Expect().
//      Status(http.StatusOK).JSON().Object().Value("token").String().Raw()
//
//  auth := e.Builder(func (req *httpexpect.Request) {
//      req.WithHeader("Authorization", "Bearer "+token)
//  })
//
//  auth.GET("/restricted").
//     Expect().
//     Status(http.StatusOK)
func (e *Expect) Builder(builder func(*Request)) *Expect {
	ret := *e
	ret.builders = append(e.builders, builder)
	return &ret
}

// Request returns a new Request object.
// Arguments a similar to NewRequest.
// After creating request, all builders attached to Expect object are invoked.
// See Builder.
func (e *Expect) Request(method, path string, pathargs ...interface{}) *Request {
	req := NewRequest(e.config, method, path, pathargs...)

	for _, builder := range e.builders {
		builder(req)
	}

	return req
}

// OPTIONS is a shorthand for e.Request("OPTIONS", path, pathargs...).
func (e *Expect) OPTIONS(path string, pathargs ...interface{}) *Request {
	return e.Request("OPTIONS", path, pathargs...)
}

// HEAD is a shorthand for e.Request("HEAD", path, pathargs...).
func (e *Expect) HEAD(path string, pathargs ...interface{}) *Request {
	return e.Request("HEAD", path, pathargs...)
}

// GET is a shorthand for e.Request("GET", path, pathargs...).
func (e *Expect) GET(path string, pathargs ...interface{}) *Request {
	return e.Request("GET", path, pathargs...)
}

// POST is a shorthand for e.Request("POST", path, pathargs...).
func (e *Expect) POST(path string, pathargs ...interface{}) *Request {
	return e.Request("POST", path, pathargs...)
}

// PUT is a shorthand for e.Request("PUT", path, pathargs...).
func (e *Expect) PUT(path string, pathargs ...interface{}) *Request {
	return e.Request("PUT", path, pathargs...)
}

// PATCH is a shorthand for e.Request("PATCH", path, pathargs...).
func (e *Expect) PATCH(path string, pathargs ...interface{}) *Request {
	return e.Request("PATCH", path, pathargs...)
}

// DELETE is a shorthand for e.Request("DELETE", path, pathargs...).
func (e *Expect) DELETE(path string, pathargs ...interface{}) *Request {
	return e.Request("DELETE", path, pathargs...)
}

// Value is a shorthand for NewValue(e.config.Reporter, value).
func (e *Expect) Value(value interface{}) *Value {
	return NewValue(e.config.Reporter, value)
}

// Object is a shorthand for NewObject(e.config.Reporter, value).
func (e *Expect) Object(value map[string]interface{}) *Object {
	return NewObject(e.config.Reporter, value)
}

// Array is a shorthand for NewArray(e.config.Reporter, value).
func (e *Expect) Array(value []interface{}) *Array {
	return NewArray(e.config.Reporter, value)
}

// String is a shorthand for NewString(e.config.Reporter, value).
func (e *Expect) String(value string) *String {
	return NewString(e.config.Reporter, value)
}

// Number is a shorthand for NewNumber(e.config.Reporter, value).
func (e *Expect) Number(value float64) *Number {
	return NewNumber(e.config.Reporter, value)
}

// Boolean is a shorthand for NewBoolean(e.config.Reporter, value).
func (e *Expect) Boolean(value bool) *Boolean {
	return NewBoolean(e.config.Reporter, value)
}
