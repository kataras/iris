package context

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/json-iterator/go"
	"github.com/microcosm-cc/bluemonday"
	"github.com/monoculum/formam"
	"github.com/russross/blackfriday"

	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/memstore"
)

type (
	// BodyDecoder is an interface which any struct can implement in order to customize the decode action
	// from ReadJSON and ReadXML
	//
	// Trivial example of this could be:
	// type User struct { Username string }
	//
	// func (u *User) Decode(data []byte) error {
	//	  return json.Unmarshal(data, u)
	// }
	//
	// the 'context.ReadJSON/ReadXML(&User{})' will call the User's
	// Decode option to decode the request body
	//
	// Note: This is totally optionally, the default decoders
	// for ReadJSON is the encoding/json and for ReadXML is the encoding/xml.
	BodyDecoder interface {
		Decode(data []byte) error
	}

	// Unmarshaler is the interface implemented by types that can unmarshal any raw data
	// TIP INFO: Any v object which implements the BodyDecoder can be override the unmarshaler.
	Unmarshaler interface {
		Unmarshal(data []byte, v interface{}) error
	}

	// UnmarshalerFunc a shortcut for the Unmarshaler interface
	//
	// See 'Unmarshaler' and 'BodyDecoder' for more.
	UnmarshalerFunc func(data []byte, v interface{}) error
)

// Unmarshal parses the X-encoded data and stores the result in the value pointed to by v.
// Unmarshal uses the inverse of the encodings that Marshal uses, allocating maps,
// slices, and pointers as necessary.
func (u UnmarshalerFunc) Unmarshal(data []byte, v interface{}) error {
	return u(data, v)
}

// RequestParams is a key string - value string storage which context's request params should implement.
// RequestValues is for communication between middleware, RequestParams cannot be changed, are setted at the routing
// time, stores the dynamic named parameters, can be empty if the route is static.
type RequestParams struct {
	store memstore.Store
}

// Set shouldn't be used as a local storage, context's values store
// is the local storage, not params.
func (r *RequestParams) Set(key, value string) {
	r.store.Set(key, value)
}

// Visit accepts a visitor which will be filled
// by the key-value params.
func (r *RequestParams) Visit(visitor func(key string, value string)) {
	r.store.Visit(func(k string, v interface{}) {
		visitor(k, v.(string)) // always string here.
	})
}

// Get returns a path parameter's value based on its route's dynamic path key.
func (r RequestParams) Get(key string) string {
	return r.store.GetString(key)
}

// GetInt returns the param's value as int, based on its key.
func (r RequestParams) GetInt(key string) (int, error) {
	return r.store.GetInt(key)
}

// GetInt64 returns the user's value as int64, based on its key.
func (r RequestParams) GetInt64(key string) (int64, error) {
	return r.store.GetInt64(key)
}

// GetDecoded returns the url-query-decoded user's value based on its key.
func (r RequestParams) GetDecoded(key string) string {
	return DecodeQuery(DecodeQuery(r.Get(key)))
}

// GetIntUnslashed same as Get but it removes the first slash if found.
// Usage: Get an id from a wildcard path.
//
// Returns -1 with an error if the parameter couldn't be found.
func (r RequestParams) GetIntUnslashed(key string) (int, error) {
	v := r.Get(key)
	if v != "" {
		if len(v) > 1 {
			if v[0] == '/' {
				v = v[1:]
			}
		}
		return strconv.Atoi(v)

	}

	return -1, memstore.ErrIntParse.Format(v)
}

// Len returns the full length of the parameters.
func (r RequestParams) Len() int {
	return r.store.Len()
}

