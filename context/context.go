package context

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
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
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/v12/core/memstore"

	"github.com/Shopify/goreferrer"
	"github.com/fatih/structs"
	"github.com/iris-contrib/blackfriday"
	"github.com/iris-contrib/schema"
	jsoniter "github.com/json-iterator/go"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/yaml.v3"
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
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-custom-per-type/main.go
	BodyDecoder interface {
		Decode(data []byte) error
	}

	// Unmarshaler is the interface implemented by types that can unmarshal any raw data.
	// TIP INFO: Any pointer to a value which implements the BodyDecoder can be override the unmarshaler.
	Unmarshaler interface {
		Unmarshal(data []byte, outPtr interface{}) error
	}

	// UnmarshalerFunc a shortcut for the Unmarshaler interface
	//
	// See 'Unmarshaler' and 'BodyDecoder' for more.
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-custom-via-unmarshaler/main.go
	UnmarshalerFunc func(data []byte, outPtr interface{}) error
)

// Unmarshal parses the X-encoded data and stores the result in the value pointed to by v.
// Unmarshal uses the inverse of the encodings that Marshal uses, allocating maps,
// slices, and pointers as necessary.
func (u UnmarshalerFunc) Unmarshal(data []byte, v interface{}) error {
	return u(data, v)
}

// Context is the midle-man server's "object" dealing with incoming requests.
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
	// ResetRequest sets the Context's Request,
	// It is useful to store the new request created by a std *http.Request#WithContext() into Iris' Context.
	// Use `ResetRequest` when for some reason you want to make a full
	// override of the *http.Request.
	// Note that: when you just want to change one of each fields you can use the Request() which returns a pointer to Request,
	// so the changes will have affect without a full override.
	// Usage: you use a native http handler which uses the standard "context" package
	// to get values instead of the Iris' Context#Values():
	// r := ctx.Request()
	// stdCtx := context.WithValue(r.Context(), key, val)
	// ctx.ResetRequest(r.WithContext(stdCtx)).
	ResetRequest(r *http.Request)

	// SetCurrentRouteName sets the route's name internally,
	// in order to be able to find the correct current "read-only" Route when
	// end-developer calls the `GetCurrentRoute()` function.
	// It's being initialized by the Router, if you change that name
	// manually nothing really happens except that you'll get other
	// route via `GetCurrentRoute()`.
	// Instead, to execute a different path
	// from this context you should use the `Exec` function
	// or change the handlers via `SetHandlers/AddHandler` functions.
	SetCurrentRouteName(currentRouteName string)
	// GetCurrentRoute returns the current registered "read-only" route that
	// was being registered to this request's path.
	GetCurrentRoute() RouteReadOnly

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
	// current handler index without change the current index.
	//
	// Look Handlers(), Next() and StopExecution() too.
	HandlerIndex(n int) (currentIndex int)
	// Proceed is an alternative way to check if a particular handler
	// has been executed and called the `ctx.Next` function inside it.
	// This is useful only when you run a handler inside
	// another handler. It justs checks for before index and the after index.
	//
	// A usecase example is when you want to execute a middleware
	// inside controller's `BeginRequest` that calls the `ctx.Next` inside it.
	// The Controller looks the whole flow (BeginRequest, method handler, EndRequest)
	// as one handler, so `ctx.Next` will not be reflected to the method handler
	// if called from the `BeginRequest`.
	//
	// Although `BeginRequest` should NOT be used to call other handlers,
	// the `BeginRequest` has been introduced to be able to set
	// common data to all method handlers before their execution.
	// Controllers can accept middleware(s) from the MVC's Application's Router as normally.
	//
	// That said let's see an example of `ctx.Proceed`:
	//
	// var authMiddleware = basicauth.New(basicauth.Config{
	// 	Users: map[string]string{
	// 		"admin": "password",
	// 	},
	// })
	//
	// func (c *UsersController) BeginRequest(ctx iris.Context) {
	// 	if !ctx.Proceed(authMiddleware) {
	// 		ctx.StopExecution()
	// 	}
	// }
	// This Get() will be executed in the same handler as `BeginRequest`,
	// internally controller checks for `ctx.StopExecution`.
	// So it will not be fired if BeginRequest called the `StopExecution`.
	// func(c *UsersController) Get() []models.User {
	//	  return c.Service.GetAll()
	//}
	// Alternative way is `!ctx.IsStopped()` if middleware make use of the `ctx.StopExecution()` on failure.
	Proceed(Handler) bool
	// HandlerName returns the current handler's name, helpful for debugging.
	HandlerName() string
	// HandlerFileLine returns the current running handler's function source file and line information.
	// Useful mostly when debugging.
	HandlerFileLine() (file string, line int)
	// RouteName returns the route name that this handler is running on.
	// Note that it will return empty on not found handlers.
	RouteName() string
	// Next calls all the next handler from the handlers chain,
	// it should be used inside a middleware.
	//
	// Note: Custom context should override this method in order to be able to pass its own context.Context implementation.
	Next()
	// NextOr checks if chain has a next handler, if so then it executes it
	// otherwise it sets a new chain assigned to this Context based on the given handler(s)
	// and executes its first handler.
	//
	// Returns true if next handler exists and executed, otherwise false.
	//
	// Note that if no next handler found and handlers are missing then
	// it sends a Status Not Found (404) to the client and it stops the execution.
	NextOr(handlers ...Handler) bool
	// NextOrNotFound checks if chain has a next handler, if so then it executes it
	// otherwise it sends a Status Not Found (404) to the client and stops the execution.
	//
	// Returns true if next handler exists and executed, otherwise false.
	NextOrNotFound() bool
	// NextHandler returns (it doesn't execute) the next handler from the handlers chain.
	//
	// Use .Skip() to skip this handler if needed to execute the next of this returning handler.
	NextHandler() Handler
	// Skip skips/ignores the next handler from the handlers chain,
	// it should be used inside a middleware.
	Skip()
	// StopExecution if called then the following .Next calls are ignored,
	// as a result the next handlers in the chain will not be fire.
	StopExecution()
	// IsStopped checks and returns true if the current position of the Context is 255,
	// means that the StopExecution() was called.
	IsStopped() bool
	// OnConnectionClose registers the "cb" function which will fire (on its own goroutine, no need to be registered goroutine by the end-dev)
	// when the underlying connection has gone away.
	//
	// This mechanism can be used to cancel long operations on the server
	// if the client has disconnected before the response is ready.
	//
	// It depends on the `http#CloseNotify`.
	// CloseNotify may wait to notify until Request.Body has been
	// fully read.
	//
	// After the main Handler has returned, there is no guarantee
	// that the channel receives a value.
	//
	// Finally, it reports whether the protocol supports pipelines (HTTP/1.1 with pipelines disabled is not supported).
	// The "cb" will not fire for sure if the output value is false.
	//
	// Note that you can register only one callback for the entire request handler chain/per route.
	//
	// Look the `ResponseWriter#CloseNotifier` for more.
	OnConnectionClose(fnGoroutine func()) bool
	// OnClose registers the callback function "cb" to the underline connection closing event using the `Context#OnConnectionClose`
	// and also in the end of the request handler using the `ResponseWriter#SetBeforeFlush`.
	// Note that you can register only one callback for the entire request handler chain/per route.
	//
	// Look the `Context#OnConnectionClose` and `ResponseWriter#SetBeforeFlush` for more.
	OnClose(cb func())

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

	//  +------------------------------------------------------------+
	//  | Path, Host, Subdomain, IP, Headers, Localization etc...    |
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
	// FindClosest returns a list of "n" paths close to
	// this request based on subdomain and request path.
	//
	// Order may change.
	// Example: https://github.com/kataras/iris/tree/master/_examples/routing/not-found-suggests
	FindClosest(n int) []string
	// IsWWW returns true if the current subdomain (if any) is www.
	IsWWW() bool
	// FullRqeuestURI returns the full URI,
	// including the scheme, the host and the relative requested path/resource.
	FullRequestURI() string
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
	// IsMobile checks if client is using a mobile device(phone or tablet) to communicate with this server.
	// If the return value is true that means that the http client using a mobile
	// device to communicate with the server, otherwise false.
	//
	// Keep note that this checks the "User-Agent" request header.
	IsMobile() bool
	// IsScript reports whether a client is a script.
	IsScript() bool
	// GetReferrer extracts and returns the information from the "Referer" header as specified
	// in https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
	// or by the URL query parameter "referer".
	GetReferrer() Referrer
	// GetLocale returns the current request's `Locale` found by i18n middleware.
	// See `Tr` too.
	GetLocale() Locale
	// Tr returns a i18n localized message based on format with optional arguments.
	// See `GetLocale` too.
	// Example: https://github.com/kataras/iris/tree/master/_examples/i18n
	Tr(format string, args ...interface{}) string

	//  +------------------------------------------------------------+
	//  | Headers helpers                                            |
	//  +------------------------------------------------------------+

	// Header adds a header to the response writer.
	Header(name string, value string)

	// ContentType sets the response writer's header key "Content-Type" to the 'cType'.
	ContentType(cType string)
	// GetContentType returns the response writer's header value of "Content-Type"
	// which may, set before with the 'ContentType'.
	GetContentType() string
	// GetContentType returns the request's header value of "Content-Type".
	GetContentTypeRequested() string

	// GetContentLength returns the request's header value of "Content-Length".
	// Returns 0 if header was unable to be found or its value was not a valid number.
	GetContentLength() int64

	// StatusCode sets the status code header to the response.
	// Look .`GetStatusCode` too.
	StatusCode(statusCode int)
	// GetStatusCode returns the current status code of the response.
	// Look `StatusCode` too.
	GetStatusCode() int

	// AbsoluteURI parses the "s" and returns its absolute URI form.
	AbsoluteURI(s string) string
	// Redirect sends a redirect response to the client
	// to a specific url or relative path.
	// accepts 2 parameters string and an optional int
	// first parameter is the url to redirect
	// second parameter is the http status should send,
	// default is 302 (StatusFound),
	// you can set it to 301 (Permant redirect)
	// or 303 (StatusSeeOther) if POST method,
	// or StatusTemporaryRedirect(307) if that's nessecery.
	Redirect(urlToRedirect string, statusHeader ...int)
	//  +------------------------------------------------------------+
	//  | Various Request and Post Data                              |
	//  +------------------------------------------------------------+

	// URLParam returns true if the url parameter exists, otherwise false.
	URLParamExists(name string) bool
	// URLParamDefault returns the get parameter from a request,
	// if not found then "def" is returned.
	URLParamDefault(name string, def string) string
	// URLParam returns the get parameter from a request, if any.
	URLParam(name string) string
	// URLParamTrim returns the url query parameter with trailing white spaces removed from a request.
	URLParamTrim(name string) string
	// URLParamEscape returns the escaped url query parameter from a request.
	URLParamEscape(name string) string
	// URLParamInt returns the url query parameter as int value from a request,
	// returns -1 and an error if parse failed.
	URLParamInt(name string) (int, error)
	// URLParamIntDefault returns the url query parameter as int value from a request,
	// if not found or parse failed then "def" is returned.
	URLParamIntDefault(name string, def int) int
	// URLParamInt32Default returns the url query parameter as int32 value from a request,
	// if not found or parse failed then "def" is returned.
	URLParamInt32Default(name string, def int32) int32
	// URLParamInt64 returns the url query parameter as int64 value from a request,
	// returns -1 and an error if parse failed.
	URLParamInt64(name string) (int64, error)
	// URLParamInt64Default returns the url query parameter as int64 value from a request,
	// if not found or parse failed then "def" is returned.
	URLParamInt64Default(name string, def int64) int64
	// URLParamFloat64 returns the url query parameter as float64 value from a request,
	// returns -1 and an error if parse failed.
	URLParamFloat64(name string) (float64, error)
	// URLParamFloat64Default returns the url query parameter as float64 value from a request,
	// if not found or parse failed then "def" is returned.
	URLParamFloat64Default(name string, def float64) float64
	// URLParamBool returns the url query parameter as boolean value from a request,
	// returns an error if parse failed or not found.
	URLParamBool(name string) (bool, error)
	// URLParams returns a map of GET query parameters separated by comma if more than one
	// it returns an empty map if nothing found.
	URLParams() map[string]string

	// FormValueDefault returns a single parsed form value by its "name",
	// including both the URL field's query parameters and the POST or PUT form data.
	//
	// Returns the "def" if not found.
	FormValueDefault(name string, def string) string
	// FormValue returns a single parsed form value by its "name",
	// including both the URL field's query parameters and the POST or PUT form data.
	FormValue(name string) string
	// FormValues returns the parsed form data, including both the URL
	// field's query parameters and the POST or PUT form data.
	//
	// The default form's memory maximum size is 32MB, it can be changed by the
	// `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
	//
	// NOTE: A check for nil is necessary.
	FormValues() map[string][]string

	// PostValueDefault returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name".
	//
	// If not found then "def" is returned instead.
	PostValueDefault(name string, def string) string
	// PostValue returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name"
	PostValue(name string) string
	// PostValueTrim returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name",  without trailing spaces.
	PostValueTrim(name string) string
	// PostValueInt returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as int.
	//
	// If not found returns -1 and a non-nil error.
	PostValueInt(name string) (int, error)
	// PostValueIntDefault returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as int.
	//
	// If not found returns or parse errors the "def".
	PostValueIntDefault(name string, def int) int
	// PostValueInt64 returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as float64.
	//
	// If not found returns -1 and a no-nil error.
	PostValueInt64(name string) (int64, error)
	// PostValueInt64Default returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as int64.
	//
	// If not found or parse errors returns the "def".
	PostValueInt64Default(name string, def int64) int64
	// PostValueInt64Default returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as float64.
	//
	// If not found returns -1 and a non-nil error.
	PostValueFloat64(name string) (float64, error)
	// PostValueInt64Default returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as float64.
	//
	// If not found or parse errors returns the "def".
	PostValueFloat64Default(name string, def float64) float64
	// PostValueInt64Default returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as bool.
	//
	// If not found or value is false, then it returns false, otherwise true.
	PostValueBool(name string) (bool, error)
	// PostValues returns all the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name" as a string slice.
	//
	// The default form's memory maximum size is 32MB, it can be changed by the
	// `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
	PostValues(name string) []string
	// FormFile returns the first uploaded file that received from the client.
	//
	// The default form's memory maximum size is 32MB, it can be changed by the
	//  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/http_request/upload-file
	FormFile(key string) (multipart.File, *multipart.FileHeader, error)
	// UploadFormFiles uploads any received file(s) from the client
	// to the system physical location "destDirectory".
	//
	// The second optional argument "before" gives caller the chance to
	// modify the *miltipart.FileHeader before saving to the disk,
	// it can be used to change a file's name based on the current request,
	// all FileHeader's options can be changed. You can ignore it if
	// you don't need to use this capability before saving a file to the disk.
	//
	// Note that it doesn't check if request body streamed.
	//
	// Returns the copied length as int64 and
	// a not nil error if at least one new file
	// can't be created due to the operating system's permissions or
	// http.ErrMissingFile if no file received.
	//
	// If you want to receive & accept files and manage them manually you can use the `context#FormFile`
	// instead and create a copy function that suits your needs, the below is for generic usage.
	//
	// The default form's memory maximum size is 32MB, it can be changed by the
	//  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
	//
	// See `FormFile` to a more controlled to receive a file.
	//
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/http_request/upload-files
	UploadFormFiles(destDirectory string, before ...func(Context, *multipart.FileHeader)) (n int64, err error)

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

	// GetBody reads and returns the request body.
	// The default behavior for the http request reader is to consume the data readen
	// but you can change that behavior by passing the `WithoutBodyConsumptionOnUnmarshal` iris option.
	//
	// However, whenever you can use the `ctx.Request().Body` instead.
	GetBody() ([]byte, error)
	// UnmarshalBody reads the request's body and binds it to a value or pointer of any type.
	// Examples of usage: context.ReadJSON, context.ReadXML.
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-custom-via-unmarshaler/main.go
	//
	// UnmarshalBody does not check about gzipped data.
	// Do not rely on compressed data incoming to your server. The main reason is: https://en.wikipedia.org/wiki/Zip_bomb
	// However you are still free to read the `ctx.Request().Body io.Reader` manually.
	UnmarshalBody(outPtr interface{}, unmarshaler Unmarshaler) error
	// ReadJSON reads JSON from request's body and binds it to a pointer of a value of any json-valid type.
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-json/main.go
	ReadJSON(jsonObjectPtr interface{}) error
	// ReadXML reads XML from request's body and binds it to a pointer of a value of any xml-valid type.
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-xml/main.go
	ReadXML(xmlObjectPtr interface{}) error
	// ReadYAML reads YAML from request's body and binds it to the "outPtr" value.
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-yaml/main.go
	ReadYAML(outPtr interface{}) error
	// ReadForm binds the formObject  with the form data
	// it supports any kind of type, including custom structs.
	// It will return nothing if request data are empty.
	// The struct field tag is "form".
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-form/main.go
	ReadForm(formObject interface{}) error
	// ReadQuery binds the "ptr" with the url query string. The struct field tag is "url".
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-query/main.go
	ReadQuery(ptr interface{}) error
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

	// SetLastModified sets the "Last-Modified" based on the "modtime" input.
	// If "modtime" is zero then it does nothing.
	//
	// It's mostly internally on core/router and context packages.
	//
	// Note that modtime.UTC() is being used instead of just modtime, so
	// you don't have to know the internals in order to make that works.
	SetLastModified(modtime time.Time)
	// CheckIfModifiedSince checks if the response is modified since the "modtime".
	// Note that it has nothing to do with server-side caching.
	// It does those checks by checking if the "If-Modified-Since" request header
	// sent by client or a previous server response header
	// (e.g with WriteWithExpiration or HandleDir or Favicon etc.)
	// is a valid one and it's before the "modtime".
	//
	// A check for !modtime && err == nil is necessary to make sure that
	// it's not modified since, because it may return false but without even
	// had the chance to check the client-side (request) header due to some errors,
	// like the HTTP Method is not "GET" or "HEAD" or if the "modtime" is zero
	// or if parsing time from the header failed.
	//
	// It's mostly used internally, e.g. `context#WriteWithExpiration`. See `ErrPreconditionFailed` too.
	//
	// Note that modtime.UTC() is being used instead of just modtime, so
	// you don't have to know the internals in order to make that works.
	CheckIfModifiedSince(modtime time.Time) (bool, error)
	// WriteNotModified sends a 304 "Not Modified" status code to the client,
	// it makes sure that the content type, the content length headers
	// and any "ETag" are removed before the response sent.
	//
	// It's mostly used internally on core/router/fs.go and context methods.
	WriteNotModified()
	// WriteWithExpiration works like `Write` but it will check if a resource is modified,
	// based on the "modtime" input argument,
	// otherwise sends a 304 status code in order to let the client-side render the cached content.
	WriteWithExpiration(body []byte, modtime time.Time) (int, error)
	// StreamWriter registers the given stream writer for populating
	// response body.
	//
	// Access to context's and/or its' members is forbidden from writer.
	//
	// This function may be used in the following cases:
	//
	//     * if response body is too big (more than iris.LimitRequestBodySize(if set)).
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
	// You may re-use this function in the same handler
	// to write more data many times without any troubles.
	WriteGzip(b []byte) (int, error)
	// TryWriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
	// If client does not supprots gzip then the contents are written as they are, uncompressed.
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
	// Note that the 'layoutTmplFile' argument can be set to iris.NoLayout || view.NoLayout
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
	// View renders a template based on the registered view engine(s).
	// First argument accepts the filename, relative to the view engine's Directory and Extension,
	// i.e: if directory is "./templates" and want to render the "./templates/users/index.html"
	// then you pass the "users/index.html" as the filename argument.
	//
	// The second optional argument can receive a single "view model"
	// that will be binded to the view template if it's not nil,
	// otherwise it will check for previous view data stored by the `ViewData`
	// even if stored at any previous handler(middleware) for the same request.
	//
	// Look .ViewData` and .ViewLayout too.
	//
	// Examples: https://github.com/kataras/iris/tree/master/_examples/view
	View(filename string, optionalViewModel ...interface{}) error

	// Binary writes out the raw bytes as binary data.
	Binary(data []byte) (int, error)
	// Text writes out a string as plain text.
	Text(format string, args ...interface{}) (int, error)
	// HTML writes out a string as text/html.
	HTML(format string, args ...interface{}) (int, error)
	// JSON marshals the given interface object and writes the JSON response.
	JSON(v interface{}, options ...JSON) (int, error)
	// JSONP marshals the given interface object and writes the JSON response.
	JSONP(v interface{}, options ...JSONP) (int, error)
	// XML marshals the given interface object and writes the XML response.
	// To render maps as XML see the `XMLMap` package-level function.
	XML(v interface{}, options ...XML) (int, error)
	// Problem writes a JSON or XML problem response.
	// Order of Problem fields are not always rendered the same.
	//
	// Behaves exactly like `Context.JSON`
	// but with default ProblemOptions.JSON indent of " " and
	// a response content type of "application/problem+json" instead.
	//
	// Use the options.RenderXML and XML fields to change this behavior and
	// send a response of content type "application/problem+xml" instead.
	//
	// Read more at: https://github.com/kataras/iris/wiki/Routing-error-handlers
	Problem(v interface{}, opts ...ProblemOptions) (int, error)
	// Markdown parses the markdown to html and renders its result to the client.
	Markdown(markdownB []byte, options ...Markdown) (int, error)
	// YAML parses the "v" using the yaml parser and renders its result to the client.
	YAML(v interface{}) (int, error)

	//  +-----------------------------------------------------------------------+
	//  | Content Νegotiation                                                   |
	//  | https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation |                                       |
	//  +-----------------------------------------------------------------------+

	// Negotiation creates once and returns the negotiation builder
	// to build server-side available content for specific mime type(s)
	// and charset(s).
	//
	// See `Negotiate` method too.
	Negotiation() *NegotiationBuilder
	// Negotiate used for serving different representations of a resource at the same URI.
	//
	// The "v" can be a single `N` struct value.
	// The "v" can be any value completes the `ContentSelector` interface.
	// The "v" can be any value completes the `ContentNegotiator` interface.
	// The "v" can be any value of struct(JSON, JSONP, XML, YAML) or
	// string(TEXT, HTML) or []byte(Markdown, Binary) or []byte with any matched mime type.
	//
	// If the "v" is nil, the `Context.Negotitation()` builder's
	// content will be used instead, otherwise "v" overrides builder's content
	// (server mime types are still retrieved by its registered, supported, mime list)
	//
	// Set mime type priorities by `Negotiation().JSON().XML().HTML()...`.
	// Set charset priorities by `Negotiation().Charset(...)`.
	// Set encoding algorithm priorities by `Negotiation().Encoding(...)`.
	// Modify the accepted by
	// `Negotiation().Accept./Override()/.XML().JSON().Charset(...).Encoding(...)...`.
	//
	// It returns `ErrContentNotSupported` when not matched mime type(s).
	//
	// Resources:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Charset
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Encoding
	//
	// Supports the above without quality values.
	//
	// Read more at: https://github.com/kataras/iris/wiki/Content-negotiation
	Negotiate(v interface{}) (int, error)

	//  +------------------------------------------------------------+
	//  | Serve files                                                |
	//  +------------------------------------------------------------+

	// ServeContent serves content, headers are autoset
	// receives three parameters, it's low-level function, instead you can use .ServeFile(string,bool)/SendFile(string,string)
	//
	//
	// You can define your own "Content-Type" with `context#ContentType`, before this function call.
	//
	// This function doesn't support resuming (by range),
	// use ctx.SendFile or router's `HandleDir` instead.
	ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error
	// ServeFile serves a file (to send a file, a zip for example to the client you should use the `SendFile` instead)
	// receives two parameters
	// filename/path (string)
	// gzipCompression (bool)
	//
	// You can define your own "Content-Type" with `context#ContentType`, before this function call.
	//
	// This function doesn't support resuming (by range),
	// use ctx.SendFile or router's `HandleDir` instead.
	//
	// Use it when you want to serve dynamic files to the client.
	ServeFile(filename string, gzipCompression bool) error
	// SendFile sends file for force-download to the client
	//
	// Use this instead of ServeFile to 'force-download' bigger files to the client.
	SendFile(filename string, destinationName string) error

	//  +------------------------------------------------------------+
	//  | Cookies                                                    |
	//  +------------------------------------------------------------+

	// SetCookie adds a cookie.
	// Use of the "options" is not required, they can be used to amend the "cookie".
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
	SetCookie(cookie *http.Cookie, options ...CookieOption)
	// SetCookieKV adds a cookie, requires the name(string) and the value(string).
	//
	// By default it expires at 2 hours and it's added to the root path,
	// use the `CookieExpires` and `CookiePath` to modify them.
	// Alternatively: ctx.SetCookie(&http.Cookie{...})
	//
	// If you want to set custom the path:
	// ctx.SetCookieKV(name, value, iris.CookiePath("/custom/path/cookie/will/be/stored"))
	//
	// If you want to be visible only to current request path:
	// ctx.SetCookieKV(name, value, iris.CookieCleanPath/iris.CookiePath(""))
	// More:
	//                              iris.CookieExpires(time.Duration)
	//                              iris.CookieHTTPOnly(false)
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
	SetCookieKV(name, value string, options ...CookieOption)
	// GetCookie returns cookie's value by its name
	// returns empty string if nothing was found.
	//
	// If you want more than the value then:
	// cookie, err := ctx.Request().Cookie("name")
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
	GetCookie(name string, options ...CookieOption) string
	// RemoveCookie deletes a cookie by its name and path = "/".
	// Tip: change the cookie's path to the current one by: RemoveCookie("name", iris.CookieCleanPath)
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
	RemoveCookie(name string, options ...CookieOption)
	// VisitAllCookies accepts a visitor function which is called
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

	// Exec calls the `context/Application#ServeCtx`
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
	Exec(method, path string)

	// RouteExists reports whether a particular route exists
	// It will search from the current subdomain of context's host, if not inside the root domain.
	RouteExists(method, path string) bool

	// ReflectValue caches and returns a []reflect.Value{reflect.ValueOf(ctx)}.
	// It's just a helper to maintain variable inside the context itself.
	ReflectValue() []reflect.Value

	// Application returns the iris app instance which belongs to this context.
	// Worth to notice that this function returns an interface
	// of the Application, which contains methods that are safe
	// to be executed at serve-time. The full app's fields
	// and methods are not available here for the developer's safety.
	Application() Application

	// String returns the string representation of this request.
	// Each context has a unique string representation.
	// It can be used for simple debugging scenarios, i.e print context as string.
	//
	// What it returns? A number which declares the length of the
	// total `String` calls per executable application, followed
	// by the remote IP (the client) and finally the method:url.
	String() string
}

