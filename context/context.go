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
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/memstore"

	"github.com/Shopify/goreferrer"
	"github.com/fatih/structs"
	"github.com/iris-contrib/blackfriday"
	formbinder "github.com/iris-contrib/formBinder"
	"github.com/json-iterator/go"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/yaml.v2"
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
	// IsWWW returns true if the current subdomain (if any) is www.
	IsWWW() bool
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
	// GetReferrer extracts and returns the information from the "Referer" header as specified
	// in https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
	// or by the URL query parameter "referer".
	GetReferrer() Referrer
	//  +------------------------------------------------------------+
	//  | Headers helpers                                            |
	//  +------------------------------------------------------------+

	// Header adds a header to the response writer.
	Header(name string, value string)

	// ContentType sets the response writer's header key "Content-Type" to the 'cType'.
	ContentType(cType string)
	// GetContentType returns the response writer's header value of "Content-Type"
	// which may, setted before with the 'ContentType'.
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
	// URLParamTrim returns the escaped url query parameter from a request.
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
	// ReadForm binds the formObject  with the form data
	// it supports any kind of type, including custom structs.
	// It will return nothing if request data are empty.
	//
	// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-form/main.go
	ReadForm(formObjectPtr interface{}) error

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
	// (e.g with WriteWithExpiration or StaticEmbedded or Favicon etc.)
	// is a valid one and it's before the "modtime".
	//
	// A check for !modtime && err == nil is necessary to make sure that
	// it's not modified since, because it may return false but without even
	// had the chance to check the client-side (request) header due to some errors,
	// like the HTTP Method is not "GET" or "HEAD" or if the "modtime" is zero
	// or if parsing time from the header failed.
	//
	// It's mostly used internally, e.g. `context#WriteWithExpiration`.
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
	Text(text string) (int, error)
	// HTML writes out a string as text/html.
	HTML(htmlContents string) (int, error)
	// JSON marshals the given interface object and writes the JSON response.
	JSON(v interface{}, options ...JSON) (int, error)
	// JSONP marshals the given interface object and writes the JSON response.
	JSONP(v interface{}, options ...JSONP) (int, error)
	// XML marshals the given interface object and writes the XML response.
	XML(v interface{}, options ...XML) (int, error)
	// Markdown parses the markdown to html and renders its result to the client.
	Markdown(markdownB []byte, options ...Markdown) (int, error)
	// YAML parses the "v" using the yaml parser and renders its result to the client.
	YAML(v interface{}) (int, error)
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
	// use ctx.SendFile or router's `StaticWeb` instead.
	ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error
	// ServeFile serves a file (to send a file, a zip for example to the client you should use the `SendFile` instead)
	// receives two parameters
	// filename/path (string)
	// gzipCompression (bool)
	//
	// You can define your own "Content-Type" with `context#ContentType`, before this function call.
	//
	// This function doesn't support resuming (by range),
	// use ctx.SendFile or router's `StaticWeb` instead.
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
	// GetCookie returns cookie's value by it's name
	// returns empty string if nothing was found.
	//
	// If you want more than the value then:
	// cookie, err := ctx.Request().Cookie("name")
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
	GetCookie(name string, options ...CookieOption) string
	// RemoveCookie deletes a cookie by it's name and path = "/".
	// Tip: change the cookie's path to the current one by: RemoveCookie("name", iris.CookieCleanPath)
	//
	// Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
	RemoveCookie(name string, options ...CookieOption)
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

// Map is just a shortcut of the map[string]interface{}.
type Map map[string]interface{}

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

// Host returns the host part of the current URI.
func (ctx *context) Host() string {
	return GetHost(ctx.request)
}