// Context is the midle-man server's "object" for the clients.
//
// A New context is being acquired from a sync.Pool on each connection.
// The Context is the most important thing on the iris's http flow.
//
// Developers send responses to the client's request through a Context.
// Developers get request information from the client's request a Context.
//
// This context is an implementation of the context.Context sub-package.
// context.Context is very extensible and developers can override
// its methods if that is actually needed.
type Context interface {

	// BeginRequest is executing once for each request
	// it should prepare the (new or acquired from pool) context's fields for the new request.
	//
	// To follow the iris' flow, developer should:
	// 1. reset handlers to nil
	// 2. reset values to empty
	// 3. reset sessions to nil
	// 4. reset response writer to the http.ResponseWriter
	// 5. reset request to the *http.Request
	// and any other optional steps, depends on dev's application type.
	BeginRequest(http.ResponseWriter, *http.Request)
	// EndRequest is executing once after a response to the request was sent and this context is useless or released.
	//
	// To follow the iris' flow, developer should:
	// 1. flush the response writer's result
	// 2. release the response writer
	// and any other optional steps, depends on dev's application type.
	EndRequest()

	// ResponseWriter returns an http.ResponseWriter compatible response writer, as expected.
	ResponseWriter() ResponseWriter
	// ResetResponseWriter should change or upgrade the Context's ResponseWriter.
	ResetResponseWriter(ResponseWriter)

	// Request returns the original *http.Request, as expected.
	Request() *http.Request

	// Do calls the SetHandlers(handlers)
	// and executes the first handler,
	// handlers should not be empty.
	//
	// It's used by the router, developers may use that
	// to replace and execute handlers immediately.
	Do(Handlers)

	// AddHandler can add handler(s)
	// to the current request in serve-time,
	// these handlers are not persistenced to the router.
	//
	// Router is calling this function to add the route's handler.
	// If AddHandler called then the handlers will be inserted
	// to the end of the already-defined route's handler.
	//
	AddHandler(...Handler)
	// SetHandlers replaces all handlers with the new.
	SetHandlers(Handlers)
	// Handlers keeps tracking of the current handlers.
	Handlers() Handlers

	// HandlerIndex sets the current index of the
	// current context's handlers chain.
	// If -1 passed then it just returns the
	// current handler index without change the current index.rns that index, useless return value.
	//
	// Look Handlers(), Next() and StopExecution() too.
	HandlerIndex(n int) (currentIndex int)
	// HandlerName returns the current handler's name, helpful for debugging.
	HandlerName() string
	// Next calls all the next handler from the handlers chain,
	// it should be used inside a middleware.
	//
	// Note: Custom context should override this method in order to be able to pass its own context.Context implementation.
	Next()
	// NextHandler returns(but it is NOT executes) the next handler from the handlers chain.
	//
	// Use .Skip() to skip this handler if needed to execute the next of this returning handler.
	NextHandler() Handler
	// Skip skips/ignores the next handler from the handlers chain,
	// it should be used inside a middleware.
	Skip()
	// StopExecution if called then the following .Next calls are ignored.
	StopExecution()
	// IsStopped checks and returns true if the current position of the Context is 255,
	// means that the StopExecution() was called.
	IsStopped() bool

	//  +------------------------------------------------------------+
	//  | Current "user/request" storage                             |
	//  | and share information between the handlers - Values().     |
	//  | Save and get named path parameters - Params()              |
	//  +------------------------------------------------------------+

	// Params returns the current url's named parameters key-value storage.
	// Named path parameters are being saved here.
	// This storage, as the whole Context, is per-request lifetime.
	Params() *RequestParams

	// Values returns the current "user" storage.
	// Named path parameters and any optional data can be saved here.
	// This storage, as the whole Context, is per-request lifetime.
	//
	// You can use this function to Set and Get local values
	// that can be used to share information between handlers and middleware.
	Values() *memstore.Store
	// Translate is the i18n (localization) middleware's function,
	// it calls the Get("translate") to return the translated value.
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/miscellaneous/i18n
	Translate(format string, args ...interface{}) string

	//  +------------------------------------------------------------+
	//  | Path, Host, Subdomain, IP, Headers etc...                  |
	//  +------------------------------------------------------------+

	// Method returns the request.Method, the client's http method to the server.
	Method() string
	// Path returns the full request path,
	// escaped if EnablePathEscape config field is true.
	Path() string
	// RequestPath returns the full request path,
	// based on the 'escape'.
	RequestPath(escape bool) string

	// Host returns the host part of the current url.
	Host() string
	// Subdomain returns the subdomain of this request, if any.
	// Note that this is a fast method which does not cover all cases.
	Subdomain() (subdomain string)
	// RemoteAddr tries to parse and return the real client's request IP.
	//
	// Based on allowed headers names that can be modified from Configuration.RemoteAddrHeaders.
	//
	// If parse based on these headers fail then it will return the Request's `RemoteAddr` field
	// which is filled by the server before the HTTP handler.
	//
	// Look `Configuration.RemoteAddrHeaders`,
	//      `Configuration.WithRemoteAddrHeader(...)`,
	//      `Configuration.WithoutRemoteAddrHeader(...)` for more.
	RemoteAddr() string
	// GetHeader returns the request header's value based on its name.
	GetHeader(name string) string
	// IsAjax returns true if this request is an 'ajax request'( XMLHttpRequest)
	//
	// There is no a 100% way of knowing that a request was made via Ajax.
	// You should never trust data coming from the client, they can be easily overcome by spoofing.
	//
	// Note that "X-Requested-With" Header can be modified by any client(because of "X-"),
	// so don't rely on IsAjax for really serious stuff,
	// try to find another way of detecting the type(i.e, content type),
	// there are many blogs that describe these problems and provide different kind of solutions,
	// it's always depending on the application you're building,
	// this is the reason why this `IsAjax`` is simple enough for general purpose use.
	//
	// Read more at: https://developer.mozilla.org/en-US/docs/AJAX
	// and https://xhr.spec.whatwg.org/
	IsAjax() bool

	//  +------------------------------------------------------------+
	//  | Response Headers helpers                                   |
	//  +------------------------------------------------------------+

	// Header adds a header to the response writer.
	Header(name string, value string)

	// ContentType sets the response writer's header key "Content-Type" to the 'cType'.
	ContentType(cType string)
	// GetContentType returns the response writer's header value of "Content-Type"
	// which may, setted before with the 'ContentType'.
	GetContentType() string

	// StatusCode sets the status code header to the response.
	// Look .GetStatusCode too.
	StatusCode(statusCode int)
	// GetStatusCode returns the current status code of the response.
	// Look StatusCode too.
	GetStatusCode() int

	// Redirect redirect sends a redirect response the client
	// accepts 2 parameters string and an optional int
	// first parameter is the url to redirect
	// second parameter is the http status should send, default is 302 (StatusFound),
	// you can set it to 301 (Permant redirect), if that's nessecery
	Redirect(urlToRedirect string, statusHeader ...int)

	//  +------------------------------------------------------------+
	//  | Various Request and Post Data                              |
	//  +------------------------------------------------------------+

	// URLParam returns the get parameter from a request , if any.
	URLParam(name string) string
	// URLParamInt returns the url query parameter as int value from a request,
	// returns an error if parse failed.
	URLParamInt(name string) (int, error)
	// URLParamInt64 returns the url query parameter as int64 value from a request,
	// returns an error if parse failed.
	URLParamInt64(name string) (int64, error)
	// URLParams returns a map of GET query parameters separated by comma if more than one
	// it returns an empty map if nothing found.
	URLParams() map[string]string

	// FormValue returns a single form value by its name/key
	FormValue(name string) string
	// FormValues returns all post data values with their keys
	// form data, get, post & put query arguments
	//
	// NOTE: A check for nil is necessary.
	FormValues() map[string][]string
	// PostValue returns a form's only-post value by its name,
	// same as Request.PostFormValue.
	PostValue(name string) string
	// FormFile returns the first file for the provided form key.
	// FormFile calls ctx.Request.ParseMultipartForm and ParseForm if necessary.
	//
	// same as Request.FormFile.
	FormFile(key string) (multipart.File, *multipart.FileHeader, error)

	//  +------------------------------------------------------------+
	//  | Custom HTTP Errors                                         |
	//  +------------------------------------------------------------+

	// NotFound emits an error 404 to the client, using the specific custom error error handler.
	// Note that you may need to call ctx.StopExecution() if you don't want the next handlers
	// to be executed. Next handlers are being executed on iris because you can alt the
	// error code and change it to a more specific one, i.e
	// users := app.Party("/users")
	// users.Done(func(ctx context.Context){ if ctx.StatusCode() == 400 { /*  custom error code for /users */ }})
	NotFound()

	//  +------------------------------------------------------------+
	//  | Body Readers                                               |
	//  +------------------------------------------------------------+

	// SetMaxRequestBodySize sets a limit to the request body size
	// should be called before reading the request body from the client.
	SetMaxRequestBodySize(limitOverBytes int64)

	// UnmarshalBody reads the request's body and binds it to a value or pointer of any type
	// Examples of usage: context.ReadJSON, context.ReadXML.
	UnmarshalBody(v interface{}, unmarshaler Unmarshaler) error
	// ReadJSON reads JSON from request's body and binds it to a value of any json-valid type.
	ReadJSON(jsonObject interface{}) error
	// ReadXML reads XML from request's body and binds it to a value of any xml-valid type.
	ReadXML(xmlObject interface{}) error
	// ReadForm binds the formObject  with the form data
	// it supports any kind of struct.
	ReadForm(formObject interface{}) error

	//  +------------------------------------------------------------+
	//  | Body (raw) Writers                                         |
	//  +------------------------------------------------------------+

	// Write writes the data to the connection as part of an HTTP reply.
	//
	// If WriteHeader has not yet been called, Write calls
	// WriteHeader(http.StatusOK) before writing the data. If the Header
	// does not contain a Content-Type line, Write adds a Content-Type set
	// to the result of passing the initial 512 bytes of written data to
	// DetectContentType.
	//
	// Depending on the HTTP protocol version and the client, calling
	// Write or WriteHeader may prevent future reads on the
	// Request.Body. For HTTP/1.x requests, handlers should read any
	// needed request body data before writing the response. Once the
	// headers have been flushed (due to either an explicit Flusher.Flush
	// call or writing enough data to trigger a flush), the request body
	// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
	// handlers to continue to read the request body while concurrently
	// writing the response. However, such behavior may not be supported
	// by all HTTP/2 clients. Handlers should read before writing if
	// possible to maximize compatibility.
	Write(body []byte) (int, error)
	// Writef formats according to a format specifier and writes to the response.
	//
	// Returns the number of bytes written and any write error encountered.
	Writef(format string, args ...interface{}) (int, error)
	// WriteString writes a simple string to the response.
	//
	// Returns the number of bytes written and any write error encountered.
	WriteString(body string) (int, error)
	// WriteWithExpiration like Write but it sends with an expiration datetime
	// which is refreshed every package-level `StaticCacheDuration` field.
	WriteWithExpiration(body []byte, modtime time.Time) (int, error)
	// StreamWriter registers the given stream writer for populating
	// response body.
	//
	// Access to context's and/or its' members is forbidden from writer.
	//
	// This function may be used in the following cases:
	//
	//     * if response body is too big (more than iris.LimitRequestBodySize(if setted)).
	//     * if response body is streamed from slow external sources.
	//     * if response body must be streamed to the client in chunks.
	//     (aka `http server push`).
	//
	// receives a function which receives the response writer
	// and returns false when it should stop writing, otherwise true in order to continue
	StreamWriter(writer func(w io.Writer) bool)

	//  +------------------------------------------------------------+
	//  | Body Writers with compression                              |
	//  +------------------------------------------------------------+
	// ClientSupportsGzip retruns true if the client supports gzip compression.
	ClientSupportsGzip() bool
	// WriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
	// returns the number of bytes written and an error ( if the client doesn' supports gzip compression)
	//
	// This function writes temporary gzip contents, the ResponseWriter is untouched.
	WriteGzip(b []byte) (int, error)
	// TryWriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
	// If client does not supprots gzip then the contents are written as they are, uncompressed.
	//
	// This function writes temporary gzip contents, the ResponseWriter is untouched.
	TryWriteGzip(b []byte) (int, error)
	// GzipResponseWriter converts the current response writer into a response writer
	// which when its .Write called it compress the data to gzip and writes them to the client.
	//
	// Can be also disabled with its .Disable and .ResetBody to rollback to the usual response writer.
	GzipResponseWriter() *GzipResponseWriter
	// Gzip enables or disables (if enabled before) the gzip response writer,if the client
	// supports gzip compression, so the following response data will
	// be sent as compressed gzip data to the client.
	Gzip(enable bool)

	//  +------------------------------------------------------------+
	//  | Rich Body Content Writers/Renderers                        |
	//  +------------------------------------------------------------+

	// ViewLayout sets the "layout" option if and when .View
	// is being called afterwards, in the same request.
	// Useful when need to set or/and change a layout based on the previous handlers in the chain.
	//
	// Note that the 'layoutTmplFile' argument can be setted to iris.NoLayout || view.NoLayout
	// to disable the layout for a specific view render action,
	// it disables the engine's configuration's layout property.
	//
	// Look .ViewData and .View too.
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/view/context-view-data/
	ViewLayout(layoutTmplFile string)

	// ViewData saves one or more key-value pair in order to be passed if and when .View
	// is being called afterwards, in the same request.
	// Useful when need to set or/and change template data from previous hanadlers in the chain.
	//
	// If .View's "binding" argument is not nil and it's not a type of map
	// then these data are being ignored, binding has the priority, so the main route's handler can still decide.
	// If binding is a map or context.Map then these data are being added to the view data
	// and passed to the template.
	//
	// After .View, the data are not destroyed, in order to be re-used if needed (again, in the same request as everything else),
	// to clear the view data, developers can call:
	// ctx.Set(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey(), nil)
	//
	// If 'key' is empty then the value is added as it's (struct or map) and developer is unable to add other value.
	//
	// Look .ViewLayout and .View too.
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/view/context-view-data/
	ViewData(key string, value interface{})

	// GetViewData returns the values registered by `context#ViewData`.
	// The return value is `map[string]interface{}`, this means that
	// if a custom struct registered to ViewData then this function
	// will try to parse it to map, if failed then the return value is nil
	// A check for nil is always a good practise if different
	// kind of values or no data are registered via `ViewData`.
	//
	// Similarly to `viewData := ctx.Values().Get("iris.viewData")` or
	// `viewData := ctx.Values().Get(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey())`.
	GetViewData() map[string]interface{}

	// View renders templates based on the adapted view engines.
	// First argument accepts the filename, relative to the view engine's Directory,
	// i.e: if directory is "./templates" and want to render the "./templates/users/index.html"
	// then you pass the "users/index.html" as the filename argument.
	//
	// Look: .ViewData and .ViewLayout too.
	//
	// Examples: https://github.com/kataras/iris/tree/master/_examples/view/
	View(filename string) error

	// Binary writes out the raw bytes as binary data.
	Binary(data []byte) (int, error)
	// Text writes out a string as plain text.
	Text(text string) (int, error)
	// HTML writes out a string as text/html.
	HTML(htmlContents string) (int, error)
	// JSON marshals the given interface object and writes the JSON response.
	JSON(v interface{}, options ...JSON) (int, error)
	// JSONP marshals the given interface object and writes the JSON response.
	JSONP(v interface{}, options ...JSONP) (int, error)
	// XML marshals the given interface object and writes the XML response.
	XML(v interface{}, options ...XML) (int, error)
	// Markdown parses the markdown to html and renders to client.
	Markdown(markdownB []byte, options ...Markdown) (int, error)

	//  +------------------------------------------------------------+
	//  | Serve files                                                |
	//  +------------------------------------------------------------+

	// ServeContent serves content, headers are autoset
	// receives three parameters, it's low-level function, instead you can use .ServeFile(string,bool)/SendFile(string,string)
	//
	// You can define your own "Content-Type" header also, after this function call
	// Doesn't implements resuming (by range), use ctx.SendFile instead
	ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error
	// ServeFile serves a view file, to send a file ( zip for example) to the client you should use the SendFile(serverfilename,clientfilename)
	// receives two parameters
	// filename/path (string)
	// gzipCompression (bool)
	//
	// You can define your own "Content-Type" header also, after this function call
	// This function doesn't implement resuming (by range), use ctx.SendFile instead
	//
	// Use it when you want to serve css/js/... files to the client, for bigger files and 'force-download' use the SendFile.
	ServeFile(filename string, gzipCompression bool) error
	// SendFile sends file for force-download to the client
	//
	// Use this instead of ServeFile to 'force-download' bigger files to the client.
	SendFile(filename string, destinationName string) error

	//  +------------------------------------------------------------+
	//  | Cookies                                                    |
	//  +------------------------------------------------------------+

	// SetCookie adds a cookie
	SetCookie(cookie *http.Cookie)
	// SetCookieKV adds a cookie, receives just a name(string) and a value(string)
	//
	// If you use this method, it expires at 2 hours
	// use ctx.SetCookie or http.SetCookie if you want to change more fields.
	SetCookieKV(name, value string)
	// GetCookie returns cookie's value by it's name
	// returns empty string if nothing was found.
	GetCookie(name string) string
	// RemoveCookie deletes a cookie by it's name.
	RemoveCookie(name string)
	// VisitAllCookies takes a visitor which loops
	// on each (request's) cookies' name and value.
	VisitAllCookies(visitor func(name string, value string))

	// MaxAge returns the "cache-control" request header's value
	// seconds as int64
	// if header not found or parse failed then it returns -1.
	MaxAge() int64

	//  +------------------------------------------------------------+
	//  | Advanced: Response Recorder and Transactions               |
	//  +------------------------------------------------------------+

	// Record transforms the context's basic and direct responseWriter to a ResponseRecorder
	// which can be used to reset the body, reset headers, get the body,
	// get & set the status code at any time and more.
	Record()
	// Recorder returns the context's ResponseRecorder
	// if not recording then it starts recording and returns the new context's ResponseRecorder
	Recorder() *ResponseRecorder
	// IsRecording returns the response recorder and a true value
	// when the response writer is recording the status code, body, headers and so on,
	// else returns nil and false.
	IsRecording() (*ResponseRecorder, bool)

	// BeginTransaction starts a scoped transaction.
	//
	// You can search third-party articles or books on how Business Transaction works (it's quite simple, especially here).
	//
	// Note that this is unique and new
	// (=I haver never seen any other examples or code in Golang on this subject, so far, as with the most of iris features...)
	// it's not covers all paths,
	// such as databases, this should be managed by the libraries you use to make your database connection,
	// this transaction scope is only for context's response.
	// Transactions have their own middleware ecosystem also, look iris.go:UseTransaction.
	//
	// See https://github.com/kataras/iris/tree/master/_examples/ for more
	BeginTransaction(pipe func(t *Transaction))
	// SkipTransactions if called then skip the rest of the transactions
	// or all of them if called before the first transaction
	SkipTransactions()
	// TransactionsSkipped returns true if the transactions skipped or canceled at all.
	TransactionsSkipped() bool

	// Exec calls the framewrok's ServeCtx
	// based on this context but with a changed method and path
	// like it was requested by the user, but it is not.
	//
	// Offline means that the route is registered to the iris and have all features that a normal route has
	// BUT it isn't available by browsing, its handlers executed only when other handler's context call them
	// it can validate paths, has sessions, path parameters and all.
	//
	// You can find the Route by app.GetRoute("theRouteName")
	// you can set a route name as: myRoute := app.Get("/mypath", handler)("theRouteName")
	// that will set a name to the route and returns its RouteInfo instance for further usage.
	//
	// It doesn't changes the global state, if a route was "offline" it remains offline.
	//
	// app.None(...) and app.GetRoutes().Offline(route)/.Online(route, method)
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/routing/route-state
	//
	// User can get the response by simple using rec := ctx.Recorder(); rec.Body()/rec.StatusCode()/rec.Header().
	//
	// Context's Values and the Session are kept in order to be able to communicate via the result route.
	//
	// It's for extreme use cases, 99% of the times will never be useful for you.
	Exec(method string, path string)

	// Application returns the iris app instance which belongs to this context.
	// Worth to notice that this function returns an interface
	// of the Application, which contains methods that are safe
	// to be executed at serve-time. The full app's fields
	// and methods are not available here for the developer's safety.
	Application() Application

	//  +--------------------------------------------------------------+
	//  | https://github.com/golang/net/blob/master/context/context.go |                                     |
	//  +--------------------------------------------------------------+

	// Deadline returns the time when work done on behalf of this context
	// should be canceled.  Deadline returns ok==false when no deadline is
	// set.  Successive calls to Deadline return the same results.
	Deadline() (deadline time.Time, ok bool)

	// Done returns a channel that's closed when work done on behalf of this
	// context should be canceled.  Done may return nil if this context can
	// never be canceled.  Successive calls to Done return the same value.
	//
	// WithCancel arranges for Done to be closed when cancel is called;
	// WithDeadline arranges for Done to be closed when the deadline
	// expires; WithTimeout arranges for Done to be closed when the timeout
	// elapses.
	//
	// Done is provided for use in select statements:
	//
	//  // Stream generates values with DoSomething and sends them to out
	//  // until DoSomething returns an error or ctx.Done is closed.
	//  func Stream(ctx context.Context, out chan<- Value) error {
	//  	for {
	//  		v, err := DoSomething(ctx)
	//  		if err != nil {
	//  			return err
	//  		}
	//  		select {
	//  		case <-ctx.Done():
	//  			return ctx.Err()
	//  		case out <- v:
	//  		}
	//  	}
	//  }
	//
	// See http://blog.golang.org/pipelines for more examples of how to use
	// a Done channel for cancelation.
	Done() <-chan struct{}

	// Err returns a non-nil error value after Done is closed.  Err returns
	// Canceled if the context was canceled or DeadlineExceeded if the
	// context's deadline passed.  No other values for Err are defined.
	// After Done is closed, successive calls to Err return the same value.
	Err() error

	// Value returns the value associated with this context for key, or nil
	// if no value is associated with key.  Successive calls to Value with
	// the same key returns the same result.
	//
	// Use context values only for request-scoped data that transits
	// processes and API boundaries, not for passing optional parameters to
	// functions.
	//
	// A key identifies a specific value in a Context.  Functions that wish
	// to store values in Context typically allocate a key in a global
	// variable then use that key as the argument to context.WithValue and
	// Context.Value.  A key can be any type that supports equality;
	// packages should define keys as an unexported type to avoid
	// collisions.
	//
	// Packages that define a Context key should provide type-safe accessors
	// for the values stores using that key:
	//
	// 	// Package user defines a User type that's stored in Contexts.
	// 	package user
	//
	// 	import "golang.org/x/net/context"
	//
	// 	// User is the type of value stored in the Contexts.
	// 	type User struct {...}
	//
	// 	// key is an unexported type for keys defined in this package.
	// 	// This prevents collisions with keys defined in other packages.
	// 	type key int
	//
	// 	// userKey is the key for user.User values in Contexts.  It is
	// 	// unexported; clients use user.NewContext and user.FromContext
	// 	// instead of using this key directly.
	// 	var userKey key = 0
	//
	// 	// NewContext returns a new Context that carries value u.
	// 	func NewContext(ctx context.Context, u *User) context.Context {
	// 		return context.WithValue(ctx, userKey, u)
	// 	}
	//
	// 	// FromContext returns the User value stored in ctx, if any.
	// 	func FromContext(ctx context.Context) (*User, bool) {
	// 		u, ok := ctx.Value(userKey).(*User)
	// 		return u, ok
	// 	}
	Value(key interface{}) interface{}
}