var _ Context = (*context)(nil)

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

// Map is just a type alias of the map[string]interface{} type.
type Map = map[string]interface{}

//  +------------------------------------------------------------+
//  | Context Implementation                                     |
//  +------------------------------------------------------------+

type context struct {
	// the unique id, it's zero until `String` function is called,
	// it's here to cache the random, unique context's id, although `String`
	// returns more than this.
	id uint64
	// the http.ResponseWriter wrapped by custom writer.
	writer ResponseWriter
	// the original http.Request
	request *http.Request
	// the current route's name registered to this request path.
	currentRouteName string

	// the local key-value storage
	params RequestParams  // url named parameters.
	values memstore.Store // generic storage, middleware communication.

	// the underline application app.
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
	ctx.params.Store = ctx.params.Store[0:0]
	ctx.request = r
	ctx.currentHandlerIndex = 0
	ctx.writer = AcquireResponseWriter()
	ctx.writer.BeginResponse(w)
}

// StatusCodeNotSuccessful defines if a specific "statusCode" is not
// a valid status code for a successful response.
// It defaults to < 200 || >= 400
//
// Read more at `iris#DisableAutoFireStatusCode`, `iris/core/router#ErrorCodeHandler`
// and `iris/core/router#OnAnyErrorCode` for relative information.
//
// Do NOT change it.
//
// It's exported for extreme situations--special needs only, when the Iris server and the client
// is not following the RFC: https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
var StatusCodeNotSuccessful = func(statusCode int) bool {
	return statusCode < 200 || statusCode >= 400
}