// GetHost returns the host part of the current URI.
func GetHost(r *http.Request) string {
	h := r.URL.Host
	if h == "" {
		h = r.Host
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

var isMobileRegex = regexp.MustCompile(`(?i)(android|avantgo|blackberry|bolt|boost|cricket|docomo|fone|hiptop|mini|mobi|palm|phone|pie|tablet|up\.browser|up\.link|webos|wos)`)

// IsMobile checks if client is using a mobile device(phone or tablet) to communicate with this server.
// If the return value is true that means that the http client using a mobile
// device to communicate with the server, otherwise false.
//
// Keep note that this checks the "User-Agent" request header.
func (ctx *context) IsMobile() bool {
	s := ctx.GetHeader("User-Agent")
	return isMobileRegex.MatchString(s)
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
	ReferrerType int

	// ReferrerGoogleSearchType is the goreferrer enum for a google search type (organic, adwords).
	ReferrerGoogleSearchType int
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

func (gs ReferrerGoogleSearchType) String() string {
	return goreferrer.GoogleSearchType(gs).String()
}

func (r ReferrerType) String() string {
	return goreferrer.ReferrerType(r).String()
}

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
		if cType != ContentBinaryHeaderValue {
			cType += "; charset=" + ctx.Application().ConfigurationReadOnly().GetCharset()
		}
	}

	ctx.writer.Header().Set(ContentTypeHeaderKey, cType)
}

// GetContentType returns the response writer's header value of "Content-Type"
// which may, setted before with the 'ContentType'.
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

// URLParamTrim returns the escaped url query parameter from a request.
func (ctx *context) URLParamEscape(name string) string {
	return DecodeQuery(ctx.URLParam(name))
}

var errURLParamNotFound = errors.New("url param '%s' does not exist")

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

	return -1, errURLParamNotFound.Format(name)
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

	return -1, errURLParamNotFound.Format(name)
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

	return -1, errURLParamNotFound.Format(name)
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
	/*
		net/http/request.go#1219
		for k, v := range f.Value {
			r.Form[k] = append(r.Form[k], v...)
			// r.PostForm should also be populated. See Issue 9305.
			r.PostForm[k] = append(r.PostForm[k], v...)
		}
	*/

	// ParseMultipartForm calls `request.ParseForm` automatically
	// therefore we don't need to call it here, although it doesn't hurt.
	// After one call to ParseMultipartForm or ParseForm,
	// subsequent calls have no effect, are idempotent.
	ctx.request.ParseMultipartForm(ctx.Application().ConfigurationReadOnly().GetPostMaxMemory())

	if form := ctx.request.Form; len(form) > 0 {
		return form, true
	}

	if form := ctx.request.PostForm; len(form) > 0 {
		return form, true
	}

	if m := ctx.request.MultipartForm; m != nil {
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

var errUnableToFindPostValue = errors.New("unable to find post value '%s'")

// PostValueInt returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as int.
//
// If not found returns -1 and a non-nil error.
func (ctx *context) PostValueInt(name string) (int, error) {
	v := ctx.PostValue(name)
	if v == "" {
		return -1, errUnableToFindPostValue.Format(name)
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
		return -1, errUnableToFindPostValue.Format(name)
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
		return -1, errUnableToFindPostValue.Format(name)
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
		return false, errUnableToFindPostValue.Format(name)
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
func (ctx *context) ReadJSON(jsonObject interface{}) error {
	var unmarshaler = json.Unmarshal
	if ctx.shouldOptimize() {
		unmarshaler = jsoniter.Unmarshal
	}
	return ctx.UnmarshalBody(jsonObject, UnmarshalerFunc(unmarshaler))
}

// ReadXML reads XML from request's body and binds it to a value of any xml-valid type.
//
// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-xml/main.go
func (ctx *context) ReadXML(xmlObject interface{}) error {
	return ctx.UnmarshalBody(xmlObject, UnmarshalerFunc(xml.Unmarshal))
}

// IsErrPath can be used at `context#ReadForm`.
// It reports whether the incoming error is type of `formbinder.ErrPath`,
// which can be ignored when server allows unknown post values to be sent by the client.
//
// A shortcut for the `formbinder#IsErrPath`.
var IsErrPath = formbinder.IsErrPath

// ReadForm binds the formObject  with the form data
// it supports any kind of type, including custom structs.
// It will return nothing if request data are empty.
//
// Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-form/main.go
func (ctx *context) ReadForm(formObject interface{}) error {
	values := ctx.FormValues()
	if len(values) == 0 {
		return nil
	}

	// or dec := formbinder.NewDecoder(&formbinder.DecoderOptions{TagName: "form"})
	// somewhere at the app level. I did change the tagName to "form"
	// inside its source code, so it's not needed for now.
	return formbinder.Decode(values, formObject)
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

// CheckIfModifiedSince checks if the response is modified since the "modtime".
// Note that it has nothing to do with server-side caching.
// It does those checks by checking if the "If-Modified-Since" request header
// sent by client or a previous server response header
// (e.g with WriteWithExpiration or StaticEmbedded or Favicon etc.)
// is a valid one and it's before the "modtime".
//
// A check for !modtime && err == nil is necessary to make sure that
// it's not modified since, because it may return false but without even
// had the chance to check the client-side (request) header due to some errors,
// like the HTTP Method is not "GET" or "HEAD" or if the "modtime" is zero
// or if parsing time from the header failed.
//
// It's mostly used internally, e.g. `context#WriteWithExpiration`.
func (ctx *context) CheckIfModifiedSince(modtime time.Time) (bool, error) {
	if method := ctx.Method(); method != http.MethodGet && method != http.MethodHead {
		return false, errors.New("skip: method")
	}
	ims := ctx.GetHeader(IfModifiedSinceHeaderKey)
	if ims == "" || IsZeroTime(modtime) {
		return false, errors.New("skip: zero time")
	}
	t, err := ParseTime(ctx, ims)
	if err != nil {
		return false, errors.New("skip: " + err.Error())
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

// WriteWithExpiration like Write but it sends with an expiration datetime
// which is refreshed every package-level `StaticCacheDuration` field.
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
	if h := ctx.GetHeader(AcceptEncodingHeaderKey); h != "" {
		for _, v := range strings.Split(h, ";") {
			if strings.Contains(v, GzipHeaderValue) { // we do Contains because sometimes browsers has the q=, we don't use it atm. || strings.Contains(v,"deflate"){
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
// You may re-use this function in the same handler
// to write more data many times without any troubles.
func (ctx *context) WriteGzip(b []byte) (int, error) {
	if !ctx.ClientSupportsGzip() {
		return 0, errClientDoesNotSupportGzip
	}

	return ctx.GzipResponseWriter().Write(b)
}

// TryWriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
// If client does not supprots gzip then the contents are written as they are, uncompressed.
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

	err := ctx.Application().View(ctx.writer, filename, layout, bindingData)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.StopExecution()
	}

	return err
}

const (
	// ContentBinaryHeaderValue header value for binary data.
	ContentBinaryHeaderValue = "application/octet-stream"
	// ContentHTMLHeaderValue is the  string of text/html response header's content type value.
	ContentHTMLHeaderValue = "text/html"
	// ContentJSONHeaderValue header value for JSON data.
	ContentJSONHeaderValue = "application/json"
	// ContentJavascriptHeaderValue header value for JSONP & Javascript data.
	ContentJavascriptHeaderValue = "application/javascript"
	// ContentTextHeaderValue header value for Text data.
	ContentTextHeaderValue = "text/plain"
	// ContentXMLHeaderValue header value for XML data.
	ContentXMLHeaderValue = "text/xml"
	// ContentMarkdownHeaderValue custom key/content type, the real is the text/html.
	ContentMarkdownHeaderValue = "text/markdown"
	// ContentYAMLHeaderValue header value for YAML data.
	ContentYAMLHeaderValue = "application/x-yaml"
)

// Binary writes out the raw bytes as binary data.
func (ctx *context) Binary(data []byte) (int, error) {
	ctx.ContentType(ContentBinaryHeaderValue)
	return ctx.Write(data)
}

// Text writes out a string as plain text.
func (ctx *context) Text(text string) (int, error) {
	ctx.ContentType(ContentTextHeaderValue)
	return ctx.writer.WriteString(text)
}

// HTML writes out a string as text/html.
func (ctx *context) HTML(htmlContents string) (int, error) {
	ctx.ContentType(ContentHTMLHeaderValue)
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

	n, err = WriteJSON(ctx.writer, v, options, ctx.shouldOptimize())
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

// DefaultXMLOptions is the optional settings that are being used
// from `ctx.XML`.
var DefaultXMLOptions = XML{}

// XML marshals the given interface object and writes the XML response to the client.
func (ctx *context) XML(v interface{}, opts ...XML) (int, error) {
	options := DefaultXMLOptions

	if len(opts) > 0 {
		options = opts[0]
	}

	ctx.ContentType(ContentXMLHeaderValue)

	n, err := WriteXML(ctx.writer, v, options)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
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
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	return n, err
}

// YAML marshals the "v" using the yaml marshaler and renders its result to the client.
func (ctx *context) YAML(v interface{}) (int, error) {
	out, err := yaml.Marshal(v)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return 0, err
	}

	ctx.ContentType(ContentYAMLHeaderValue)
	return ctx.Write(out)
}

//  +------------------------------------------------------------+
//  | Serve files                                                |
//  +------------------------------------------------------------+

var (
	errServeContent = errors.New("while trying to serve content to the client. Trace %s")
)

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

	ctx.ContentType(filename)
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
	return errServeContent.With(err) ///TODO: add an int64 as return value for the content length written like other writers or let it as it's in order to keep the stable api?
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
// Any custom or built'n `CookieOption` is valid,
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
	// Should accept as first argument the cookie name
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
	// Should accept as first argument the cookie name,
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

// GetCookie returns cookie's value by it's name
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

// SetCookieKVExpiration is 2 hours by-default
// you can change it or simple, use the SetCookie for more control.
//
// See `SetCookieKVExpiration` and `CookieExpires` for more.
var SetCookieKVExpiration = time.Duration(120) * time.Minute

// RemoveCookie deletes a cookie by it's name and path = "/".
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