// Next calls all the next handler from the handlers chain,
// it should be used inside a middleware.
func Next(ctx Context) {
	if ctx.IsStopped() {
		return
	}
	if n, handlers := ctx.HandlerIndex(-1)+1, ctx.Handlers(); n < len(handlers) {
		ctx.HandlerIndex(n)
		handlers[n](ctx)
	}
}

// Do calls the SetHandlers(handlers)
// and executes the first handler,
// handlers should not be empty.
//
// It's used by the router, developers may use that
// to replace and execute handlers immediately.
func Do(ctx Context, handlers Handlers) {
	if len(handlers) > 0 {
		ctx.SetHandlers(handlers)
		handlers[0](ctx)
	}
}

// LimitRequestBodySize is a middleware which sets a request body size limit
// for all next handlers in the chain.
var LimitRequestBodySize = func(maxRequestBodySizeBytes int64) Handler {
	return func(ctx Context) {
		ctx.SetMaxRequestBodySize(maxRequestBodySizeBytes)
		ctx.Next()
	}
}

// Gzip is a middleware which enables writing
// using gzip compression, if client supports.
var Gzip = func(ctx Context) {
	ctx.Gzip(true)
	ctx.Next()
}

// Map is just a shortcut of the map[string]interface{}.
type Map map[string]interface{}

//  +------------------------------------------------------------+
//  | Context Implementation                                     |
//  +------------------------------------------------------------+

type context struct {
	// the http.ResponseWriter wrapped by custom writer
	writer ResponseWriter
	// the original http.Request
	request *http.Request
	// the local key-value storage
	params RequestParams  // url named parameters
	values memstore.Store // generic storage, middleware communication

	// the underline application app
	app Application
	// the route's handlers
	handlers Handlers
	// the current position of the handler's chain
	currentHandlerIndex int
}

// NewContext returns the default, internal, context implementation.
// You may use this function to embed the default context implementation
// to a custom one.
//
// This context is received by the context pool.
func NewContext(app Application) Context {
	return &context{app: app}
}

// BeginRequest is executing once for each request
// it should prepare the (new or acquired from pool) context's fields for the new request.
//
// To follow the iris' flow, developer should:
// 1. reset handlers to nil
// 2. reset store to empty
// 3. reset sessions to nil
// 4. reset response writer to the http.ResponseWriter
// 5. reset request to the *http.Request
// and any other optional steps, depends on dev's application type.
func (ctx *context) BeginRequest(w http.ResponseWriter, r *http.Request) {
	ctx.handlers = nil           // will be filled by router.Serve/HTTP
	ctx.values = ctx.values[0:0] // >>      >>     by context.Values().Set
	ctx.params.store = ctx.params.store[0:0]
	ctx.request = r
	ctx.currentHandlerIndex = 0
	ctx.writer = AcquireResponseWriter()
	ctx.writer.BeginResponse(w)
}