// EndRequest is executing once after a response to the request was sent and this context is useless or released.
//
// To follow the iris' flow, developer should:
// 1. flush the response writer's result
// 2. release the response writer
// and any other optional steps, depends on dev's application type.
func (ctx *context) EndRequest() {
	if StatusCodeNotSuccessful(ctx.GetStatusCode()) &&
		!ctx.Application().ConfigurationReadOnly().GetDisableAutoFireStatusCode() {
		// author's note:
		// if recording, the error handler can handle
		// the rollback and remove any response written before,
		// we don't have to do anything here, written is <=0 (-1 for default empty, even no status code)
		// when Recording
		// because we didn't flush the response yet
		// if !recording  then check if the previous handler didn't send something
		// to the client.
		if ctx.writer.Written() <= 0 {
			// Author's notes:
			// previously: == -1,
			// <=0 means even if empty write called which has meaning;
			// rel: core/router/status.go#Fire-else
			// mvc/activator/funcmethod/func_result_dispatcher.go#DispatchCommon-write
			// mvc/response.go#defaultFailureResponse - no text given but
			// status code should be fired, but it couldn't because of the .Write
			// action, the .Written() was 0 even on empty response, this 0 means that
			// a status code given, the previous check of the "== -1" didn't make check for that,
			// we do now.
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

// ResetRequest sets the Context's Request,
// It is useful to store the new request created by a std *http.Request#WithContext() into Iris' Context.
// Use `ResetRequest` when for some reason you want to make a full
// override of the *http.Request.
// Note that: when you just want to change one of each fields you can use the Request() which returns a pointer to Request,
// so the changes will have affect without a full override.
// Usage: you use a native http handler which uses the standard "context" package
// to get values instead of the Iris' Context#Values():
// r := ctx.Request()
// stdCtx := context.WithValue(r.Context(), key, val)
// ctx.ResetRequest(r.WithContext(stdCtx)).
func (ctx *context) ResetRequest(r *http.Request) {
	ctx.request = r
}

// SetCurrentRouteName sets the route's name internally,
// in order to be able to find the correct current "read-only" Route when
// end-developer calls the `GetCurrentRoute()` function.
// It's being initialized by the Router, if you change that name
// manually nothing really happens except that you'll get other
// route via `GetCurrentRoute()`.
// Instead, to execute a different path
// from this context you should use the `Exec` function
// or change the handlers via `SetHandlers/AddHandler` functions.
func (ctx *context) SetCurrentRouteName(currentRouteName string) {
	ctx.currentRouteName = currentRouteName
}

// GetCurrentRoute returns the current registered "read-only" route that
// was being registered to this request's path.
func (ctx *context) GetCurrentRoute() RouteReadOnly {
	return ctx.app.GetRouteReadOnly(ctx.currentRouteName)
}

// Do calls the SetHandlers(handlers)
// and executes the first handler,
// handlers should not be empty.
//
// It's used by the router, developers may use that
// to replace and execute handlers immediately.
func (ctx *context) Do(handlers Handlers) {
	Do(ctx, handlers)
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

// Proceed is an alternative way to check if a particular handler
// has been executed and called the `ctx.Next` function inside it.
// This is useful only when you run a handler inside
// another handler. It justs checks for before index and the after index.
//
// A usecase example is when you want to execute a middleware
// inside controller's `BeginRequest` that calls the `ctx.Next` inside it.
// The Controller looks the whole flow (BeginRequest, method handler, EndRequest)
// as one handler, so `ctx.Next` will not be reflected to the method handler
// if called from the `BeginRequest`.
//
// Although `BeginRequest` should NOT be used to call other handlers,
// the `BeginRequest` has been introduced to be able to set
// common data to all method handlers before their execution.
// Controllers can accept middleware(s) from the MVC's Application's Router as normally.
//
// That said let's see an example of `ctx.Proceed`:
//
// var authMiddleware = basicauth.New(basicauth.Config{
// 	Users: map[string]string{
// 		"admin": "password",
// 	},
// })
//
// func (c *UsersController) BeginRequest(ctx iris.Context) {
// 	if !ctx.Proceed(authMiddleware) {
// 		ctx.StopExecution()
// 	}
// }
// This Get() will be executed in the same handler as `BeginRequest`,
// internally controller checks for `ctx.StopExecution`.
// So it will not be fired if BeginRequest called the `StopExecution`.
// func(c *UsersController) Get() []models.User {
//	  return c.Service.GetAll()
//}
// Alternative way is `!ctx.IsStopped()` if middleware make use of the `ctx.StopExecution()` on failure.
func (ctx *context) Proceed(h Handler) bool {
	beforeIdx := ctx.currentHandlerIndex
	h(ctx)
	if ctx.currentHandlerIndex > beforeIdx && !ctx.IsStopped() {
		return true
	}
	return false
}

// HandlerName returns the current handler's name, helpful for debugging.
func (ctx *context) HandlerName() string {
	return HandlerName(ctx.handlers[ctx.currentHandlerIndex])
}

// HandlerFileLine returns the current running handler's function source file and line information.
// Useful mostly when debugging.
func (ctx *context) HandlerFileLine() (file string, line int) {
	return HandlerFileLine(ctx.handlers[ctx.currentHandlerIndex])
}

// RouteName returns the route name that this handler is running on.
// Note that it will return empty on not found handlers.
func (ctx *context) RouteName() string {
	return ctx.currentRouteName
}

// Next is the function that executed when `ctx.Next()` is called.
// It can be changed to a customized one if needed (very advanced usage).
//
// See `DefaultNext` for more information about this and why it's exported like this.
var Next = DefaultNext

// DefaultNext is the default function that executed on each middleware if `ctx.Next()`
// is called.
//
// DefaultNext calls the next handler from the handlers chain by registration order,
// it should be used inside a middleware.
//
// It can be changed to a customized one if needed (very advanced usage).
//
// Developers are free to customize the whole or part of the Context's implementation
// by implementing a new `context.Context` (see https://github.com/kataras/iris/tree/master/_examples/routing/custom-context)
// or by just override the `context.Next` package-level field, `context.DefaultNext` is exported
// in order to be able for developers to merge your customized version one with the default behavior as well.
func DefaultNext(ctx Context) {
	if ctx.IsStopped() {
		return
	}
	if n, handlers := ctx.HandlerIndex(-1)+1, ctx.Handlers(); n < len(handlers) {
		ctx.HandlerIndex(n)
		handlers[n](ctx)
	}
}

// Next calls all the next handler from the handlers chain,
// it should be used inside a middleware.
//
// Note: Custom context should override this method in order to be able to pass its own context.Context implementation.
func (ctx *context) Next() { // or context.Next(ctx)
	Next(ctx)
}

// NextOr checks if chain has a next handler, if so then it executes it
// otherwise it sets a new chain assigned to this Context based on the given handler(s)
// and executes its first handler.
//
// Returns true if next handler exists and executed, otherwise false.
//
// Note that if no next handler found and handlers are missing then
// it sends a Status Not Found (404) to the client and it stops the execution.
func (ctx *context) NextOr(handlers ...Handler) bool {
	if next := ctx.NextHandler(); next != nil {
		next(ctx)
		ctx.Skip() // skip this handler from the chain.
		return true
	}

	if len(handlers) == 0 {
		ctx.NotFound()
		ctx.StopExecution()
		return false
	}

	ctx.Do(handlers)

	return false
}

// NextOrNotFound checks if chain has a next handler, if so then it executes it
// otherwise it sends a Status Not Found (404) to the client and stops the execution.
//
// Returns true if next handler exists and executed, otherwise false.
func (ctx *context) NextOrNotFound() bool { return ctx.NextOr() }

// NextHandler returns (it doesn't execute) the next handler from the handlers chain.
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

// StopExecution if called then the following .Next calls are ignored,
// as a result the next handlers in the chain will not be fire.
func (ctx *context) StopExecution() {
	ctx.currentHandlerIndex = stopExecutionIndex
}

// IsStopped checks and returns true if the current position of the context is -1,
// means that the StopExecution() was called.
func (ctx *context) IsStopped() bool {
	return ctx.currentHandlerIndex == stopExecutionIndex
}

// OnConnectionClose registers the "cb" function which will fire (on its own goroutine, no need to be registered goroutine by the end-dev)
// when the underlying connection has gone away.
//
// This mechanism can be used to cancel long operations on the server
// if the client has disconnected before the response is ready.
//
// It depends on the `http#CloseNotify`.
// CloseNotify may wait to notify until Request.Body has been
// fully read.
//
// After the main Handler has returned, there is no guarantee
// that the channel receives a value.
//
// Finally, it reports whether the protocol supports pipelines (HTTP/1.1 with pipelines disabled is not supported).
// The "cb" will not fire for sure if the output value is false.
//
// Note that you can register only one callback for the entire request handler chain/per route.
//
// Look the `ResponseWriter#CloseNotifier` for more.
func (ctx *context) OnConnectionClose(cb func()) bool {
	// Note that `ctx.ResponseWriter().CloseNotify()` can already do the same
	// but it returns a channel which will never fire if it the protocol version is not compatible,
	// here we don't want to allocate an empty channel, just skip it.
	notifier, ok := ctx.writer.CloseNotifier()
	if !ok {
		return false
	}

	notify := notifier.CloseNotify()
	go func() {
		<-notify
		if cb != nil {
			cb()
		}
	}()

	return true
}

// OnClose registers the callback function "cb" to the underline connection closing event using the `Context#OnConnectionClose`
// and also in the end of the request handler using the `ResponseWriter#SetBeforeFlush`.
// Note that you can register only one callback for the entire request handler chain/per route.
//
// Look the `Context#OnConnectionClose` and `ResponseWriter#SetBeforeFlush` for more.
func (ctx *context) OnClose(cb func()) {
	if cb == nil {
		return
	}

	// Register the on underline connection close handler first.
	ctx.OnConnectionClose(cb)

	// Author's notes:
	// This is fired on `ctx.ResponseWriter().FlushResponse()` which is fired by the framework automatically, internally, on the end of request handler(s),
	// it is not fired on the underline streaming function of the writer: `ctx.ResponseWriter().Flush()` (which can be fired more than one if streaming is supported by the client).
	// The `FlushResponse` is called only once, so add the "cb" here, no need to add done request handlers each time `OnClose` is called by the end-dev.
	//
	// Don't allow more than one because we don't allow that on `OnConnectionClose` too:
	// old := ctx.writer.GetBeforeFlush()
	// if old != nil {
	// 	ctx.writer.SetBeforeFlush(func() {
	// 		old()
	// 		cb()
	// 	})
	// 	return
	// }

	ctx.writer.SetBeforeFlush(cb)
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
		return ctx.request.URL.EscapedPath() // DecodeQuery(ctx.request.URL.EscapedPath())
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

// Host returns the host part of the current URI.
func (ctx *context) Host() string {
	return GetHost(ctx.request)
}

// GetHost returns the host part of the current URI.
func GetHost(r *http.Request) string {
	if host := r.Host; host != "" {
		return host
	}

	return r.URL.Host
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

// FindClosest returns a list of "n" paths close to
// this request based on subdomain and request path.
//
// Order may change.
// Example: https://github.com/kataras/iris/tree/master/_examples/routing/not-found-suggests
func (ctx *context) FindClosest(n int) []string {
	return ctx.Application().FindClosestPaths(ctx.Subdomain(), ctx.Path(), n)
}

// IsWWW returns true if the current subdomain (if any) is www.
func (ctx *context) IsWWW() bool {
	host := ctx.Host()
	if index := strings.IndexByte(host, '.'); index > 0 {
		// if it has a subdomain and it's www then return true.
		if subdomain := host[0:index]; !strings.Contains(ctx.Application().ConfigurationReadOnly().GetVHost(), subdomain) {
			return subdomain == "www"
		}
	}
	return false
}

// FullRqeuestURI returns the full URI,
// including the scheme, the host and the relative requested path/resource.
func (ctx *context) FullRequestURI() string {
	return ctx.AbsoluteURI(ctx.Path())
}

const xForwardedForHeaderKey = "X-Forwarded-For"

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
			if headerName == xForwardedForHeaderKey {
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

var isMobileRegex = regexp.MustCompile("(?:hpw|i|web)os|alamofire|alcatel|amoi|android|avantgo|blackberry|blazer|cell|cfnetwork|darwin|dolfin|dolphin|fennec|htc|ip(?:hone|od|ad)|ipaq|j2me|kindle|midp|minimo|mobi|motorola|nec-|netfront|nokia|opera m(ob|in)i|palm|phone|pocket|portable|psp|silk-accelerated|skyfire|sony|ucbrowser|up.browser|up.link|windows ce|xda|zte|zune")

// IsMobile checks if client is using a mobile device(phone or tablet) to communicate with this server.
// If the return value is true that means that the http client using a mobile
// device to communicate with the server, otherwise false.
//
// Keep note that this checks the "User-Agent" request header.
func (ctx *context) IsMobile() bool {
	s := strings.ToLower(ctx.GetHeader("User-Agent"))
	return isMobileRegex.MatchString(s)
}

var isScriptRegex = regexp.MustCompile("curl|wget|collectd|python|urllib|java|jakarta|httpclient|phpcrawl|libwww|perl|go-http|okhttp|lua-resty|winhttp|awesomium")

// IsScript reports whether a client is a script.
func (ctx *context) IsScript() bool {
	s := strings.ToLower(ctx.GetHeader("User-Agent"))
	return isScriptRegex.MatchString(s)
}

type (
	// Referrer contains the extracted information from the `GetReferrer`
	//
	// The structure contains struct tags for JSON, form, XML, YAML and TOML.
	// Look the `GetReferrer() Referrer` and `goreferrer` external package.
	Referrer struct {
		Type       ReferrerType             `json:"type" form:"referrer_type" xml:"Type" yaml:"Type" toml:"Type"`
		Label      string                   `json:"label" form:"referrer_form" xml:"Label" yaml:"Label" toml:"Label"`
		URL        string                   `json:"url" form:"referrer_url" xml:"URL" yaml:"URL" toml:"URL"`
		Subdomain  string                   `json:"subdomain" form:"referrer_subdomain" xml:"Subdomain" yaml:"Subdomain" toml:"Subdomain"`
		Domain     string                   `json:"domain" form:"referrer_domain" xml:"Domain" yaml:"Domain" toml:"Domain"`
		Tld        string                   `json:"tld" form:"referrer_tld" xml:"Tld" yaml:"Tld" toml:"Tld"`
		Path       string                   `json:"path" form:"referrer_path" xml:"Path" yaml:"Path" toml:"Path"`
		Query      string                   `json:"query" form:"referrer_query" xml:"Query" yaml:"Query" toml:"GoogleType"`
		GoogleType ReferrerGoogleSearchType `json:"googleType" form:"referrer_google_type" xml:"GoogleType" yaml:"GoogleType" toml:"GoogleType"`
	}

	// ReferrerType is the goreferrer enum for a referrer type (indirect, direct, email, search, social).
	ReferrerType = goreferrer.ReferrerType

	// ReferrerGoogleSearchType is the goreferrer enum for a google search type (organic, adwords).
	ReferrerGoogleSearchType = goreferrer.GoogleSearchType
)

// Contains the available values of the goreferrer enums.
const (
	ReferrerInvalid ReferrerType = iota
	ReferrerIndirect
	ReferrerDirect
	ReferrerEmail
	ReferrerSearch
	ReferrerSocial

	ReferrerNotGoogleSearch ReferrerGoogleSearchType = iota
	ReferrerGoogleOrganicSearch
	ReferrerGoogleAdwords
)

// unnecessary but good to know the default values upfront.
var emptyReferrer = Referrer{Type: ReferrerInvalid, GoogleType: ReferrerNotGoogleSearch}

// GetReferrer extracts and returns the information from the "Referer" header as specified
// in https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
// or by the URL query parameter "referer".
func (ctx *context) GetReferrer() Referrer {
	// the underline net/http follows the https://tools.ietf.org/html/rfc7231#section-5.5.2,
	// so there is nothing special left to do.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
	refURL := ctx.GetHeader("Referer")
	if refURL == "" {
		refURL = ctx.URLParam("referer")
	}

	if ref := goreferrer.DefaultRules.Parse(refURL); ref.Type > goreferrer.Invalid {
		return Referrer{
			Type:       ReferrerType(ref.Type),
			Label:      ref.Label,
			URL:        ref.URL,
			Subdomain:  ref.Subdomain,
			Domain:     ref.Domain,
			Tld:        ref.Tld,
			Path:       ref.Path,
			Query:      ref.Query,
			GoogleType: ReferrerGoogleSearchType(ref.GoogleType),
		}
	}

	return emptyReferrer
}

// GetLocale returns the current request's `Locale` found by i18n middleware.
// See `Tr` too.
func (ctx *context) GetLocale() Locale {
	contextKey := ctx.app.ConfigurationReadOnly().GetLocaleContextKey()
	if v := ctx.Values().Get(contextKey); v != nil {
		if locale, ok := v.(Locale); ok {
			return locale
		}
	}

	if locale := ctx.Application().I18nReadOnly().GetLocale(ctx); locale != nil {
		ctx.Values().Set(contextKey, locale)
		return locale
	}

	return nil
}

// Tr returns a i18n localized message based on format with optional arguments.
// See `GetLocale` too.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/i18n
func (ctx *context) Tr(format string, args ...interface{}) string { // other name could be: Localize.
	if locale := ctx.GetLocale(); locale != nil { // TODO: here... I need to change the logic, if not found then call the i18n's get locale and set the value in order to be fastest on routes that are not using (no need to reigster a middleware.)
		return locale.GetMessage(format, args...)
	}

	return fmt.Sprintf(format, args...)
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

const contentTypeContextKey = "_iris_content_type"

func shouldAppendCharset(cType string) bool {
	return cType != ContentBinaryHeaderValue && cType != ContentWebassemblyHeaderValue
}

func (ctx *context) contentTypeOnce(cType string, charset string) {
	if charset == "" {
		charset = ctx.Application().ConfigurationReadOnly().GetCharset()
	}

	if shouldAppendCharset(cType) {
		cType += "; charset=" + charset
	}

	ctx.Values().Set(contentTypeContextKey, cType)
	ctx.writer.Header().Set(ContentTypeHeaderKey, cType)
}

// ContentType sets the response writer's header key "Content-Type" to the 'cType'.
func (ctx *context) ContentType(cType string) {
	if cType == "" {
		return
	}

	if _, wroteOnce := ctx.Values().GetEntry(contentTypeContextKey); wroteOnce {
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
		if shouldAppendCharset(cType) {
			cType += "; charset=" + ctx.Application().ConfigurationReadOnly().GetCharset()
		}
	}

	ctx.writer.Header().Set(ContentTypeHeaderKey, cType)
}

// GetContentType returns the response writer's header value of "Content-Type"
// which may, set before with the 'ContentType'.
func (ctx *context) GetContentType() string {
	return ctx.writer.Header().Get(ContentTypeHeaderKey)
}

// GetContentType returns the request's header value of "Content-Type".
func (ctx *context) GetContentTypeRequested() string {
	return ctx.GetHeader(ContentTypeHeaderKey)
}

// GetContentLength returns the request's header value of "Content-Length".
// Returns 0 if header was unable to be found or its value was not a valid number.
func (ctx *context) GetContentLength() int64 {
	if v := ctx.GetHeader(ContentLengthHeaderKey); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	return 0
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

// URLParam returns true if the url parameter exists, otherwise false.
func (ctx *context) URLParamExists(name string) bool {
	if q := ctx.request.URL.Query(); q != nil {
		_, exists := q[name]
		return exists
	}

	return false
}

// URLParamDefault returns the get parameter from a request, if not found then "def" is returned.
func (ctx *context) URLParamDefault(name string, def string) string {
	if v := ctx.request.URL.Query().Get(name); v != "" {
		return v
	}

	return def
}

// URLParam returns the get parameter from a request, if any.
func (ctx *context) URLParam(name string) string {
	return ctx.URLParamDefault(name, "")
}

// URLParamTrim returns the url query parameter with trailing white spaces removed from a request.
func (ctx *context) URLParamTrim(name string) string {
	return strings.TrimSpace(ctx.URLParam(name))
}

// URLParamEscape returns the escaped url query parameter from a request.
func (ctx *context) URLParamEscape(name string) string {
	return DecodeQuery(ctx.URLParam(name))
}

// ErrNotFound is the type error which API users can make use of
// to check if a `Context` action of a `Handler` is type of Not Found,
// e.g. URL Query Parameters.
// Example:
//
// n, err := context.URLParamInt("url_query_param_name")
// if errors.Is(err, context.ErrNotFound) {
// 	// [handle error...]
// }
// Another usage would be `err == context.ErrNotFound`
// HOWEVER prefer use the new `errors.Is` as API details may change in the future.
var ErrNotFound = errors.New("not found")

// URLParamInt returns the url query parameter as int value from a request,
// returns -1 and an error if parse failed or not found.
func (ctx *context) URLParamInt(name string) (int, error) {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return -1, err
		}
		return n, nil
	}

	return -1, ErrNotFound
}

// URLParamIntDefault returns the url query parameter as int value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *context) URLParamIntDefault(name string, def int) int {
	v, err := ctx.URLParamInt(name)
	if err != nil {
		return def
	}

	return v
}

// URLParamInt32Default returns the url query parameter as int32 value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *context) URLParamInt32Default(name string, def int32) int32 {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return def
		}

		return int32(n)
	}

	return def
}

// URLParamInt64 returns the url query parameter as int64 value from a request,
// returns -1 and an error if parse failed or not found.
func (ctx *context) URLParamInt64(name string) (int64, error) {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return -1, err
		}
		return n, nil
	}

	return -1, ErrNotFound
}

// URLParamInt64Default returns the url query parameter as int64 value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *context) URLParamInt64Default(name string, def int64) int64 {
	v, err := ctx.URLParamInt64(name)
	if err != nil {
		return def
	}

	return v
}