// EndRequest is executing once after a response to the request was sent and this context is useless or released.
//
// To follow the iris' flow, developer should:
// 1. flush the response writer's result
// 2. release the response writer
// and any other optional steps, depends on dev's application type.
func (ctx *context) EndRequest() {
	if ctx.GetStatusCode() >= 400 &&
		!ctx.Application().ConfigurationReadOnly().GetDisableAutoFireStatusCode() {
		// author's note:
		// if recording, the error handler can handle
		// the rollback and remove any response written before,
		// we don't have to do anything here, written is -1 when Recording
		// because we didn't flush the response yet
		// if !recording  then check if the previous handler didn't send something
		// to the client
		if ctx.writer.Written() == -1 {
			ctx.Application().FireErrorCode(ctx)
		}
	}

	ctx.writer.FlushResponse()
	ctx.writer.EndResponse()
}

// ResponseWriter returns an http.ResponseWriter compatible response writer, as expected.
func (ctx *context) ResponseWriter() ResponseWriter {
	return ctx.writer
}

// ResetResponseWriter should change or upgrade the context's ResponseWriter.
func (ctx *context) ResetResponseWriter(newResponseWriter ResponseWriter) {
	ctx.writer = newResponseWriter
}

// Request returns the original *http.Request, as expected.
func (ctx *context) Request() *http.Request {
	return ctx.request
}

// Do calls the SetHandlers(handlers)
// and executes the first handler,
// handlers should not be empty.
//
// It's used by the router, developers may use that
// to replace and execute handlers immediately.
func (ctx *context) Do(handlers Handlers) {
	ctx.handlers = handlers
	ctx.handlers[0](ctx)
}

// AddHandler can add handler(s)
// to the current request in serve-time,
// these handlers are not persistenced to the router.
//
// Router is calling this function to add the route's handler.
// If AddHandler called then the handlers will be inserted
// to the end of the already-defined route's handler.
//
func (ctx *context) AddHandler(handlers ...Handler) {
	ctx.handlers = append(ctx.handlers, handlers...)
}

// SetHandlers replaces all handlers with the new.
func (ctx *context) SetHandlers(handlers Handlers) {
	ctx.handlers = handlers
}

// Handlers keeps tracking of the current handlers.
func (ctx *context) Handlers() Handlers {
	return ctx.handlers
}

// HandlerIndex sets the current index of the
// current context's handlers chain.
// If -1 passed then it just returns the
// current handler index without change the current index.rns that index, useless return value.
//
// Look Handlers(), Next() and StopExecution() too.
func (ctx *context) HandlerIndex(n int) (currentIndex int) {
	if n < 0 || n > len(ctx.handlers)-1 {
		return ctx.currentHandlerIndex
	}

	ctx.currentHandlerIndex = n
	return n
}

// HandlerName returns the current handler's name, helpful for debugging.
func (ctx *context) HandlerName() string {
	return runtime.FuncForPC(reflect.ValueOf(ctx.handlers[ctx.currentHandlerIndex]).Pointer()).Name()
}

// Do sets the handler index to zero, executes the first handler
// and the rest of the Handlers if ctx.Next() was called.
// func (ctx *context) Do() {
// 	ctx.currentHandlerIndex = 0
// 	ctx.handlers[0](ctx) // it calls this *context
// } // -> replaced with inline on router.go

// Next calls all the next handler from the handlers chain,
// it should be used inside a middleware.
//
// Note: Custom context should override this method in order to be able to pass its own context.context implementation.
func (ctx *context) Next() { // or context.Next(ctx)
	Next(ctx)
}

// NextHandler returns, but it doesn't executes, the next handler from the handlers chain.
//
// Use .Skip() to skip this handler if needed to execute the next of this returning handler.
func (ctx *context) NextHandler() Handler {
	if ctx.IsStopped() {
		return nil
	}
	nextIndex := ctx.currentHandlerIndex + 1
	// check if it has a next middleware
	if nextIndex < len(ctx.handlers) {
		return ctx.handlers[nextIndex]
	}
	return nil
}

// Skip skips/ignores the next handler from the handlers chain,
// it should be used inside a middleware.
func (ctx *context) Skip() {
	ctx.HandlerIndex(ctx.currentHandlerIndex + 1)
}

const stopExecutionIndex = -1 // I don't set to a max value because we want to be able to reuse the handlers even if stopped with .Skip

// StopExecution if called then the following .Next calls are ignored.
func (ctx *context) StopExecution() {
	ctx.currentHandlerIndex = stopExecutionIndex
}

// IsStopped checks and returns true if the current position of the context is -1,
// means that the StopExecution() was called.
func (ctx *context) IsStopped() bool {
	return ctx.currentHandlerIndex == stopExecutionIndex
}

//  +------------------------------------------------------------+
//  | Current "user/request" storage                             |
//  | and share information between the handlers - Values().     |
//  | Save and get named path parameters - Params()              |
//  +------------------------------------------------------------+

// Params returns the current url's named parameters key-value storage.
// Named path parameters are being saved here.
// This storage, as the whole context, is per-request lifetime.
func (ctx *context) Params() *RequestParams {
	return &ctx.params
}

// Values returns the current "user" storage.
// Named path parameters and any optional data can be saved here.
// This storage, as the whole context, is per-request lifetime.
//
// You can use this function to Set and Get local values
// that can be used to share information between handlers and middleware.
func (ctx *context) Values() *memstore.Store {
	return &ctx.values
}

// Translate is the i18n (localization) middleware's function,
// it calls the Get("translate") to return the translated value.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/miscellaneous/i18n
func (ctx *context) Translate(format string, args ...interface{}) string {
	if cb, ok := ctx.values.Get(ctx.Application().ConfigurationReadOnly().GetTranslateFunctionContextKey()).(func(format string, args ...interface{}) string); ok {
		return cb(format, args...)
	}

	return ""
}

//  +------------------------------------------------------------+
//  | Path, Host, Subdomain, IP, Headers etc...                  |
//  +------------------------------------------------------------+

// Method returns the request.Method, the client's http method to the server.
func (ctx *context) Method() string {
	return ctx.request.Method
}

// Path returns the full request path,
// escaped if EnablePathEscape config field is true.
func (ctx *context) Path() string {
	return ctx.RequestPath(ctx.Application().ConfigurationReadOnly().GetEnablePathEscape())
}

// DecodeQuery returns the uri parameter as url (string)
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.QueryUnescape
func DecodeQuery(path string) string {
	if path == "" {
		return ""
	}
	encodedPath, err := url.QueryUnescape(path)
	if err != nil {
		return path
	}
	return encodedPath
}

// DecodeURL returns the decoded uri
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.Parse
func DecodeURL(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return u.String()
}

// RequestPath returns the full request path,
// based on the 'escape'.
func (ctx *context) RequestPath(escape bool) string {
	if escape {
		return DecodeQuery(ctx.request.URL.EscapedPath())
	}
	return ctx.request.URL.Path // RawPath returns empty, requesturi can be used instead also.
}

// PathPrefixMap accepts a map of string and a handler.
// The key of "m" is the key, which is the prefix, regular expressions are not valid.
// The value of "m" is the handler that will be executed if HasPrefix(context.Path).
// func (ctx *context) PathPrefixMap(m map[string]context.Handler) bool {
// 	path := ctx.Path()
// 	for k, v := range m {
// 		if strings.HasPrefix(path, k) {
// 			v(ctx)
// 			return true
// 		}
// 	}
// 	return false
// } no, it will not work because map is a random peek data structure.

// Host returns the host part of the current url.
func (ctx *context) Host() string {
	h := ctx.request.URL.Host
	if h == "" {
		h = ctx.request.Host
	}
	return h
}

// Subdomain returns the subdomain of this request, if any.
// Note that this is a fast method which does not cover all cases.
func (ctx *context) Subdomain() (subdomain string) {
	host := ctx.Host()
	if index := strings.IndexByte(host, '.'); index > 0 {
		subdomain = host[0:index]
	}

	// listening on mydomain.com:80
	// subdomain = mydomain, but it's wrong, it should return ""
	vhost := ctx.Application().ConfigurationReadOnly().GetVHost()
	if strings.Contains(vhost, subdomain) { // then it's not subdomain
		return ""
	}

	return
}

// RemoteAddr tries to parse and return the real client's request IP.
//
// Based on allowed headers names that can be modified from Configuration.RemoteAddrHeaders.
//
// If parse based on these headers fail then it will return the Request's `RemoteAddr` field
// which is filled by the server before the HTTP handler.
//
// Look `Configuration.RemoteAddrHeaders`,
//      `Configuration.WithRemoteAddrHeader(...)`,
//      `Configuration.WithoutRemoteAddrHeader(...)` for more.
func (ctx *context) RemoteAddr() string {

	remoteHeaders := ctx.Application().ConfigurationReadOnly().GetRemoteAddrHeaders()

	for headerName, enabled := range remoteHeaders {
		if enabled {
			headerValue := ctx.GetHeader(headerName)
			// exception needed for 'X-Forwarded-For' only , if enabled.
			if headerName == "X-Forwarded-For" {
				idx := strings.IndexByte(headerValue, ',')
				if idx >= 0 {
					headerValue = headerValue[0:idx]
				}
			}

			realIP := strings.TrimSpace(headerValue)
			if realIP != "" {
				return realIP
			}
		}
	}

	addr := strings.TrimSpace(ctx.request.RemoteAddr)
	if addr != "" {
		// if addr has port use the net.SplitHostPort otherwise(error occurs) take as it is
		if ip, _, err := net.SplitHostPort(addr); err == nil {
			return ip
		}
	}

	return addr
}

// GetHeader returns the request header's value based on its name.
func (ctx *context) GetHeader(name string) string {
	return ctx.request.Header.Get(name)
}

// IsAjax returns true if this request is an 'ajax request'( XMLHttpRequest)
//
// There is no a 100% way of knowing that a request was made via Ajax.
// You should never trust data coming from the client, they can be easily overcome by spoofing.
//
// Note that "X-Requested-With" Header can be modified by any client(because of "X-"),
// so don't rely on IsAjax for really serious stuff,
// try to find another way of detecting the type(i.e, content type),
// there are many blogs that describe these problems and provide different kind of solutions,
// it's always depending on the application you're building,
// this is the reason why this `IsAjax`` is simple enough for general purpose use.
//
// Read more at: https://developer.mozilla.org/en-US/docs/AJAX
// and https://xhr.spec.whatwg.org/
func (ctx *context) IsAjax() bool {
	return ctx.GetHeader("X-Requested-With") == "XMLHttpRequest"
}