// URLParamFloat64 returns the url query parameter as float64 value from a request,
// returns an error and -1 if parse failed.
func (ctx *context) URLParamFloat64(name string) (float64, error) {
	if v := ctx.URLParam(name); v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return -1, err
		}
		return n, nil
	}

	return -1, ErrNotFound
}

// URLParamFloat64Default returns the url query parameter as float64 value from a request,
// if not found or parse failed then "def" is returned.
func (ctx *context) URLParamFloat64Default(name string, def float64) float64 {
	v, err := ctx.URLParamFloat64(name)
	if err != nil {
		return def
	}

	return v
}

// URLParamBool returns the url query parameter as boolean value from a request,
// returns an error if parse failed.
func (ctx *context) URLParamBool(name string) (bool, error) {
	return strconv.ParseBool(ctx.URLParam(name))
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

// No need anymore, net/http checks for the Form already.
// func (ctx *context) askParseForm() error {
// 	if ctx.request.Form == nil {
// 		if err := ctx.request.ParseForm(); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// FormValueDefault returns a single parsed form value by its "name",
// including both the URL field's query parameters and the POST or PUT form data.
//
// Returns the "def" if not found.
func (ctx *context) FormValueDefault(name string, def string) string {
	if form, has := ctx.form(); has {
		if v := form[name]; len(v) > 0 {
			return v[0]
		}
	}
	return def
}

// FormValueDefault retruns a single parsed form value.
func FormValueDefault(r *http.Request, name string, def string, postMaxMemory int64, resetBody bool) string {
	if form, has := GetForm(r, postMaxMemory, resetBody); has {
		if v := form[name]; len(v) > 0 {
			return v[0]
		}
	}
	return def
}

// FormValue returns a single parsed form value by its "name",
// including both the URL field's query parameters and the POST or PUT form data.
func (ctx *context) FormValue(name string) string {
	return ctx.FormValueDefault(name, "")
}

// FormValues returns the parsed form data, including both the URL
// field's query parameters and the POST or PUT form data.
//
// The default form's memory maximum size is 32MB, it can be changed by the
// `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
// NOTE: A check for nil is necessary.
func (ctx *context) FormValues() map[string][]string {
	form, _ := ctx.form()
	return form
}

// Form contains the parsed form data, including both the URL
// field's query parameters and the POST or PUT form data.
func (ctx *context) form() (form map[string][]string, found bool) {
	return GetForm(ctx.request, ctx.Application().ConfigurationReadOnly().GetPostMaxMemory(), ctx.Application().ConfigurationReadOnly().GetDisableBodyConsumptionOnUnmarshal())
}

// GetForm returns the request form (url queries, post or multipart) values.
func GetForm(r *http.Request, postMaxMemory int64, resetBody bool) (form map[string][]string, found bool) {
	/*
		net/http/request.go#1219
		for k, v := range f.Value {
			r.Form[k] = append(r.Form[k], v...)
			// r.PostForm should also be populated. See Issue 9305.
			r.PostForm[k] = append(r.PostForm[k], v...)
		}
	*/

	if form := r.Form; len(form) > 0 {
		return form, true
	}

	if form := r.PostForm; len(form) > 0 {
		return form, true
	}

	if m := r.MultipartForm; m != nil {
		if len(m.Value) > 0 {
			return m.Value, true
		}
	}

	var bodyCopy []byte

	if resetBody {
		// on POST, PUT and PATCH it will read the form values from request body otherwise from URL queries.
		if m := r.Method; m == "POST" || m == "PUT" || m == "PATCH" {
			bodyCopy, _ = GetBody(r, resetBody)
			if len(bodyCopy) == 0 {
				return nil, false
			}
			// r.Body = ioutil.NopCloser(io.TeeReader(r.Body, buf))
		} else {
			resetBody = false
		}
	}

	// ParseMultipartForm calls `request.ParseForm` automatically
	// therefore we don't need to call it here, although it doesn't hurt.
	// After one call to ParseMultipartForm or ParseForm,
	// subsequent calls have no effect, are idempotent.
	err := r.ParseMultipartForm(postMaxMemory)
	if resetBody {
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyCopy))
	}
	if err != nil && err != http.ErrNotMultipart {
		return nil, false
	}

	if form := r.Form; len(form) > 0 {
		return form, true
	}

	if form := r.PostForm; len(form) > 0 {
		return form, true
	}

	if m := r.MultipartForm; m != nil {
		if len(m.Value) > 0 {
			return m.Value, true
		}
	}

	return nil, false
}

// PostValueDefault returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name".
//
// If not found then "def" is returned instead.
func (ctx *context) PostValueDefault(name string, def string) string {
	ctx.form()
	if v := ctx.request.PostForm[name]; len(v) > 0 {
		return v[0]
	}
	return def
}

// PostValue returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name"
func (ctx *context) PostValue(name string) string {
	return ctx.PostValueDefault(name, "")
}

// PostValueTrim returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name",  without trailing spaces.
func (ctx *context) PostValueTrim(name string) string {
	return strings.TrimSpace(ctx.PostValue(name))
}

// PostValueInt returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as int.
//
// If not found returns -1 and a non-nil error.
func (ctx *context) PostValueInt(name string) (int, error) {
	v := ctx.PostValue(name)
	if v == "" {
		return -1, ErrNotFound
	}
	return strconv.Atoi(v)
}

// PostValueIntDefault returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as int.
//
// If not found or parse errors returns the "def".
func (ctx *context) PostValueIntDefault(name string, def int) int {
	if v, err := ctx.PostValueInt(name); err == nil {
		return v
	}

	return def
}

// PostValueInt64 returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as float64.
//
// If not found returns -1 and a non-nil error.
func (ctx *context) PostValueInt64(name string) (int64, error) {
	v := ctx.PostValue(name)
	if v == "" {
		return -1, ErrNotFound
	}
	return strconv.ParseInt(v, 10, 64)
}

// PostValueInt64Default returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as int64.
//
// If not found or parse errors returns the "def".
func (ctx *context) PostValueInt64Default(name string, def int64) int64 {
	if v, err := ctx.PostValueInt64(name); err == nil {
		return v
	}

	return def
}

// PostValueInt64Default returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as float64.
//
// If not found returns -1 and a non-nil error.
func (ctx *context) PostValueFloat64(name string) (float64, error) {
	v := ctx.PostValue(name)
	if v == "" {
		return -1, ErrNotFound
	}
	return strconv.ParseFloat(v, 64)
}

// PostValueInt64Default returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as float64.
//
// If not found or parse errors returns the "def".
func (ctx *context) PostValueFloat64Default(name string, def float64) float64 {
	if v, err := ctx.PostValueFloat64(name); err == nil {
		return v
	}

	return def
}