//  +------------------------------------------------------------+
//  | Response Headers helpers                                   |
//  +------------------------------------------------------------+

// Header adds a header to the response, if value is empty
// it removes the header by its name.
func (ctx *context) Header(name string, value string) {
	if value == "" {
		ctx.writer.Header().Del(name)
		return
	}
	ctx.writer.Header().Add(name, value)
}

const contentTypeHeaderKey = "Content-Type"

// ContentType sets the response writer's header key "Content-Type" to the 'cType'.
func (ctx *context) ContentType(cType string) {
	if cType == "" {
		return
	}

	// 1. if it's path or a filename or an extension,
	// then take the content type from that
	if strings.Contains(cType, ".") {
		ext := filepath.Ext(cType)
		cType = mime.TypeByExtension(ext)
	}
	// if doesn't contain a charset already then append it
	if !strings.Contains(cType, "charset") {
		if cType != contentBinaryHeaderValue {
			charset := ctx.Application().ConfigurationReadOnly().GetCharset()
			cType += "; charset=" + charset
		}
	}
	ctx.writer.Header().Set(contentTypeHeaderKey, cType)
}

// GetContentType returns the response writer's header value of "Content-Type"
// which may, setted before with the 'ContentType'.
func (ctx *context) GetContentType() string {
	return ctx.writer.Header().Get(contentTypeHeaderKey)
}

// StatusCode sets the status code header to the response.
// Look .GetStatusCode & .FireStatusCode too.
//
// Remember, the last one before .Write matters except recorder and transactions.
func (ctx *context) StatusCode(statusCode int) {
	ctx.writer.WriteHeader(statusCode)
}

// NotFound emits an error 404 to the client, using the specific custom error error handler.
// Note that you may need to call ctx.StopExecution() if you don't want the next handlers
// to be executed. Next handlers are being executed on iris because you can alt the
// error code and change it to a more specific one, i.e
// users := app.Party("/users")
// users.Done(func(ctx context.Context){ if ctx.StatusCode() == 400 { /*  custom error code for /users */ }})
func (ctx *context) NotFound() {
	ctx.StatusCode(http.StatusNotFound)
}

// GetStatusCode returns the current status code of the response.
// Look StatusCode too.
func (ctx *context) GetStatusCode() int {
	return ctx.writer.StatusCode()
}

//  +------------------------------------------------------------+
//  | Various Request and Post Data                              |
//  +------------------------------------------------------------+

// URLParam returns the get parameter from a request , if any.
func (ctx *context) URLParam(name string) string {
	return ctx.request.URL.Query().Get(name)
}

// URLParamInt returns the url query parameter as int value from a request,
// returns an error if parse failed.
func (ctx *context) URLParamInt(name string) (int, error) {
	return strconv.Atoi(ctx.URLParam(name))
}

// URLParamInt64 returns the url query parameter as int64 value from a request,
// returns an error if parse failed.
func (ctx *context) URLParamInt64(name string) (int64, error) {
	return strconv.ParseInt(ctx.URLParam(name), 10, 64)
}

// URLParams returns a map of GET query parameters separated by comma if more than one
// it returns an empty map if nothing found.
func (ctx *context) URLParams() map[string]string {
	values := map[string]string{}

	q := ctx.request.URL.Query()
	if q != nil {
		for k, v := range q {
			values[k] = strings.Join(v, ",")
		}
	}

	return values
}

func (ctx *context) askParseForm() error {
	if ctx.request.Form == nil {
		if err := ctx.request.ParseForm(); err != nil {
			return err
		}
	}
	return nil
}

// FormValue returns a single form value by its name/key
func (ctx *context) FormValue(name string) string {
	return ctx.request.FormValue(name)
}

// FormValues returns all post data values with their keys
// form data, get, post & put query arguments
//
// NOTE: A check for nil is necessary.
func (ctx *context) FormValues() map[string][]string {
	//  we skip the check of multipart form, takes too much memory, if user wants it can do manually now.
	if err := ctx.askParseForm(); err != nil {
		return nil
	}
	return ctx.request.Form // nothing more to do, it's already contains both query and post & put args.

}

// PostValue returns a form's only-post value by its name,
// same as Request.PostFormValue.
func (ctx *context) PostValue(name string) string {
	return ctx.request.PostFormValue(name)
}

// FormFile returns the first file for the provided form key.
// FormFile calls ctx.request.ParseMultipartForm and ParseForm if necessary.
//
// same as Request.FormFile.
func (ctx *context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return ctx.request.FormFile(key)
}

// Redirect redirect sends a redirect response the client
// accepts 2 parameters string and an optional int
// first parameter is the url to redirect
// second parameter is the http status should send, default is 302 (StatusFound),
// you can set it to 301 (Permant redirect), if that's nessecery
func (ctx *context) Redirect(urlToRedirect string, statusHeader ...int) {
	ctx.StopExecution()

	httpStatus := http.StatusFound // a 'temporary-redirect-like' which works better than for our purpose
	if len(statusHeader) > 0 && statusHeader[0] > 0 {
		httpStatus = statusHeader[0]
	}

	http.Redirect(ctx.writer, ctx.request, urlToRedirect, httpStatus)
}

//  +------------------------------------------------------------+
//  | Body Readers                                               |
//  +------------------------------------------------------------+

// SetMaxRequestBodySize sets a limit to the request body size
// should be called before reading the request body from the client.
func (ctx *context) SetMaxRequestBodySize(limitOverBytes int64) {
	ctx.request.Body = http.MaxBytesReader(ctx.writer, ctx.request.Body, limitOverBytes)
}

// UnmarshalBody reads the request's body and binds it to a value or pointer of any type
// Examples of usage: context.ReadJSON, context.ReadXML.
func (ctx *context) UnmarshalBody(v interface{}, unmarshaler Unmarshaler) error {
	if ctx.request.Body == nil {
		return errors.New("unmarshal: empty body")
	}

	rawData, err := ioutil.ReadAll(ctx.request.Body)
	if err != nil {
		return err
	}

	if ctx.Application().ConfigurationReadOnly().GetDisableBodyConsumptionOnUnmarshal() {
		// * remember, Request.Body has no Bytes(), we have to consume them first
		// and after re-set them to the body, this is the only solution.
		ctx.request.Body = ioutil.NopCloser(bytes.NewBuffer(rawData))
	}

	// check if the v contains its own decode
	// in this case the v should be a pointer also,
	// but this is up to the user's custom Decode implementation*
	//
	// See 'BodyDecoder' for more
	if decoder, isDecoder := v.(BodyDecoder); isDecoder {
		return decoder.Decode(rawData)
	}

	// check if v is already a pointer, if yes then pass as it's
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		return unmarshaler.Unmarshal(rawData, v)
	}
	// finally, if the v doesn't contains a self-body decoder and it's not a pointer
	// use the custom unmarshaler to bind the body
	return unmarshaler.Unmarshal(rawData, &v)
}

func (ctx *context) shouldOptimize() bool {
	return ctx.Application().ConfigurationReadOnly().GetEnableOptimizations()
}

// ReadJSON reads JSON from request's body and binds it to a value of any json-valid type.
func (ctx *context) ReadJSON(jsonObject interface{}) error {
	var unmarshaler = json.Unmarshal
	if ctx.shouldOptimize() {
		unmarshaler = jsoniter.Unmarshal
	}
	return ctx.UnmarshalBody(jsonObject, UnmarshalerFunc(unmarshaler))
}

// ReadXML reads XML from request's body and binds it to a value of any xml-valid type.
func (ctx *context) ReadXML(xmlObject interface{}) error {
	return ctx.UnmarshalBody(xmlObject, UnmarshalerFunc(xml.Unmarshal))
}

var (
	errReadBody = errors.New("while trying to read %s from the request body. Trace %s")
)

// ReadForm binds the formObject  with the form data
// it supports any kind of struct.
func (ctx *context) ReadForm(formObject interface{}) error {
	values := ctx.FormValues()
	if values == nil {
		return errors.New("An empty form passed on ReadForm")
	}

	// or dec := formam.NewDecoder(&formam.DecoderOptions{TagName: "form"})
	// somewhere at the app level. I did change the tagName to "form"
	// inside its source code, so it's not needed for now.
	return errReadBody.With(formam.Decode(values, formObject))
}

//  +------------------------------------------------------------+
//  | Body (raw) Writers                                         |
//  +------------------------------------------------------------+

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
func (ctx *context) Write(rawBody []byte) (int, error) {
	return ctx.writer.Write(rawBody)
}

// Writef formats according to a format specifier and writes to the response.
//
// Returns the number of bytes written and any write error encountered.
func (ctx *context) Writef(format string, a ...interface{}) (n int, err error) {
	return ctx.writer.Writef(format, a...)
}

// WriteString writes a simple string to the response.
//
// Returns the number of bytes written and any write error encountered.
func (ctx *context) WriteString(body string) (n int, err error) {
	return ctx.writer.WriteString(body)
}

var (
	// StaticCacheDuration expiration duration for INACTIVE file handlers, it's the only one global configuration
	// which can be changed.
	StaticCacheDuration = 20 * time.Second

	lastModifiedHeaderKey       = "Last-Modified"
	ifModifiedSinceHeaderKey    = "If-Modified-Since"
	contentDispositionHeaderKey = "Content-Disposition"
	cacheControlHeaderKey       = "Cache-Control"
	contentEncodingHeaderKey    = "Content-Encoding"
	acceptEncodingHeaderKey     = "Accept-Encoding"
	varyHeaderKey               = "Vary"
)

// staticCachePassed checks the IfModifiedSince header and
// returns true if (client-side) duration has expired
func (ctx *context) staticCachePassed(modtime time.Time) bool {
	if t, err := time.Parse(ctx.Application().ConfigurationReadOnly().GetTimeFormat(), ctx.GetHeader(ifModifiedSinceHeaderKey)); err == nil && modtime.Before(t.Add(StaticCacheDuration)) {
		ctx.writer.Header().Del(contentTypeHeaderKey)
		ctx.writer.Header().Del(contentLengthHeaderKey)
		ctx.StatusCode(http.StatusNotModified)
		return true
	}
	return false
}

// WriteWithExpiration like Write but it sends with an expiration datetime
// which is refreshed every package-level `StaticCacheDuration` field.
func (ctx *context) WriteWithExpiration(body []byte, modtime time.Time) (int, error) {

	if ctx.staticCachePassed(modtime) {
		return 0, nil
	}

	modtimeFormatted := modtime.UTC().Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
	ctx.Header(lastModifiedHeaderKey, modtimeFormatted)

	return ctx.writer.Write(body)
}

// StreamWriter registers the given stream writer for populating
// response body.
//
// Access to context's and/or its' members is forbidden from writer.
//
// This function may be used in the following cases:
//
//     * if response body is too big (more than iris.LimitRequestBodySize(if setted)).
//     * if response body is streamed from slow external sources.
//     * if response body must be streamed to the client in chunks.
//     (aka `http server push`).
//
// receives a function which receives the response writer
// and returns false when it should stop writing, otherwise true in order to continue
func (ctx *context) StreamWriter(writer func(w io.Writer) bool) {
	w := ctx.writer
	notifyClosed := w.CloseNotify()
	for {
		select {
		// response writer forced to close, exit.
		case <-notifyClosed:
			return
		default:
			shouldContinue := writer(w)
			w.Flush()
			if !shouldContinue {
				return
			}
		}
	}
}

//  +------------------------------------------------------------+
//  | Body Writers with compression                              |
//  +------------------------------------------------------------+

// ClientSupportsGzip retruns true if the client supports gzip compression.
func (ctx *context) ClientSupportsGzip() bool {
	if h := ctx.GetHeader(acceptEncodingHeaderKey); h != "" {
		for _, v := range strings.Split(h, ";") {
			if strings.Contains(v, "gzip") { // we do Contains because sometimes browsers has the q=, we don't use it atm. || strings.Contains(v,"deflate"){
				return true
			}
		}
	}
	return false
}

var (
	errClientDoesNotSupportGzip = errors.New("client doesn't supports gzip compression")
)

// WriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
// returns the number of bytes written and an error ( if the client doesn' supports gzip compression)
//
// This function writes temporary gzip contents, the ResponseWriter is untouched.
func (ctx *context) WriteGzip(b []byte) (int, error) {
	if ctx.ClientSupportsGzip() {
		ctx.writer.Header().Add(varyHeaderKey, acceptEncodingHeaderKey)

		gzipWriter := acquireGzipWriter(ctx.writer)
		defer releaseGzipWriter(gzipWriter)
		n, err := gzipWriter.Write(b)

		if err == nil {
			ctx.Header(contentEncodingHeaderKey, "gzip")
		} // else write the contents as it is? no let's create a new func for this
		return n, err
	}

	return 0, errClientDoesNotSupportGzip
}

// TryWriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
// If client does not supprots gzip then the contents are written as they are, uncompressed.
//
// This function writes temporary gzip contents, the ResponseWriter is untouched.
func (ctx *context) TryWriteGzip(b []byte) (int, error) {
	n, err := ctx.WriteGzip(b)
	if err != nil {
		// check if the error came from gzip not allowed and not the writer itself
		if _, ok := err.(*errors.Error); ok {
			// client didn't supported gzip, write them uncompressed:
			return ctx.writer.Write(b)
		}
	}
	return n, err
}

// GzipResponseWriter converts the current response writer into a response writer
// which when its .Write called it compress the data to gzip and writes them to the client.
//
// Can be also disabled with its .Disable and .ResetBody to rollback to the usual response writer.
func (ctx *context) GzipResponseWriter() *GzipResponseWriter {
	// if it's already a gzip response writer then just return it
	if gzipResWriter, ok := ctx.writer.(*GzipResponseWriter); ok {
		return gzipResWriter
	}
	// if it's not acquire a new from a pool
	// and register that as the ctx.ResponseWriter.
	gzipResWriter := AcquireGzipResponseWriter()
	gzipResWriter.BeginGzipResponse(ctx.writer)
	ctx.ResetResponseWriter(gzipResWriter)
	return gzipResWriter
}

// Gzip enables or disables (if enabled before) the gzip response writer,if the client
// supports gzip compression, so the following response data will
// be sent as compressed gzip data to the client.
func (ctx *context) Gzip(enable bool) {
	if enable {
		if ctx.ClientSupportsGzip() {
			_ = ctx.GzipResponseWriter()
		}
	} else {
		if gzipResWriter, ok := ctx.writer.(*GzipResponseWriter); ok {
			gzipResWriter.Disable()
		}
	}
}

//  +------------------------------------------------------------+
//  | Rich Body Content Writers/Renderers                        |
//  +------------------------------------------------------------+

const (
	// NoLayout to disable layout for a particular template file
	NoLayout = "iris.nolayout"
)

// ViewLayout sets the "layout" option if and when .View
// is being called afterwards, in the same request.
// Useful when need to set or/and change a layout based on the previous handlers in the chain.
//
// Note that the 'layoutTmplFile' argument can be setted to iris.NoLayout || view.NoLayout || context.NoLayout
// to disable the layout for a specific view render action,
// it disables the engine's configuration's layout property.
//
// Look .ViewData and .View too.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/view/context-view-data/
func (ctx *context) ViewLayout(layoutTmplFile string) {
	ctx.values.Set(ctx.Application().ConfigurationReadOnly().GetViewLayoutContextKey(), layoutTmplFile)
}

// ViewData saves one or more key-value pair in order to be passed if and when .View
// is being called afterwards, in the same request.
// Useful when need to set or/and change template data from previous hanadlers in the chain.
//
// If .View's "binding" argument is not nil and it's not a type of map
// then these data are being ignored, binding has the priority, so the main route's handler can still decide.
// If binding is a map or context.Map then these data are being added to the view data
// and passed to the template.
//
// After .View, the data are not destroyed, in order to be re-used if needed (again, in the same request as everything else),
// to clear the view data, developers can call:
// ctx.Set(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey(), nil)
//
// If 'key' is empty then the value is added as it's (struct or map) and developer is unable to add other value.
//
// Look .ViewLayout and .View too.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/view/context-view-data/
func (ctx *context) ViewData(key string, value interface{}) {
	viewDataContextKey := ctx.Application().ConfigurationReadOnly().GetViewDataContextKey()
	if key == "" {
		ctx.values.Set(viewDataContextKey, value)
		return
	}

	v := ctx.values.Get(viewDataContextKey)
	if v == nil {
		ctx.values.Set(viewDataContextKey, Map{key: value})
		return
	}

	if data, ok := v.(map[string]interface{}); ok {
		data[key] = value
	} else if data, ok := v.(Map); ok {
		data[key] = value
	}
}

// GetViewData returns the values registered by `context#ViewData`.
// The return value is `map[string]interface{}`, this means that
// if a custom struct registered to ViewData then this function
// will try to parse it to map, if failed then the return value is nil
// A check for nil is always a good practise if different
// kind of values or no data are registered via `ViewData`.
//
// Similarly to `viewData := ctx.Values().Get("iris.viewData")` or
// `viewData := ctx.Values().Get(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey())`.
func (ctx *context) GetViewData() map[string]interface{} {
	viewDataContextKey := ctx.Application().ConfigurationReadOnly().GetViewDataContextKey()
	v := ctx.Values().Get(viewDataContextKey)

	// if no values found, then return nil
	if v == nil {
		return nil
	}

	// if struct, convert it to map[string]interface{}
	if structs.IsStruct(v) {
		return structs.Map(v)
	}

	// if pure map[string]interface{}
	if viewData, ok := v.(map[string]interface{}); ok {
		return viewData
	}

	// if context#Map
	if viewData, ok := v.(Map); ok {
		return viewData
	}

	// if failure, then return nil
	return nil
}

// View renders templates based on the adapted view engines.
// First argument accepts the filename, relative to the view engine's Directory,
// i.e: if directory is "./templates" and want to render the "./templates/users/index.html"
// then you pass the "users/index.html" as the filename argument.
//
// Look: .ViewData and .ViewLayout too.
//
// Examples: https://github.com/kataras/iris/tree/master/_examples/view/
func (ctx *context) View(filename string) error {
	ctx.ContentType(contentHTMLHeaderValue)
	cfg := ctx.Application().ConfigurationReadOnly()

	layout := ctx.values.GetString(cfg.GetViewLayoutContextKey())
	bindingData := ctx.values.Get(cfg.GetViewDataContextKey())

	err := ctx.Application().View(ctx.writer, filename, layout, bindingData)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.StopExecution()
	}

	return err
}

const (
	// contentBinaryHeaderValue header value for binary data.
	contentBinaryHeaderValue = "application/octet-stream"
	// contentHTMLHeaderValue is the  string of text/html response header's content type value.
	contentHTMLHeaderValue = "text/html"
	// ContentJSON header value for JSON data.
	contentJSONHeaderValue = "application/json"
	// ContentJSONP header value for JSONP & Javascript data.
	contentJavascriptHeaderValue = "application/javascript"
	// contentTextHeaderValue header value for Text data.
	contentTextHeaderValue = "text/plain"
	// contentXMLHeaderValue header value for XML data.
	contentXMLHeaderValue = "text/xml"

	// contentMarkdownHeaderValue custom key/content type, the real is the text/html.
	contentMarkdownHeaderValue = "text/markdown"
)

// Binary writes out the raw bytes as binary data.
func (ctx *context) Binary(data []byte) (int, error) {
	ctx.ContentType(contentBinaryHeaderValue)
	return ctx.Write(data)
}

// Text writes out a string as plain text.
func (ctx *context) Text(text string) (int, error) {
	ctx.ContentType(contentTextHeaderValue)
	return ctx.writer.WriteString(text)
}

// HTML writes out a string as text/html.
func (ctx *context) HTML(htmlContents string) (int, error) {
	ctx.ContentType(contentHTMLHeaderValue)
	return ctx.writer.WriteString(htmlContents)
}

// JSON contains the options for the JSON (Context's) Renderer.
type JSON struct {
	// http-specific
	StreamingJSON bool
	// content-specific
	UnescapeHTML bool
	Indent       string
	Prefix       string
}

// JSONP contains the options for the JSONP (Context's) Renderer.
type JSONP struct {
	// content-specific
	Indent   string
	Callback string
}