// PostValueInt64Default returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as bool.
//
// If not found or value is false, then it returns false, otherwise true.
func (ctx *context) PostValueBool(name string) (bool, error) {
	v := ctx.PostValue(name)
	if v == "" {
		return false, ErrNotFound
	}

	return strconv.ParseBool(v)
}

// PostValues returns all the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name" as a string slice.
//
// The default form's memory maximum size is 32MB, it can be changed by the
// `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
func (ctx *context) PostValues(name string) []string {
	ctx.form()
	return ctx.request.PostForm[name]
}

// FormFile returns the first uploaded file that received from the client.
//
//
// The default form's memory maximum size is 32MB, it can be changed by the
// `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/http_request/upload-file
func (ctx *context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	// we don't have access to see if the request is body stream
	// and then the ParseMultipartForm can be useless
	// here but do it in order to apply the post limit,
	// the internal request.FormFile will not do it if that's filled
	// and it's not a stream body.
	if err := ctx.request.ParseMultipartForm(ctx.Application().ConfigurationReadOnly().GetPostMaxMemory()); err != nil {
		return nil, nil, err
	}

	return ctx.request.FormFile(key)
}

// UploadFormFiles uploads any received file(s) from the client
// to the system physical location "destDirectory".
//
// The second optional argument "before" gives caller the chance to
// modify the *miltipart.FileHeader before saving to the disk,
// it can be used to change a file's name based on the current request,
// all FileHeader's options can be changed. You can ignore it if
// you don't need to use this capability before saving a file to the disk.
//
// Note that it doesn't check if request body streamed.
//
// Returns the copied length as int64 and
// a not nil error if at least one new file
// can't be created due to the operating system's permissions or
// http.ErrMissingFile if no file received.
//
// If you want to receive & accept files and manage them manually you can use the `context#FormFile`
// instead and create a copy function that suits your needs, the below is for generic usage.
//
// The default form's memory maximum size is 32MB, it can be changed by the
//  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
//
// See `FormFile` to a more controlled to receive a file.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/http_request/upload-files
func (ctx *context) UploadFormFiles(destDirectory string, before ...func(Context, *multipart.FileHeader)) (n int64, err error) {
	err = ctx.request.ParseMultipartForm(ctx.Application().ConfigurationReadOnly().GetPostMaxMemory())
	if err != nil {
		return 0, err
	}

	if ctx.request.MultipartForm != nil {
		if fhs := ctx.request.MultipartForm.File; fhs != nil {
			for _, files := range fhs {
				for _, file := range files {

					for _, b := range before {
						b(ctx, file)
					}

					n0, err0 := uploadTo(file, destDirectory)
					if err0 != nil {
						return 0, err0
					}
					n += n0
				}
			}
			return n, nil
		}
	}

	return 0, http.ErrMissingFile
}

func uploadTo(fh *multipart.FileHeader, destDirectory string) (int64, error) {
	src, err := fh.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()

	out, err := os.OpenFile(filepath.Join(destDirectory, fh.Filename),
		os.O_WRONLY|os.O_CREATE, os.FileMode(0666))
	if err != nil {
		return 0, err
	}
	defer out.Close()

	return io.Copy(out, src)
}

// AbsoluteURI parses the "s" and returns its absolute URI form.
func (ctx *context) AbsoluteURI(s string) string {
	if s == "" {
		return ""
	}

	if s[0] == '/' {
		scheme := ctx.request.URL.Scheme
		if scheme == "" {
			if ctx.request.TLS != nil {
				scheme = "https:"
			} else {
				scheme = "http:"
			}
		}

		host := ctx.Host()

		return scheme + "//" + host + path.Clean(s)
	}

	if u, err := url.Parse(s); err == nil {
		r := ctx.request

		if u.Scheme == "" && u.Host == "" {
			oldpath := r.URL.Path
			if oldpath == "" {
				oldpath = "/"
			}

			if s == "" || s[0] != '/' {
				olddir, _ := path.Split(oldpath)
				s = olddir + s
			}

			var query string
			if i := strings.Index(s, "?"); i != -1 {
				s, query = s[:i], s[i:]
			}

			// clean up but preserve trailing slash
			trailing := strings.HasSuffix(s, "/")
			s = path.Clean(s)
			if trailing && !strings.HasSuffix(s, "/") {
				s += "/"
			}
			s += query
		}
	}

	return s
}

// Redirect sends a redirect response to the client
// to a specific url or relative path.
// accepts 2 parameters string and an optional int
// first parameter is the url to redirect
// second parameter is the http status should send,
// default is 302 (StatusFound),
// you can set it to 301 (Permant redirect)
// or 303 (StatusSeeOther) if POST method,
// or StatusTemporaryRedirect(307) if that's nessecery.
func (ctx *context) Redirect(urlToRedirect string, statusHeader ...int) {
	ctx.StopExecution()
	// get the previous status code given by the end-developer.
	status := ctx.GetStatusCode()
	if status < 300 { // the previous is not a RCF-valid redirect status.
		status = 0
	}

	if len(statusHeader) > 0 {
		// check if status code is passed via receivers.
		if s := statusHeader[0]; s > 0 {
			status = s
		}
	}
	if status == 0 {
		// if status remains zero then default it.
		// a 'temporary-redirect-like' which works better than for our purpose
		status = http.StatusFound
	}

	http.Redirect(ctx.writer, ctx.request, urlToRedirect, status)
}

//  +------------------------------------------------------------+
//  | Body Readers                                               |
//  +------------------------------------------------------------+

// SetMaxRequestBodySize sets a limit to the request body size
// should be called before reading the request body from the client.
func (ctx *context) SetMaxRequestBodySize(limitOverBytes int64) {
	ctx.request.Body = http.MaxBytesReader(ctx.writer, ctx.request.Body, limitOverBytes)
}

// GetBody reads and returns the request body.
// The default behavior for the http request reader is to consume the data readen
// but you can change that behavior by passing the `WithoutBodyConsumptionOnUnmarshal` iris option.
//
// However, whenever you can use the `ctx.Request().Body` instead.
func (ctx *context) GetBody() ([]byte, error) {
	return GetBody(ctx.request, ctx.Application().ConfigurationReadOnly().GetDisableBodyConsumptionOnUnmarshal())
}

// GetBody reads and returns the request body.
func GetBody(r *http.Request, resetBody bool) ([]byte, error) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	if resetBody {
		// * remember, Request.Body has no Bytes(), we have to consume them first
		// and after re-set them to the body, this is the only solution.
		r.Body = ioutil.NopCloser(bytes.NewBuffer(data))
	}

	return data, nil
}

// UnmarshalBody reads the request's body and binds it to a value or pointer of any type
// Examples of usage: context.ReadJSON, context.ReadXML.
//
// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-custom-via-unmarshaler/main.go
//
// UnmarshalBody does not check about gzipped data.
// Do not rely on compressed data incoming to your server. The main reason is: https://en.wikipedia.org/wiki/Zip_bomb
// However you are still free to read the `ctx.Request().Body io.Reader` manually.
func (ctx *context) UnmarshalBody(outPtr interface{}, unmarshaler Unmarshaler) error {
	if ctx.request.Body == nil {
		return fmt.Errorf("unmarshal: empty body: %w", ErrNotFound)
	}

	rawData, err := ctx.GetBody()
	if err != nil {
		return err
	}

	// check if the v contains its own decode
	// in this case the v should be a pointer also,
	// but this is up to the user's custom Decode implementation*
	//
	// See 'BodyDecoder' for more.
	if decoder, isDecoder := outPtr.(BodyDecoder); isDecoder {
		return decoder.Decode(rawData)
	}

	// // check if v is already a pointer, if yes then pass as it's
	// if reflect.TypeOf(v).Kind() == reflect.Ptr {
	// 	return unmarshaler.Unmarshal(rawData, v)
	// } <- no need for that, ReadJSON is documented enough to receive a pointer,
	// we don't need to reduce the performance here by using the reflect.TypeOf method.

	// f the v doesn't contains a self-body decoder use the custom unmarshaler to bind the body.
	return unmarshaler.Unmarshal(rawData, outPtr)
}

func (ctx *context) shouldOptimize() bool {
	return ctx.Application().ConfigurationReadOnly().GetEnableOptimizations()
}

// ReadJSON reads JSON from request's body and binds it to a value of any json-valid type.
//
// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-json/main.go
func (ctx *context) ReadJSON(outPtr interface{}) error {
	unmarshaler := json.Unmarshal
	if ctx.shouldOptimize() {
		unmarshaler = jsoniter.Unmarshal
	}
	return ctx.UnmarshalBody(outPtr, UnmarshalerFunc(unmarshaler))
}

// ReadXML reads XML from request's body and binds it to a value of any xml-valid type.
//
// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-xml/main.go
func (ctx *context) ReadXML(outPtr interface{}) error {
	return ctx.UnmarshalBody(outPtr, UnmarshalerFunc(xml.Unmarshal))
}

// ReadYAML reads YAML from request's body and binds it to the "outPtr" value.
//
// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-yaml/main.go
func (ctx *context) ReadYAML(outPtr interface{}) error {
	return ctx.UnmarshalBody(outPtr, UnmarshalerFunc(yaml.Unmarshal))
}

// IsErrPath can be used at `context#ReadForm` and `context#ReadQuery`.
// It reports whether the incoming error
// can be ignored when server allows unknown post values to be sent by the client.
//
// A shortcut for the `schema#IsErrPath`.
var IsErrPath = schema.IsErrPath

// ReadForm binds the formObject  with the form data
// it supports any kind of type, including custom structs.
// It will return nothing if request data are empty.
// The struct field tag is "form".
//
// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-form/main.go
func (ctx *context) ReadForm(formObject interface{}) error {
	values := ctx.FormValues()
	if len(values) == 0 {
		return nil
	}

	return schema.DecodeForm(values, formObject)
}

// ReadQuery binds the "ptr" with the url query string. The struct field tag is "url".
//
// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-query/main.go
func (ctx *context) ReadQuery(ptr interface{}) error {
	values := ctx.request.URL.Query()
	if len(values) == 0 {
		return nil
	}

	return schema.DecodeQuery(values, ptr)
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

const (
	// ContentTypeHeaderKey is the header key of "Content-Type".
	ContentTypeHeaderKey = "Content-Type"

	// LastModifiedHeaderKey is the header key of "Last-Modified".
	LastModifiedHeaderKey = "Last-Modified"
	// IfModifiedSinceHeaderKey is the header key of "If-Modified-Since".
	IfModifiedSinceHeaderKey = "If-Modified-Since"
	// CacheControlHeaderKey is the header key of "Cache-Control".
	CacheControlHeaderKey = "Cache-Control"
	// ETagHeaderKey is the header key of "ETag".
	ETagHeaderKey = "ETag"

	// ContentDispositionHeaderKey is the header key of "Content-Disposition".
	ContentDispositionHeaderKey = "Content-Disposition"
	// ContentLengthHeaderKey is the header key of "Content-Length"
	ContentLengthHeaderKey = "Content-Length"
	// ContentEncodingHeaderKey is the header key of "Content-Encoding".
	ContentEncodingHeaderKey = "Content-Encoding"
	// GzipHeaderValue is the header value of "gzip".
	GzipHeaderValue = "gzip"
	// AcceptEncodingHeaderKey is the header key of "Accept-Encoding".
	AcceptEncodingHeaderKey = "Accept-Encoding"
	// VaryHeaderKey is the header key of "Vary".
	VaryHeaderKey = "Vary"
)

var unixEpochTime = time.Unix(0, 0)

// IsZeroTime reports whether t is obviously unspecified (either zero or Unix()=0).
func IsZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

// ParseTime parses a time header (such as the Date: header),
// trying each forth formats (or three if Application's configuration's TimeFormat is defaulted)
// that are allowed by HTTP/1.1:
// Application's configuration's TimeFormat or/and http.TimeFormat,
// time.RFC850, and time.ANSIC.
//
// Look `context#FormatTime` for the opossite operation (Time to string).
var ParseTime = func(ctx Context, text string) (t time.Time, err error) {
	t, err = time.Parse(ctx.Application().ConfigurationReadOnly().GetTimeFormat(), text)
	if err != nil {
		return http.ParseTime(text)
	}

	return
}

// FormatTime returns a textual representation of the time value formatted
// according to the Application's configuration's TimeFormat field
// which defines the format.
//
// Look `context#ParseTime` for the opossite operation (string to Time).
var FormatTime = func(ctx Context, t time.Time) string {
	return t.Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
}

// SetLastModified sets the "Last-Modified" based on the "modtime" input.
// If "modtime" is zero then it does nothing.
//
// It's mostly internally on core/router and context packages.
func (ctx *context) SetLastModified(modtime time.Time) {
	if !IsZeroTime(modtime) {
		ctx.Header(LastModifiedHeaderKey, FormatTime(ctx, modtime.UTC())) // or modtime.UTC()?
	}
}

// ErrPreconditionFailed may be returned from `Context` methods
// that has to perform one or more client side preconditions before the actual check, e.g. `CheckIfModifiedSince`.
// Usage:
// ok, err := context.CheckIfModifiedSince(modTime)
// if err != nil {
//    if errors.Is(err, context.ErrPreconditionFailed) {
//         [handle missing client conditions,such as not valid request method...]
//     }else {
//         [the error is probably a time parse error...]
//    }
// }
var ErrPreconditionFailed = errors.New("precondition failed")

// CheckIfModifiedSince checks if the response is modified since the "modtime".
// Note that it has nothing to do with server-side caching.
// It does those checks by checking if the "If-Modified-Since" request header
// sent by client or a previous server response header
// (e.g with WriteWithExpiration or HandleDir or Favicon etc.)
// is a valid one and it's before the "modtime".
//
// A check for !modtime && err == nil is necessary to make sure that
// it's not modified since, because it may return false but without even
// had the chance to check the client-side (request) header due to some errors,
// like the HTTP Method is not "GET" or "HEAD" or if the "modtime" is zero
// or if parsing time from the header failed. See `ErrPreconditionFailed` too.
//
// It's mostly used internally, e.g. `context#WriteWithExpiration`.
func (ctx *context) CheckIfModifiedSince(modtime time.Time) (bool, error) {
	if method := ctx.Method(); method != http.MethodGet && method != http.MethodHead {
		return false, fmt.Errorf("method: %w", ErrPreconditionFailed)
	}
	ims := ctx.GetHeader(IfModifiedSinceHeaderKey)
	if ims == "" || IsZeroTime(modtime) {
		return false, fmt.Errorf("zero time: %w", ErrPreconditionFailed)
	}
	t, err := ParseTime(ctx, ims)
	if err != nil {
		return false, err
	}
	// sub-second precision, so
	// use mtime < t+1s instead of mtime <= t to check for unmodified.
	if modtime.UTC().Before(t.Add(1 * time.Second)) {
		return false, nil
	}
	return true, nil
}

// WriteNotModified sends a 304 "Not Modified" status code to the client,
// it makes sure that the content type, the content length headers
// and any "ETag" are removed before the response sent.
//
// It's mostly used internally on core/router/fs.go and context methods.
func (ctx *context) WriteNotModified() {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g.," Last-Modified" might be useful if the
	// response does not have an ETag field).
	h := ctx.ResponseWriter().Header()
	delete(h, ContentTypeHeaderKey)
	delete(h, ContentLengthHeaderKey)
	if h.Get(ETagHeaderKey) != "" {
		delete(h, LastModifiedHeaderKey)
	}
	ctx.StatusCode(http.StatusNotModified)
}

// WriteWithExpiration works like `Write` but it will check if a resource is modified,
// based on the "modtime" input argument,
// otherwise sends a 304 status code in order to let the client-side render the cached content.
func (ctx *context) WriteWithExpiration(body []byte, modtime time.Time) (int, error) {
	if modified, err := ctx.CheckIfModifiedSince(modtime); !modified && err == nil {
		ctx.WriteNotModified()
		return 0, nil
	}

	ctx.SetLastModified(modtime)
	return ctx.writer.Write(body)
}

// StreamWriter registers the given stream writer for populating
// response body.
//
// Access to context's and/or its' members is forbidden from writer.
//
// This function may be used in the following cases:
//
//     * if response body is too big (more than iris.LimitRequestBodySize(if set)).
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
	if h := ctx.GetHeader(AcceptEncodingHeaderKey); h != "" {
		for _, v := range strings.Split(h, ";") {
			if strings.Contains(v, GzipHeaderValue) { // we do Contains because sometimes browsers has the q=, we don't use it atm. || strings.Contains(v,"deflate"){
				return true
			}
		}
	}
	return false
}