// XML contains the options for the XML (Context's) Renderer.
type XML struct {
	// content-specific
	Indent string
	Prefix string
}

// Markdown contains the options for the Markdown (Context's) Renderer.
type Markdown struct {
	// content-specific
	Sanitize bool
}

var (
	newLineB = []byte("\n")
	// the html codes for unescaping
	ltHex = []byte("\\u003c")
	lt    = []byte("<")

	gtHex = []byte("\\u003e")
	gt    = []byte(">")

	andHex = []byte("\\u0026")
	and    = []byte("&")
)

// WriteJSON marshals the given interface object and writes the JSON response to the 'writer'.
// Ignores StatusCode, Gzip, StreamingJSON options.
func WriteJSON(writer io.Writer, v interface{}, options JSON, enableOptimization ...bool) (int, error) {
	var (
		result   []byte
		err      error
		optimize = len(enableOptimization) > 0 && enableOptimization[0]
	)

	if indent := options.Indent; indent != "" {
		marshalIndent := json.MarshalIndent
		if optimize {
			marshalIndent = jsoniter.ConfigCompatibleWithStandardLibrary.MarshalIndent
		}

		result, err = marshalIndent(v, "", indent)
		result = append(result, newLineB...)
	} else {
		marshal := json.Marshal
		if optimize {
			marshal = jsoniter.ConfigCompatibleWithStandardLibrary.Marshal
		}

		result, err = marshal(v)
	}

	if err != nil {
		return 0, err
	}

	if options.UnescapeHTML {
		result = bytes.Replace(result, ltHex, lt, -1)
		result = bytes.Replace(result, gtHex, gt, -1)
		result = bytes.Replace(result, andHex, and, -1)
	}

	if prefix := options.Prefix; prefix != "" {
		result = append([]byte(prefix), result...)
	}

	return writer.Write(result)
}

var defaultJSONOptions = JSON{}

// JSON marshals the given interface object and writes the JSON response to the client.
func (ctx *context) JSON(v interface{}, opts ...JSON) (n int, err error) {
	options := defaultJSONOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	optimize := ctx.shouldOptimize()

	ctx.ContentType(contentJSONHeaderValue)

	if options.StreamingJSON {
		if optimize {
			var jsoniterConfig = jsoniter.Config{
				EscapeHTML:    !options.UnescapeHTML,
				IndentionStep: 4,
			}.Froze()
			enc := jsoniterConfig.NewEncoder(ctx.writer)
			err = enc.Encode(v)
		} else {
			enc := json.NewEncoder(ctx.writer)
			enc.SetEscapeHTML(!options.UnescapeHTML)
			enc.SetIndent(options.Prefix, options.Indent)
			err = enc.Encode(v)
		}

		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError) // it handles the fallback to normal mode here which also removes the gzip headers.
			return 0, err
		}
		return ctx.writer.Written(), err
	}

	n, err = WriteJSON(ctx.writer, v, options, optimize)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

var (
	finishCallbackB = []byte(");")
)

// WriteJSONP marshals the given interface object and writes the JSON response to the writer.
func WriteJSONP(writer io.Writer, v interface{}, options JSONP, enableOptimization ...bool) (int, error) {
	if callback := options.Callback; callback != "" {
		writer.Write([]byte(callback + "("))
		defer writer.Write(finishCallbackB)
	}

	optimize := len(enableOptimization) > 0 && enableOptimization[0]

	if indent := options.Indent; indent != "" {
		marshalIndent := json.MarshalIndent
		if optimize {
			marshalIndent = jsoniter.ConfigCompatibleWithStandardLibrary.MarshalIndent
		}

		result, err := marshalIndent(v, "", indent)
		if err != nil {
			return 0, err
		}
		result = append(result, newLineB...)
		return writer.Write(result)
	}

	marshal := json.Marshal
	if optimize {
		marshal = jsoniter.ConfigCompatibleWithStandardLibrary.Marshal
	}

	result, err := marshal(v)
	if err != nil {
		return 0, err
	}
	return writer.Write(result)
}

var defaultJSONPOptions = JSONP{}

// JSONP marshals the given interface object and writes the JSON response to the client.
func (ctx *context) JSONP(v interface{}, opts ...JSONP) (int, error) {
	options := defaultJSONPOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	ctx.ContentType(contentJavascriptHeaderValue)

	n, err := WriteJSONP(ctx.writer, v, options, ctx.shouldOptimize())
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

// WriteXML marshals the given interface object and writes the XML response to the writer.
func WriteXML(writer io.Writer, v interface{}, options XML) (int, error) {
	if prefix := options.Prefix; prefix != "" {
		writer.Write([]byte(prefix))
	}

	if indent := options.Indent; indent != "" {
		result, err := xml.MarshalIndent(v, "", indent)
		if err != nil {
			return 0, err
		}
		result = append(result, newLineB...)
		return writer.Write(result)
	}

	result, err := xml.Marshal(v)
	if err != nil {
		return 0, err
	}
	return writer.Write(result)
}

var defaultXMLOptions = XML{}

// XML marshals the given interface object and writes the XML response to the client.
func (ctx *context) XML(v interface{}, opts ...XML) (int, error) {
	options := defaultXMLOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	ctx.ContentType(contentXMLHeaderValue)

	n, err := WriteXML(ctx.writer, v, options)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

// WriteMarkdown parses the markdown to html and renders these contents to the writer.
func WriteMarkdown(writer io.Writer, markdownB []byte, options Markdown) (int, error) {
	buf := blackfriday.MarkdownCommon(markdownB)
	if options.Sanitize {
		buf = bluemonday.UGCPolicy().SanitizeBytes(buf)
	}
	return writer.Write(buf)
}

var defaultMarkdownOptions = Markdown{}

// Markdown parses the markdown to html and renders to the client.
func (ctx *context) Markdown(markdownB []byte, opts ...Markdown) (int, error) {
	options := defaultMarkdownOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	ctx.ContentType(contentHTMLHeaderValue)

	n, err := WriteMarkdown(ctx.writer, markdownB, options)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

//  +------------------------------------------------------------+
//  | Serve files                                                |
//  +------------------------------------------------------------+

var (
	errServeContent = errors.New("while trying to serve content to the client. Trace %s")
)

const (
	// contentLengthHeaderKey represents the header["Content-Length"]
	contentLengthHeaderKey = "Content-Length"
)

// ServeContent serves content, headers are autoset
// receives three parameters, it's low-level function, instead you can use .ServeFile(string,bool)/SendFile(string,string)
//
// You can define your own "Content-Type" header also, after this function call
// Doesn't implements resuming (by range), use ctx.SendFile instead
func (ctx *context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error {
	if t, err := time.Parse(ctx.Application().ConfigurationReadOnly().GetTimeFormat(), ctx.GetHeader(ifModifiedSinceHeaderKey)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.writer.Header().Del(contentTypeHeaderKey)
		ctx.writer.Header().Del(contentLengthHeaderKey)
		ctx.StatusCode(http.StatusNotModified)
		return nil
	}

	ctx.ContentType(filename)
	ctx.writer.Header().Set(lastModifiedHeaderKey, modtime.UTC().Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat()))
	ctx.StatusCode(http.StatusOK)
	var out io.Writer
	if gzipCompression && ctx.ClientSupportsGzip() {
		ctx.writer.Header().Add(varyHeaderKey, acceptEncodingHeaderKey)
		ctx.Header(contentEncodingHeaderKey, "gzip")

		gzipWriter := acquireGzipWriter(ctx.writer)
		defer releaseGzipWriter(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.writer
	}
	_, err := io.Copy(out, content)
	return errServeContent.With(err)
}

// ServeFile serves a view file, to send a file ( zip for example) to the client you should use the SendFile(serverfilename,clientfilename)
// receives two parameters
// filename/path (string)
// gzipCompression (bool)
//
// You can define your own "Content-Type" header also, after this function call
// This function doesn't implement resuming (by range), use ctx.SendFile instead
//
// Use it when you want to serve css/js/... files to the client, for bigger files and 'force-download' use the SendFile.
func (ctx *context) ServeFile(filename string, gzipCompression bool) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("%d", 404)
	}
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() {
		filename = path.Join(filename, "index.html")
		f, err = os.Open(filename)
		if err != nil {
			return fmt.Errorf("%d", 404)
		}
		fi, _ = f.Stat()
	}
	return ctx.ServeContent(f, fi.Name(), fi.ModTime(), gzipCompression)

}

// SendFile sends file for force-download to the client
//
// Use this instead of ServeFile to 'force-download' bigger files to the client.
func (ctx *context) SendFile(filename string, destinationName string) error {
	ctx.writer.Header().Set(contentDispositionHeaderKey, "attachment;filename="+destinationName)
	return ctx.ServeFile(filename, false)
}

//  +------------------------------------------------------------+
//  | Cookies, Session and Flashes                               |
//  +------------------------------------------------------------+

// SetCookie adds a cookie
func (ctx *context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.writer, cookie)
}

var (
	// SetCookieKVExpiration is 2 hours by-default
	// you can change it or simple, use the SetCookie for more control.
	SetCookieKVExpiration = time.Duration(120) * time.Minute
)

// SetCookieKV adds a cookie, receives just a name(string) and a value(string)
//
// If you use this method, it expires at 2 hours
// use ctx.SetCookie or http.SetCookie if you want to change more fields.
func (ctx *context) SetCookieKV(name, value string) {
	c := &http.Cookie{}
	c.Name = name
	c.Value = value
	c.HttpOnly = true
	c.Expires = time.Now().Add(SetCookieKVExpiration)
	c.MaxAge = int(SetCookieKVExpiration.Seconds())
	ctx.SetCookie(c)
}

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found.
func (ctx *context) GetCookie(name string) string {
	cookie, err := ctx.request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// RemoveCookie deletes a cookie by it's name.
func (ctx *context) RemoveCookie(name string) {
	c := &http.Cookie{}
	c.Name = name
	c.Value = ""
	c.Path = "/"
	c.HttpOnly = true
	// RFC says 1 second, but let's do it 1 minute to make sure is working
	exp := time.Now().Add(-time.Duration(1) * time.Minute)
	c.Expires = exp
	c.MaxAge = -1
	ctx.SetCookie(c)
	// delete request's cookie also, which is temporary available
	ctx.request.Header.Set("Cookie", "")
}

// VisitAllCookies takes a visitor which loops
// on each (request's) cookies' name and value.
func (ctx *context) VisitAllCookies(visitor func(name string, value string)) {
	for _, cookie := range ctx.request.Cookies() {
		visitor(cookie.Name, cookie.Value)
	}
}

var maxAgeExp = regexp.MustCompile(`maxage=(\d+)`)

// MaxAge returns the "cache-control" request header's value
// seconds as int64
// if header not found or parse failed then it returns -1.
func (ctx *context) MaxAge() int64 {
	header := ctx.GetHeader(cacheControlHeaderKey)
	if header == "" {
		return -1
	}
	m := maxAgeExp.FindStringSubmatch(header)
	if len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			return int64(v)
		}
	}
	return -1
}

//  +------------------------------------------------------------+
//  | Advanced: Response Recorder and Transactions               |
//  +------------------------------------------------------------+

// Record transforms the context's basic and direct responseWriter to a *ResponseRecorder
// which can be used to reset the body, reset headers, get the body,
// get & set the status code at any time and more.
func (ctx *context) Record() {
	if w, ok := ctx.writer.(*responseWriter); ok {
		recorder := AcquireResponseRecorder()
		recorder.BeginRecord(w)
		ctx.ResetResponseWriter(recorder)
	}
}

// Recorder returns the context's ResponseRecorder
// if not recording then it starts recording and returns the new context's ResponseRecorder
func (ctx *context) Recorder() *ResponseRecorder {
	ctx.Record()
	return ctx.writer.(*ResponseRecorder)
}

// IsRecording returns the response recorder and a true value
// when the response writer is recording the status code, body, headers and so on,
// else returns nil and false.
func (ctx *context) IsRecording() (*ResponseRecorder, bool) {
	//NOTE:
	// two return values in order to minimize the if statement:
	// if (Recording) then writer = Recorder()
	// instead we do: recorder,ok = Recording()
	rr, ok := ctx.writer.(*ResponseRecorder)
	return rr, ok
}

// non-detailed error log for transacton unexpected panic
var errTransactionInterrupted = errors.New("transaction interrupted, recovery from panic:\n%s")

// BeginTransaction starts a scoped transaction.
//
// Can't say a lot here because it will take more than 200 lines to write about.
// You can search third-party articles or books on how Business Transaction works (it's quite simple, especially here).
//
// Note that this is unique and new
// (=I haver never seen any other examples or code in Golang on this subject, so far, as with the most of iris features...)
// it's not covers all paths,
// such as databases, this should be managed by the libraries you use to make your database connection,
// this transaction scope is only for context's response.
// Transactions have their own middleware ecosystem also, look iris.go:UseTransaction.
//
// See https://github.com/kataras/iris/tree/master/_examples/ for more
func (ctx *context) BeginTransaction(pipe func(t *Transaction)) {
	// do NOT begin a transaction when the previous transaction has been failed
	// and it was requested scoped or SkipTransactions called manually.
	if ctx.TransactionsSkipped() {
		return
	}

	// start recording in order to be able to control the full response writer
	ctx.Record()

	t := newTransaction(ctx) // it calls this *context, so the overriding with a new pool's New of context.Context wil not work here.
	defer func() {
		if err := recover(); err != nil {
			ctx.Application().Logger().Warn(errTransactionInterrupted.Format(err).Error())
			// complete (again or not , doesn't matters) the scope without loud
			t.Complete(nil)
			// we continue as normal, no need to return here*
		}

		// write the temp contents to the original writer
		t.Context().ResponseWriter().WriteTo(ctx.writer)
		// give back to the transaction the original writer (SetBeforeFlush works this way and only this way)
		// this is tricky but nessecery if we want ctx.FireStatusCode to work inside transactions
		t.Context().ResetResponseWriter(ctx.writer)

	}()

	// run the worker with its context clone inside.
	pipe(t)
}

// skipTransactionsContextKey set this to any value to stop executing next transactions
// it's a context-key in order to be used from anywhere, set it by calling the SkipTransactions()
const skipTransactionsContextKey = "@transictions_skipped"

// SkipTransactions if called then skip the rest of the transactions
// or all of them if called before the first transaction
func (ctx *context) SkipTransactions() {
	ctx.values.Set(skipTransactionsContextKey, 1)
}

// TransactionsSkipped returns true if the transactions skipped or canceled at all.
func (ctx *context) TransactionsSkipped() bool {
	if n, err := ctx.values.GetInt(skipTransactionsContextKey); err == nil && n == 1 {
		return true
	}
	return false
}

// Exec calls the framewrok's ServeCtx
// based on this context but with a changed method and path
// like it was requested by the user, but it is not.
//
// Offline means that the route is registered to the iris and have all features that a normal route has
// BUT it isn't available by browsing, its handlers executed only when other handler's context call them
// it can validate paths, has sessions, path parameters and all.
//
// You can find the Route by app.GetRoute("theRouteName")
// you can set a route name as: myRoute := app.Get("/mypath", handler)("theRouteName")
// that will set a name to the route and returns its RouteInfo instance for further usage.
//
// It doesn't changes the global state, if a route was "offline" it remains offline.
//
// app.None(...) and app.GetRoutes().Offline(route)/.Online(route, method)
//
// Example: https://github.com/kataras/iris/tree/master/_examples/routing/route-state
//
// User can get the response by simple using rec := ctx.Recorder(); rec.Body()/rec.StatusCode()/rec.Header().
//
// context's Values and the Session are kept in order to be able to communicate via the result route.
//
// It's for extreme use cases, 99% of the times will never be useful for you.
func (ctx *context) Exec(method string, path string) {
	if path != "" {
		if method == "" {
			method = "GET"
		}

		// backup the handlers
		backupHandlers := ctx.Handlers()[0:]
		backupPos := ctx.HandlerIndex(-1)

		// backup the request path information
		backupPath := ctx.Path()
		bakcupMethod := ctx.Method()
		// don't backupValues := ctx.Values().ReadOnly()

		// [sessions stays]
		// [values stays]
		// reset handlers
		ctx.SetHandlers(nil)

		req := ctx.Request()
		// set the request to be align with the 'againstRequestPath'
		req.RequestURI = path
		req.URL.Path = path
		req.Method = method
		// execute the route from the (internal) context router
		// this way we keep the sessions and the values
		ctx.Application().ServeHTTPC(ctx)

		// set back the old handlers and the last known index
		ctx.SetHandlers(backupHandlers)
		ctx.HandlerIndex(backupPos)
		// set the request back to its previous state
		req.RequestURI = backupPath
		req.URL.Path = backupPath
		req.Method = bakcupMethod

		// don't fill the values in order to be able to communicate from and to.
		// // fill the values as they were before
		// backupValues.Visit(func(key string, value interface{}) {
		// 	ctx.Values().Set(key, value)
		// })
	}
}

// Application returns the iris app instance which belongs to this context.
// Worth to notice that this function returns an interface
// of the Application, which contains methods that are safe
// to be executed at serve-time. The full app's fields
// and methods are not available here for the developer's safety.
func (ctx *context) Application() Application {
	return ctx.app
}

//  +--------------------------------------------------------------+
//  | https://github.com/golang/net/blob/master/context/context.go |                                     |
//  +--------------------------------------------------------------+

// Deadline returns the time when work done on behalf of this context
// should be canceled.  Deadline returns ok==false when no deadline is
// set.  Successive calls to Deadline return the same results.
func (ctx *context) Deadline() (deadline time.Time, ok bool) {
	return
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled.  Done may return nil if this context can
// never be canceled.  Successive calls to Done return the same value.
//
// WithCancel arranges for Done to be closed when cancel is called;
// WithDeadline arranges for Done to be closed when the deadline
// expires; WithTimeout arranges for Done to be closed when the timeout
// elapses.
//
// Done is provided for use in select statements:
//
//  // Stream generates values with DoSomething and sends them to out
//  // until DoSomething returns an error or ctx.Done is closed.
//  func Stream(ctx context.Context, out chan<- Value) error {
//  	for {
//  		v, err := DoSomething(ctx)
//  		if err != nil {
//  			return err
//  		}
//  		select {
//  		case <-ctx.Done():
//  			return ctx.Err()
//  		case out <- v:
//  		}
//  	}
//  }
//
// See http://blog.golang.org/pipelines for more examples of how to use
// a Done channel for cancelation.
func (ctx *context) Done() <-chan struct{} {
	return nil
}

// Err returns a non-nil error value after Done is closed.  Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed.  No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (ctx *context) Err() error {
	return nil
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key.  Successive calls to Value with
// the same key returns the same result.
//
// Use context values only for request-scoped data that transits
// processes and API boundaries, not for passing optional parameters to
// functions.
//
// A key indentifies a specific value in a context.  Functions that wish
// to store values in context typically allocate a key in a global
// variable then use that key as the argument to context.WithValue and
// context.Value.  A key can be any type that supports equality;
// packages should define keys as an unexported type to avoid
// collisions.
//
// Packages that define a context key should provide type-safe accessors
// for the values stores using that key:
//
// 	// Package user defines a User type that's stored in Contexts.
// 	package user
//
// 	import "golang.org/x/net/context"
//
// 	// User is the type of value stored in the Contexts.
// 	type User struct {...}
//
// 	// key is an unexported type for keys defined in this package.
// 	// This prevents collisions with keys defined in other packages.
// 	type key int
//
// 	// userKey is the key for user.User values in Contexts.  It is
// 	// unexported; clients use user.NewContext and user.FromContext
// 	// instead of using this key directly.
// 	var userKey key = 0
//
// 	// NewContext returns a new context that carries value u.
// 	func NewContext(ctx context.Context, u *User) context.Context {
// 		return context.WithValue(ctx, userKey, u)
// 	}
//
// 	// FromContext returns the User value stored in ctx, if any.
// 	func FromContext(ctx context.Context) (*User, bool) {
// 		u, ok := ctx.Value(userKey).(*User)
// 		return u, ok
// 	}
func (ctx *context) Value(key interface{}) interface{} {
	if key == 0 {
		return ctx.request
	}
	if k, ok := key.(string); ok {
		return ctx.values.GetString(k)
	}
	return nil
}