// ErrGzipNotSupported may be returned from `WriteGzip` methods if
// the client does not support the "gzip" compression.
var ErrGzipNotSupported = errors.New("client does not support gzip compression")

// WriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
// returns the number of bytes written and an error ( if the client doesn't support gzip compression)
//
// You may re-use this function in the same handler
// to write more data many times without any troubles.
func (ctx *context) WriteGzip(b []byte) (int, error) {
	if !ctx.ClientSupportsGzip() {
		return 0, ErrGzipNotSupported
	}

	return ctx.GzipResponseWriter().Write(b)
}

// TryWriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
// If client does not supprots gzip then the contents are written as they are, uncompressed.
func (ctx *context) TryWriteGzip(b []byte) (int, error) {
	n, err := ctx.WriteGzip(b)
	if err != nil {
		// check if the error came from gzip not allowed and not the writer itself
		if errors.Is(err, ErrGzipNotSupported) {
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
// Note that the 'layoutTmplFile' argument can be set to iris.NoLayout || view.NoLayout || context.NoLayout
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

// View renders a template based on the registered view engine(s).
// First argument accepts the filename, relative to the view engine's Directory and Extension,
// i.e: if directory is "./templates" and want to render the "./templates/users/index.html"
// then you pass the "users/index.html" as the filename argument.
//
// The second optional argument can receive a single "view model"
// that will be binded to the view template if it's not nil,
// otherwise it will check for previous view data stored by the `ViewData`
// even if stored at any previous handler(middleware) for the same request.
//
// Look .ViewData and .ViewLayout too.
//
// Examples: https://github.com/kataras/iris/tree/master/_examples/view
func (ctx *context) View(filename string, optionalViewModel ...interface{}) error {
	ctx.ContentType(ContentHTMLHeaderValue)
	cfg := ctx.Application().ConfigurationReadOnly()

	layout := ctx.values.GetString(cfg.GetViewLayoutContextKey())

	var bindingData interface{}
	if len(optionalViewModel) > 0 {
		// a nil can override the existing data or model sent by `ViewData`.
		bindingData = optionalViewModel[0]
	} else {
		bindingData = ctx.values.Get(cfg.GetViewDataContextKey())
	}

	err := ctx.Application().View(ctx, filename, layout, bindingData)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.StopExecution()
	}

	return err
}

const (
	// ContentBinaryHeaderValue header value for binary data.
	ContentBinaryHeaderValue = "application/octet-stream"
	// ContentWebassemblyHeaderValue header value for web assembly files.
	ContentWebassemblyHeaderValue = "application/wasm"
	// ContentHTMLHeaderValue is the  string of text/html response header's content type value.
	ContentHTMLHeaderValue = "text/html"
	// ContentJSONHeaderValue header value for JSON data.
	ContentJSONHeaderValue = "application/json"
	// ContentJSONProblemHeaderValue header value for JSON API problem error.
	// Read more at: https://tools.ietf.org/html/rfc7807
	ContentJSONProblemHeaderValue = "application/problem+json"
	// ContentXMLProblemHeaderValue header value for XML API problem error.
	// Read more at: https://tools.ietf.org/html/rfc7807
	ContentXMLProblemHeaderValue = "application/problem+xml"
	// ContentJavascriptHeaderValue header value for JSONP & Javascript data.
	ContentJavascriptHeaderValue = "application/javascript"
	// ContentTextHeaderValue header value for Text data.
	ContentTextHeaderValue = "text/plain"
	// ContentXMLHeaderValue header value for XML data.
	ContentXMLHeaderValue = "text/xml"
	// ContentXMLUnreadableHeaderValue obselete header value for XML.
	ContentXMLUnreadableHeaderValue = "application/xml"
	// ContentMarkdownHeaderValue custom key/content type, the real is the text/html.
	ContentMarkdownHeaderValue = "text/markdown"
	// ContentYAMLHeaderValue header value for YAML data.
	ContentYAMLHeaderValue = "application/x-yaml"
	// ContentFormHeaderValue header value for post form data.
	ContentFormHeaderValue = "application/x-www-form-urlencoded"
	// ContentFormMultipartHeaderValue header value for post multipart form data.
	ContentFormMultipartHeaderValue = "multipart/form-data"
)

// Binary writes out the raw bytes as binary data.
func (ctx *context) Binary(data []byte) (int, error) {
	ctx.ContentType(ContentBinaryHeaderValue)
	return ctx.Write(data)
}

// Text writes out a string as plain text.
func (ctx *context) Text(format string, args ...interface{}) (int, error) {
	ctx.ContentType(ContentTextHeaderValue)
	return ctx.Writef(format, args...)
}

// HTML writes out a string as text/html.
func (ctx *context) HTML(format string, args ...interface{}) (int, error) {
	ctx.ContentType(ContentHTMLHeaderValue)
	return ctx.Writef(format, args...)
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

// DefaultJSONOptions is the optional settings that are being used
// inside `ctx.JSON`.
var DefaultJSONOptions = JSON{}

// JSON marshals the given interface object and writes the JSON response to the client.
func (ctx *context) JSON(v interface{}, opts ...JSON) (n int, err error) {
	options := DefaultJSONOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	ctx.ContentType(ContentJSONHeaderValue)

	if options.StreamingJSON {
		if ctx.shouldOptimize() {
			jsoniterConfig := jsoniter.Config{
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
			ctx.Application().Logger().Debugf("JSON: %v", err)
			ctx.StatusCode(http.StatusInternalServerError) // it handles the fallback to normal mode here which also removes the gzip headers.
			return 0, err
		}
		return ctx.writer.Written(), err
	}

	n, err = WriteJSON(ctx.writer, v, options, ctx.shouldOptimize())
	if err != nil {
		ctx.Application().Logger().Debugf("JSON: %v", err)
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

var finishCallbackB = []byte(");")

// WriteJSONP marshals the given interface object and writes the JSON response to the writer.
func WriteJSONP(writer io.Writer, v interface{}, options JSONP, enableOptimization ...bool) (int, error) {
	if callback := options.Callback; callback != "" {
		n, err := writer.Write([]byte(callback + "("))
		if err != nil {
			return n, err
		}
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

// DefaultJSONPOptions is the optional settings that are being used
// inside `ctx.JSONP`.
var DefaultJSONPOptions = JSONP{}

// JSONP marshals the given interface object and writes the JSON response to the client.
func (ctx *context) JSONP(v interface{}, opts ...JSONP) (int, error) {
	options := DefaultJSONPOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	ctx.ContentType(ContentJavascriptHeaderValue)

	n, err := WriteJSONP(ctx.writer, v, options, ctx.shouldOptimize())
	if err != nil {
		ctx.Application().Logger().Debugf("JSONP: %v", err)
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

type xmlMapEntry struct {
	XMLName xml.Name
	Value   interface{} `xml:",chardata"`
}

// XMLMap wraps a map[string]interface{} to compatible xml marshaler,
// in order to be able to render maps as XML on the `Context.XML` method.
//
// Example: `Context.XML(XMLMap("Root", map[string]interface{}{...})`.
func XMLMap(elementName string, v Map) xml.Marshaler {
	return xmlMap{
		entries:     v,
		elementName: elementName,
	}
}

type xmlMap struct {
	entries     Map
	elementName string
}

// MarshalXML marshals a map to XML.
func (m xmlMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(m.entries) == 0 {
		return nil
	}

	start.Name = xml.Name{Local: m.elementName}
	err := e.EncodeToken(start)
	if err != nil {
		return err
	}

	for k, v := range m.entries {
		err = e.Encode(xmlMapEntry{XMLName: xml.Name{Local: k}, Value: v})
		if err != nil {
			return err
		}
	}

	return e.EncodeToken(start.End())
}

// WriteXML marshals the given interface object and writes the XML response to the writer.
func WriteXML(writer io.Writer, v interface{}, options XML) (int, error) {
	if prefix := options.Prefix; prefix != "" {
		n, err := writer.Write([]byte(prefix))
		if err != nil {
			return n, err
		}
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

// DefaultXMLOptions is the optional settings that are being used
// from `ctx.XML`.
var DefaultXMLOptions = XML{}

// XML marshals the given interface object and writes the XML response to the client.
// To render maps as XML see the `XMLMap` package-level function.
func (ctx *context) XML(v interface{}, opts ...XML) (int, error) {
	options := DefaultXMLOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	ctx.ContentType(ContentXMLHeaderValue)

	n, err := WriteXML(ctx.writer, v, options)
	if err != nil {
		ctx.Application().Logger().Debugf("XML: %v", err)
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

// Problem writes a JSON or XML problem response.
// Order of Problem fields are not always rendered the same.
//
// Behaves exactly like `Context.JSON`
// but with default ProblemOptions.JSON indent of " " and
// a response content type of "application/problem+json" instead.
//
// Use the options.RenderXML and XML fields to change this behavior and
// send a response of content type "application/problem+xml" instead.
//
// Read more at: https://github.com/kataras/iris/wiki/Routing-error-handlers
func (ctx *context) Problem(v interface{}, opts ...ProblemOptions) (int, error) {
	options := DefaultProblemOptions
	if len(opts) > 0 {
		options = opts[0]
		// Currently apply only if custom options passsed, otherwise,
		// with the current settings, it's not required.
		// This may change in the future though.
		options.Apply(ctx)
	}

	if p, ok := v.(Problem); ok {
		// if !p.Validate() {
		// 	ctx.StatusCode(http.StatusInternalServerError)
		// 	return ErrNotValidProblem
		// }
		p.updateURIsToAbs(ctx)
		code, _ := p.getStatus()
		ctx.StatusCode(code)

		if options.RenderXML {
			ctx.contentTypeOnce(ContentXMLProblemHeaderValue, "")
			// Problem is an xml Marshaler already, don't use `XMLMap`.
			return ctx.XML(v, options.XML)
		}
	}

	ctx.contentTypeOnce(ContentJSONProblemHeaderValue, "")
	return ctx.JSON(v, options.JSON)
}

// WriteMarkdown parses the markdown to html and writes these contents to the writer.
func WriteMarkdown(writer io.Writer, markdownB []byte, options Markdown) (int, error) {
	buf := blackfriday.Run(markdownB)
	if options.Sanitize {
		buf = bluemonday.UGCPolicy().SanitizeBytes(buf)
	}
	return writer.Write(buf)
}

// DefaultMarkdownOptions is the optional settings that are being used
// from `WriteMarkdown` and `ctx.Markdown`.
var DefaultMarkdownOptions = Markdown{}

// Markdown parses the markdown to html and renders its result to the client.
func (ctx *context) Markdown(markdownB []byte, opts ...Markdown) (int, error) {
	options := DefaultMarkdownOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	ctx.ContentType(ContentHTMLHeaderValue)

	n, err := WriteMarkdown(ctx.writer, markdownB, options)
	if err != nil {
		ctx.Application().Logger().Debugf("Markdown: %v", err)
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

// YAML marshals the "v" using the yaml marshaler and renders its result to the client.
func (ctx *context) YAML(v interface{}) (int, error) {
	out, err := yaml.Marshal(v)
	if err != nil {
		ctx.Application().Logger().Debugf("YAML: %v", err)
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	ctx.ContentType(ContentYAMLHeaderValue)
	return ctx.Write(out)
}

//  +-----------------------------------------------------------------------+
//  | Content Νegotiation                                                   |
//  | https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation |                                       |
//  +-----------------------------------------------------------------------+

// ErrContentNotSupported returns from the `Negotiate` method
// when server responds with 406.
var ErrContentNotSupported = errors.New("unsupported content")

// ContentSelector is the interface which structs can implement
// to manually choose a content based on the negotiated mime (content type).
// It can be passed to the `Context.Negotiate` method.
//
// See the `N` struct too.
type ContentSelector interface {
	SelectContent(mime string) interface{}
}

// ContentNegotiator is the interface which structs can implement
// to override the `Context.Negotiate` default implementation and
// manually respond to the client based on a manuall call of `Context.Negotiation().Build()`
// to get the final negotiated mime and charset.
// It can be passed to the `Context.Negotiate` method.
type ContentNegotiator interface {
	// mime and charset can be retrieved by:
	// mime, charset := Context.Negotiation().Build()
	// Pass this method to `Context.Negotiate` method
	// to write custom content.
	// Overriding the existing behavior of Context.Negotiate for selecting values based on
	// content types, although it can accept any custom mime type with []byte.
	// Content type is already set.
	// Use it with caution, 99.9% you don't need this but it's here for extreme cases.
	Negotiate(ctx Context) (int, error)
}

// N is a struct which can be passed on the `Context.Negotiate` method.
// It contains fields which should be filled based on the `Context.Negotiation()`
// server side values. If no matched mime then its "Other" field will be sent,
// which should be a string or []byte.
// It completes the `ContentSelector` interface.
type N struct {
	Text, HTML string
	Markdown   []byte
	Binary     []byte

	JSON    interface{}
	Problem Problem
	JSONP   interface{}
	XML     interface{}
	YAML    interface{}

	Other []byte // custom content types.
}

// SelectContent returns a content based on the matched negotiated "mime".
func (n N) SelectContent(mime string) interface{} {
	switch mime {
	case ContentTextHeaderValue:
		return n.Text
	case ContentHTMLHeaderValue:
		return n.HTML
	case ContentMarkdownHeaderValue:
		return n.Markdown
	case ContentBinaryHeaderValue:
		return n.Binary
	case ContentJSONHeaderValue:
		return n.JSON
	case ContentJSONProblemHeaderValue:
		return n.Problem
	case ContentJavascriptHeaderValue:
		return n.JSONP
	case ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue:
		return n.XML
	case ContentYAMLHeaderValue:
		return n.YAML
	default:
		return n.Other
	}
}

const negotiationContextKey = "_iris_negotiation_builder"

// Negotiation creates once and returns the negotiation builder
// to build server-side available prioritized content
// for specific content type(s), charset(s) and encoding algorithm(s).
//
// See `Negotiate` method too.
func (ctx *context) Negotiation() *NegotiationBuilder {
	if n := ctx.Values().Get(negotiationContextKey); n != nil {
		return n.(*NegotiationBuilder)
	}

	acceptBuilder := NegotiationAcceptBuilder{}
	acceptBuilder.accept = parseHeader(ctx.GetHeader("Accept"))
	acceptBuilder.charset = parseHeader(ctx.GetHeader("Accept-Charset"))

	n := &NegotiationBuilder{Accept: acceptBuilder}

	ctx.Values().Set(negotiationContextKey, n)

	return n
}

func parseHeader(headerValue string) []string {
	in := strings.Split(headerValue, ",")
	out := make([]string, 0, len(in))

	for _, value := range in {
		// remove any spaces and quality values such as ;q=0.8.
		v := strings.TrimSpace(strings.Split(value, ";")[0])
		if v != "" {
			out = append(out, v)
		}
	}

	return out
}

// Negotiate used for serving different representations of a resource at the same URI.
//
// The "v" can be a single `N` struct value.
// The "v" can be any value completes the `ContentSelector` interface.
// The "v" can be any value completes the `ContentNegotiator` interface.
// The "v" can be any value of struct(JSON, JSONP, XML, YAML) or
// string(TEXT, HTML) or []byte(Markdown, Binary) or []byte with any matched mime type.
//
// If the "v" is nil, the `Context.Negotitation()` builder's
// content will be used instead, otherwise "v" overrides builder's content
// (server mime types are still retrieved by its registered, supported, mime list)
//
// Set mime type priorities by `Negotiation().JSON().XML().HTML()...`.
// Set charset priorities by `Negotiation().Charset(...)`.
// Set encoding algorithm priorities by `Negotiation().Encoding(...)`.
// Modify the accepted by
// `Negotiation().Accept./Override()/.XML().JSON().Charset(...).Encoding(...)...`.
//
// It returns `ErrContentNotSupported` when not matched mime type(s).
//
// Resources:
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Charset
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Encoding
//
// Supports the above without quality values.
//
// Read more at: https://github.com/kataras/iris/wiki/Content-negotiation
func (ctx *context) Negotiate(v interface{}) (int, error) {
	contentType, charset, encoding, content := ctx.Negotiation().Build()
	if v == nil {
		v = content
	}

	if contentType == "" {
		// If the server cannot serve any matching set,
		// it can send back a 406 (Not Acceptable) error code.
		ctx.StatusCode(http.StatusNotAcceptable)
		return -1, ErrContentNotSupported
	}

	if charset == "" {
		charset = ctx.Application().ConfigurationReadOnly().GetCharset()
	}

	if encoding == "gzip" {
		ctx.Gzip(true)
	}

	ctx.contentTypeOnce(contentType, charset)

	if n, ok := v.(ContentNegotiator); ok {
		return n.Negotiate(ctx)
	}

	if s, ok := v.(ContentSelector); ok {
		v = s.SelectContent(contentType)
	}

	// switch v := value.(type) {
	// case []byte:
	// 	if contentType == ContentMarkdownHeaderValue {
	// 		return ctx.Markdown(v)
	// 	}

	// 	return ctx.Write(v)
	// case string:
	// 	return ctx.WriteString(v)
	// default:
	// make it switch by content-type only, but we lose custom mime types capability that way:
	//                                                 ^ solved with []byte on default case and
	//                                                 ^ N.Other and
	//                                                 ^ ContentSelector and ContentNegotiator interfaces.

	switch contentType {
	case ContentTextHeaderValue, ContentHTMLHeaderValue:
		return ctx.WriteString(v.(string))
	case ContentMarkdownHeaderValue:
		return ctx.Markdown(v.([]byte))
	case ContentJSONHeaderValue:
		return ctx.JSON(v)
	case ContentJSONProblemHeaderValue, ContentXMLProblemHeaderValue:
		return ctx.Problem(v)
	case ContentJavascriptHeaderValue:
		return ctx.JSONP(v)
	case ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue:
		return ctx.XML(v)
	case ContentYAMLHeaderValue:
		return ctx.YAML(v)
	default:
		// maybe "Other" or v is []byte or string but not a built-in framework mime,
		// for custom content types,
		// panic if not correct usage.
		switch vv := v.(type) {
		case []byte:
			return ctx.Write(vv)
		case string:
			return ctx.WriteString(vv)
		default:
			ctx.StatusCode(http.StatusNotAcceptable)
			return -1, ErrContentNotSupported
		}

	}
}

// NegotiationBuilder returns from the `Context.Negotitation`
// and can be used inside chain of handlers to build server-side
// mime type(s), charset(s) and encoding algorithm(s)
// that should match with the client's
// Accept, Accept-Charset and Accept-Encoding headers (by-default).
// To modify the client's accept use its "Accept" field
// which it's the `NegotitationAcceptBuilder`.
//
// See the `Negotiate` method too.
type NegotiationBuilder struct {
	Accept NegotiationAcceptBuilder

	mime     []string               // we need order.
	contents map[string]interface{} // map to the "mime" and content should be rendered if that mime requested.
	charset  []string
	encoding []string
}

// MIME registers a mime type and optionally the value that should be rendered
// through `Context.Negotiate` when this mime type is accepted by client.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) MIME(mime string, content interface{}) *NegotiationBuilder {
	mimes := parseHeader(mime) // if contains more than one sep by commas ",".
	if content == nil {
		n.mime = append(n.mime, mimes...)
		return n
	}

	if n.contents == nil {
		n.contents = make(map[string]interface{})
	}

	for _, m := range mimes {
		n.mime = append(n.mime, m)
		n.contents[m] = content
	}

	return n
}

// Text registers the "text/plain" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/plain" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Text(v ...string) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentTextHeaderValue, content)
}

// HTML registers the "text/html" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/html" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) HTML(v ...string) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentHTMLHeaderValue, content)
}

// Markdown registers the "text/markdown" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/markdown" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Markdown(v ...[]byte) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v
	}
	return n.MIME(ContentMarkdownHeaderValue, content)
}

// Binary registers the "application/octet-stream" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/octet-stream" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Binary(v ...[]byte) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentBinaryHeaderValue, content)
}

// JSON registers the "application/json" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/json" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) JSON(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentJSONHeaderValue, content)
}

// Problem registers the "application/problem+xml" or "application/problem+xml" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/problem+json" or the "application/problem+xml" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Problem(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentJSONProblemHeaderValue+","+ContentXMLProblemHeaderValue, content)
}

// JSONP registers the "application/javascript" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/javascript" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) JSONP(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentJavascriptHeaderValue, content)
}

// XML registers the "text/xml" and "application/xml" content types and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts one of the "text/xml" or "application/xml" content types.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) XML(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentXMLHeaderValue+","+ContentXMLUnreadableHeaderValue, content)
}

// YAML registers the "application/x-yaml" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/x-yaml" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) YAML(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentYAMLHeaderValue, content)
}

// Any registers a wildcard that can match any client's accept content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Any(v ...interface{}) *NegotiationBuilder {
	var content interface{}
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME("*", content)
}

// Charset overrides the application's config's charset (which defaults to "utf-8")
// that a client should match for
// (through Accept-Charset header or custom through `NegotitationBuilder.Accept.Override().Charset(...)` call).
// Do not set it if you don't know what you're doing.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Charset(charset ...string) *NegotiationBuilder {
	n.charset = append(n.charset, charset...)
	return n
}

// Encoding registers one or more encoding algorithms by name, i.e gzip, deflate.
// that a client should match for (through Accept-Encoding header).
//
// Only the "gzip" can be handlded automatically as it's the only builtin encoding algorithm
// to serve resources.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Encoding(encoding ...string) *NegotiationBuilder {
	n.encoding = append(n.encoding, encoding...)
	return n
}

// EncodingGzip registers the "gzip" encoding algorithm
// that a client should match for (through Accept-Encoding header or call of Accept.Encoding(enc)).
//
// It will make resources to served by "gzip" if Accept-Encoding contains the "gzip" as well.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) EncodingGzip() *NegotiationBuilder {
	return n.Encoding(GzipHeaderValue)
}

// Build calculates the client's and server's mime type(s), charset(s) and encoding
// and returns the final content type, charset and encoding that server should render
// to the client. It does not clear the fields, use the `Clear` method if neeeded.
//
// The returned "content" can be nil if the matched "contentType" does not provide any value,
// in that case the `Context.Negotiate(v)` must be called with a non-nil value.
func (n *NegotiationBuilder) Build() (contentType, charset, encoding string, content interface{}) {
	contentType = negotiationMatch(n.Accept.accept, n.mime)
	charset = negotiationMatch(n.Accept.charset, n.charset)
	encoding = negotiationMatch(n.Accept.encoding, n.encoding)

	if n.contents != nil {
		if data, ok := n.contents[contentType]; ok {
			content = data
		}
	}

	return
}

// Clear clears the prioritized mime type(s), charset(s) and any contents
// relative to those mime type(s).
// The "Accept" field is stay as it is, use its `Override` method
// to clear out the client's accepted mime type(s) and charset(s).
func (n *NegotiationBuilder) Clear() *NegotiationBuilder {
	n.mime = n.mime[0:0]
	n.contents = nil
	n.charset = n.charset[0:0]
	return n
}

func negotiationMatch(in []string, priorities []string) string {
	// e.g.
	// match json:
	// 	in: text/html, application/json
	// 	prioritities: application/json
	// not match:
	// 	in: text/html, application/json
	// 	prioritities: text/xml
	// match html:
	// 	in: text/html, application/json
	// 	prioritities: */*
	// not match:
	// 	in: application/json
	// 	prioritities: text/xml
	// match json:
	// 	in: text/html, application/*
	// 	prioritities: application/json

	if len(priorities) == 0 {
		return ""
	}

	if len(in) == 0 {
		return priorities[0]
	}

	for _, accepted := range in {
		for _, p := range priorities {
			// wildcard is */* or text/* and etc.
			// so loop through each char.
			for i, n := 0, len(accepted); i < n; i++ {
				if accepted[i] != p[i] {
					break
				}

				if accepted[i] == '*' || p[i] == '*' {
					return p
				}

				if i == n-1 {
					return p
				}
			}
		}
	}

	return ""
}

// NegotiationAcceptBuilder builds the accepted mime types and charset
//
// and "Accept-Charset" headers respectfully.
// The default values are set by the client side, server can append or override those.
// The end result will be challenged with runtime preffered set of content types and charsets.
//
// See the `Negotiate` method too.
type NegotiationAcceptBuilder struct {
	// initialized with "Accept" request header values.
	accept []string
	// initialized with "Accept-Encoding" request header. and if was empty then the
	// application's default (which defaults to utf-8).
	charset []string
	// initialized with "Accept-Encoding" request header values.
	encoding []string

	// To support override in request life cycle.
	// We need slice when data is the same format
	// for one or more mime types,
	// i.e text/xml and obselete application/xml.
	lastAccept   []string
	lastCharset  []string
	lastEncoding []string
}

// Override clears the default values for accept and accept charset.
// Returns itself.
func (n *NegotiationAcceptBuilder) Override() *NegotiationAcceptBuilder {
	// when called first.
	n.accept = n.accept[0:0]
	n.charset = n.charset[0:0]
	n.encoding = n.encoding[0:0]

	// when called after.
	if len(n.lastAccept) > 0 {
		n.accept = append(n.accept, n.lastAccept...)
		n.lastAccept = n.lastAccept[0:0]
	}

	if len(n.lastCharset) > 0 {
		n.charset = append(n.charset, n.lastCharset...)
		n.lastCharset = n.lastCharset[0:0]
	}

	if len(n.lastEncoding) > 0 {
		n.encoding = append(n.encoding, n.lastEncoding...)
		n.lastEncoding = n.lastEncoding[0:0]
	}

	return n
}

// MIME adds accepted client's mime type(s).
// Returns itself.
func (n *NegotiationAcceptBuilder) MIME(mimeType ...string) *NegotiationAcceptBuilder {
	n.lastAccept = mimeType
	n.accept = append(n.accept, mimeType...)
	return n
}

// Text adds the "text/plain" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Text() *NegotiationAcceptBuilder {
	return n.MIME(ContentTextHeaderValue)
}

// HTML adds the "text/html" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) HTML() *NegotiationAcceptBuilder {
	return n.MIME(ContentHTMLHeaderValue)
}

// Markdown adds the "text/markdown" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Markdown() *NegotiationAcceptBuilder {
	return n.MIME(ContentMarkdownHeaderValue)
}

// Binary adds the "application/octet-stream" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Binary() *NegotiationAcceptBuilder {
	return n.MIME(ContentBinaryHeaderValue)
}

// JSON adds the "application/json" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) JSON() *NegotiationAcceptBuilder {
	return n.MIME(ContentJSONHeaderValue)
}

// Problem adds the "application/problem+json" and "application/problem-xml"
// as accepted client content types.
// Returns itself.
func (n *NegotiationAcceptBuilder) Problem() *NegotiationAcceptBuilder {
	return n.MIME(ContentJSONProblemHeaderValue, ContentXMLProblemHeaderValue)
}

// JSONP adds the "application/javascript" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) JSONP() *NegotiationAcceptBuilder {
	return n.MIME(ContentJavascriptHeaderValue)
}

// XML adds the "text/xml" and "application/xml" as accepted client content types.
// Returns itself.
func (n *NegotiationAcceptBuilder) XML() *NegotiationAcceptBuilder {
	return n.MIME(ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue)
}

// YAML adds the "application/x-yaml" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) YAML() *NegotiationAcceptBuilder {
	return n.MIME(ContentYAMLHeaderValue)
}

// Charset adds one or more client accepted charsets.
// Returns itself.
func (n *NegotiationAcceptBuilder) Charset(charset ...string) *NegotiationAcceptBuilder {
	n.lastCharset = charset
	n.charset = append(n.charset, charset...)

	return n
}

// Encoding adds one or more client accepted encoding algorithms.
// Returns itself.
func (n *NegotiationAcceptBuilder) Encoding(encoding ...string) *NegotiationAcceptBuilder {
	n.lastEncoding = encoding
	n.encoding = append(n.encoding, encoding...)

	return n
}

// EncodingGzip adds the "gzip" as accepted encoding.
// Returns itself.
func (n *NegotiationAcceptBuilder) EncodingGzip() *NegotiationAcceptBuilder {
	return n.Encoding(GzipHeaderValue)
}

//  +------------------------------------------------------------+
//  | Serve files                                                |
//  +------------------------------------------------------------+

// ServeContent serves content, headers are autoset
// receives three parameters, it's low-level function, instead you can use .ServeFile(string,bool)/SendFile(string,string)
//
// You can define your own "Content-Type" header also, after this function call
// Doesn't implements resuming (by range), use ctx.SendFile instead
func (ctx *context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error {
	if modified, err := ctx.CheckIfModifiedSince(modtime); !modified && err == nil {
		ctx.WriteNotModified()
		return nil
	}

	if ctx.GetContentType() == "" {
		ctx.ContentType(filename)
	}

	ctx.SetLastModified(modtime)
	var out io.Writer
	if gzipCompression && ctx.ClientSupportsGzip() {
		AddGzipHeaders(ctx.writer)

		gzipWriter := acquireGzipWriter(ctx.writer)
		defer releaseGzipWriter(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.writer
	}
	_, err := io.Copy(out, content)
	return err ///TODO: add an int64 as return value for the content length written like other writers or let it as it's in order to keep the stable api?
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
		return fmt.Errorf("%d", http.StatusNotFound)
	}
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() {
		return ctx.ServeFile(path.Join(filename, "index.html"), gzipCompression)
	}

	return ctx.ServeContent(f, fi.Name(), fi.ModTime(), gzipCompression)
}

// SendFile sends file for force-download to the client
//
// Use this instead of ServeFile to 'force-download' bigger files to the client.
func (ctx *context) SendFile(filename string, destinationName string) error {
	ctx.writer.Header().Set(ContentDispositionHeaderKey, "attachment;filename="+destinationName)
	return ctx.ServeFile(filename, false)
}

//  +------------------------------------------------------------+
//  | Cookies                                                    |
//  +------------------------------------------------------------+

// CookieOption is the type of function that is accepted on
// context's methods like `SetCookieKV`, `RemoveCookie` and `SetCookie`
// as their (last) variadic input argument to amend the end cookie's form.
//
// Any custom or builtin `CookieOption` is valid,
// see `CookiePath`, `CookieCleanPath`, `CookieExpires` and `CookieHTTPOnly` for more.
type CookieOption func(*http.Cookie)

// CookiePath is a `CookieOption`.
// Use it to change the cookie's Path field.
func CookiePath(path string) CookieOption {
	return func(c *http.Cookie) {
		c.Path = path
	}
}

// CookieCleanPath is a `CookieOption`.
// Use it to clear the cookie's Path field, exactly the same as `CookiePath("")`.
func CookieCleanPath(c *http.Cookie) {
	c.Path = ""
}

// CookieExpires is a `CookieOption`.
// Use it to change the cookie's Expires and MaxAge fields by passing the lifetime of the cookie.
func CookieExpires(durFromNow time.Duration) CookieOption {
	return func(c *http.Cookie) {
		c.Expires = time.Now().Add(durFromNow)
		c.MaxAge = int(durFromNow.Seconds())
	}
}

// CookieHTTPOnly is a `CookieOption`.
// Use it to set the cookie's HttpOnly field to false or true.
// HttpOnly field defaults to true for `RemoveCookie` and `SetCookieKV`.
func CookieHTTPOnly(httpOnly bool) CookieOption {
	return func(c *http.Cookie) {
		c.HttpOnly = httpOnly
	}
}

type (
	// CookieEncoder should encode the cookie value.
	// Should accept the cookie's name as its first argument
	// and as second argument the cookie value ptr.
	// Should return an encoded value or an empty one if encode operation failed.
	// Should return an error if encode operation failed.
	//
	// Note: Errors are not printed, so you have to know what you're doing,
	// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	//
	// See `CookieDecoder` too.
	CookieEncoder func(cookieName string, value interface{}) (string, error)
	// CookieDecoder should decode the cookie value.
	// Should accept the cookie's name as its first argument,
	// as second argument the encoded cookie value and as third argument the decoded value ptr.
	// Should return a decoded value or an empty one if decode operation failed.
	// Should return an error if decode operation failed.
	//
	// Note: Errors are not printed, so you have to know what you're doing,
	// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	//
	// See `CookieEncoder` too.
	CookieDecoder func(cookieName string, cookieValue string, v interface{}) error
)

// CookieEncode is a `CookieOption`.
// Provides encoding functionality when adding a cookie.
// Accepts a `CookieEncoder` and sets the cookie's value to the encoded value.
// Users of that is the `SetCookie` and `SetCookieKV`.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/securecookie
func CookieEncode(encode CookieEncoder) CookieOption {
	return func(c *http.Cookie) {
		newVal, err := encode(c.Name, c.Value)
		if err != nil {
			c.Value = ""
		} else {
			c.Value = newVal
		}
	}
}

// CookieDecode is a `CookieOption`.
// Provides decoding functionality when retrieving a cookie.
// Accepts a `CookieDecoder` and sets the cookie's value to the decoded value before return by the `GetCookie`.
// User of that is the `GetCookie`.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/securecookie
func CookieDecode(decode CookieDecoder) CookieOption {
	return func(c *http.Cookie) {
		if err := decode(c.Name, c.Value, &c.Value); err != nil {
			c.Value = ""
		}
	}
}

// SetCookie adds a cookie.
// Use of the "options" is not required, they can be used to amend the "cookie".
//
// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
func (ctx *context) SetCookie(cookie *http.Cookie, options ...CookieOption) {
	for _, opt := range options {
		opt(cookie)
	}

	http.SetCookie(ctx.writer, cookie)
}

// SetCookieKV adds a cookie, requires the name(string) and the value(string).
//
// By default it expires at 2 hours and it's added to the root path,
// use the `CookieExpires` and `CookiePath` to modify them.
// Alternatively: ctx.SetCookie(&http.Cookie{...})
//
// If you want to set custom the path:
// ctx.SetCookieKV(name, value, iris.CookiePath("/custom/path/cookie/will/be/stored"))
//
// If you want to be visible only to current request path:
// (note that client should be responsible for that if server sent an empty cookie's path, all browsers are compatible)
// ctx.SetCookieKV(name, value, iris.CookieCleanPath/iris.CookiePath(""))
// More:
//                              iris.CookieExpires(time.Duration)
//                              iris.CookieHTTPOnly(false)
//
// Examples: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
func (ctx *context) SetCookieKV(name, value string, options ...CookieOption) {
	c := &http.Cookie{}
	c.Path = "/"
	c.Name = name
	c.Value = url.QueryEscape(value)
	c.HttpOnly = true
	c.Expires = time.Now().Add(SetCookieKVExpiration)
	c.MaxAge = int(SetCookieKVExpiration.Seconds())
	ctx.SetCookie(c, options...)
}

// GetCookie returns cookie's value by its name
// returns empty string if nothing was found.
//
// If you want more than the value then:
// cookie, err := ctx.Request().Cookie("name")
//
// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
func (ctx *context) GetCookie(name string, options ...CookieOption) string {
	cookie, err := ctx.request.Cookie(name)
	if err != nil {
		return ""
	}

	for _, opt := range options {
		opt(cookie)
	}

	value, _ := url.QueryUnescape(cookie.Value)
	return value
}

// SetCookieKVExpiration is 365 days by-default
// you can change it or simple, use the SetCookie for more control.
//
// See `SetCookieKVExpiration` and `CookieExpires` for more.
var SetCookieKVExpiration = time.Duration(8760) * time.Hour

// RemoveCookie deletes a cookie by its name and path = "/".
// Tip: change the cookie's path to the current one by: RemoveCookie("name", iris.CookieCleanPath)
//
// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
func (ctx *context) RemoveCookie(name string, options ...CookieOption) {
	c := &http.Cookie{}
	c.Name = name
	c.Value = ""
	c.Path = "/" // if user wants to change it, use of the CookieOption `CookiePath` is required if not `ctx.SetCookie`.
	c.HttpOnly = true
	// RFC says 1 second, but let's do it 1  to make sure is working
	exp := time.Now().Add(-time.Duration(1) * time.Minute)
	c.Expires = exp
	c.MaxAge = -1
	ctx.SetCookie(c, options...)
	// delete request's cookie also, which is temporary available.
	ctx.request.Header.Set("Cookie", "")
}

// VisitAllCookies takes a visitor function which is called
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
	header := ctx.GetHeader(CacheControlHeaderKey)
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
	// NOTE:
	// two return values in order to minimize the if statement:
	// if (Recording) then writer = Recorder()
	// instead we do: recorder,ok = Recording()
	rr, ok := ctx.writer.(*ResponseRecorder)
	return rr, ok
}

// ErrPanicRecovery may be returned from `Context` actions of a `Handler`
// which recovers from a manual panic.
// var ErrPanicRecovery = errors.New("recovery from panic")

// ErrTransactionInterrupt can be used to manually force-complete a Context's transaction
// and log(warn) the wrapped error's message.
// Usage: `... return fmt.Errorf("my custom error message: %w", context.ErrTransactionInterrupt)`.
var ErrTransactionInterrupt = errors.New("transaction interrupted")

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
// Transactions have their own middleware ecosystem also.
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
			ctx.Application().Logger().Warn(fmt.Errorf("recovery from panic: %w", ErrTransactionInterrupt))
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

// Exec calls the framewrok's ServeHTTPC
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
	if path == "" {
		return
	}

	if method == "" {
		method = "GET"
	}

	// backup the handlers
	backupHandlers := ctx.handlers[0:]
	backupPos := ctx.currentHandlerIndex

	req := ctx.request
	// backup the request path information
	backupPath := req.URL.Path
	backupMethod := req.Method
	// don't backupValues := ctx.Values().ReadOnly()
	// set the request to be align with the 'againstRequestPath'
	req.RequestURI = path
	req.URL.Path = path
	req.Method = method

	// [values stays]
	// reset handlers
	ctx.handlers = ctx.handlers[0:0]
	ctx.currentHandlerIndex = 0

	// execute the route from the (internal) context router
	// this way we keep the sessions and the values
	ctx.Application().ServeHTTPC(ctx)

	// set the request back to its previous state
	req.RequestURI = backupPath
	req.URL.Path = backupPath
	req.Method = backupMethod

	// set back the old handlers and the last known index
	ctx.handlers = backupHandlers
	ctx.currentHandlerIndex = backupPos
}

// RouteExists reports whether a particular route exists
// It will search from the current subdomain of context's host, if not inside the root domain.
func (ctx *context) RouteExists(method, path string) bool {
	return ctx.Application().RouteExists(ctx, method, path)
}

const (
	reflectValueContextKey = "_iris_context_reflect_value"
)

// ReflectValue caches and returns a []reflect.Value{reflect.ValueOf(ctx)}.
// It's just a helper to maintain variable inside the context itself.
func (ctx *context) ReflectValue() []reflect.Value {
	if v := ctx.Values().Get(reflectValueContextKey); v != nil {
		return v.([]reflect.Value)
	}

	v := []reflect.Value{reflect.ValueOf(ctx)}
	ctx.Values().Set(reflectValueContextKey, v)
	return v
}

// Application returns the iris app instance which belongs to this context.
// Worth to notice that this function returns an interface
// of the Application, which contains methods that are safe
// to be executed at serve-time. The full app's fields
// and methods are not available here for the developer's safety.
func (ctx *context) Application() Application {
	return ctx.app
}

var lastCapturedContextID uint64

// LastCapturedContextID returns the total number of `context#String` calls.
func LastCapturedContextID() uint64 {
	return atomic.LoadUint64(&lastCapturedContextID)
}

// String returns the string representation of this request.
// Each context has a unique string representation.
// It can be used for simple debugging scenarios, i.e print context as string.
//
// What it returns? A number which declares the length of the
// total `String` calls per executable application, followed
// by the remote IP (the client) and finally the method:url.
func (ctx *context) String() string {
	if ctx.id == 0 {
		// set the id here.
		forward := atomic.AddUint64(&lastCapturedContextID, 1)
		ctx.id = forward
	}

	return fmt.Sprintf("[%d] %s ▶ %s:%s",
		ctx.id, ctx.RemoteAddr(), ctx.Method(), ctx.Request().RequestURI)
}
