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
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fatih/structs"
	formbinder "github.com/iris-contrib/formBinder"
	"github.com/json-iterator/go"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/russross/blackfriday.v2"
	"gopkg.in/yaml.v2"

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

// RequestParams is a key string - value string storage which
// context's request dynamic path params are being kept.
// Empty if the route is static.
type RequestParams struct {
	store memstore.Store
}

// Set adds a key-value pair to the path parameters values
// it's being called internally so it shouldn't be used as a local storage by the user, use `ctx.Values()` instead.
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

var emptyEntry memstore.Entry

// GetEntryAt returns the internal Entry of the memstore based on its index,
// the stored index by the router.
// If not found then it returns a zero Entry and false.
func (r RequestParams) GetEntryAt(index int) (memstore.Entry, bool) {
	if len(r.store) > index {
		return r.store[index], true
	}
	return emptyEntry, false
}

// GetEntry returns the internal Entry of the memstore based on its "key".
// If not found then it returns a zero Entry and false.
func (r RequestParams) GetEntry(key string) (memstore.Entry, bool) {
	// we don't return the pointer here, we don't want to give the end-developer
	// the strength to change the entry that way.
	if e := r.store.GetEntry(key); e != nil {
		return *e, true
	}
	return emptyEntry, false
}

// Get returns a path parameter's value based on its route's dynamic path key.
func (r RequestParams) Get(key string) string {
	return r.store.GetString(key)
}

// GetTrim returns a path parameter's value without trailing spaces based on its route's dynamic path key.
func (r RequestParams) GetTrim(key string) string {
	return strings.TrimSpace(r.Get(key))
}

// GetEscape returns a path parameter's double-url-query-escaped value based on its route's dynamic path key.
func (r RequestParams) GetEscape(key string) string {
	return DecodeQuery(DecodeQuery(r.Get(key)))
}

// GetDecoded returns a path parameter's double-url-query-escaped value based on its route's dynamic path key.
// same as `GetEscape`.
func (r RequestParams) GetDecoded(key string) string {
	return r.GetEscape(key)
}

// GetInt returns the path parameter's value as int, based on its key.
func (r RequestParams) GetInt(key string) (int, error) {
	return r.store.GetInt(key)
}

// GetInt64 returns the path paramete's value as int64, based on its key.
func (r RequestParams) GetInt64(key string) (int64, error) {
	return r.store.GetInt64(key)
}

// GetFloat64 returns a path parameter's value based as float64 on its route's dynamic path key.
func (r RequestParams) GetFloat64(key string) (float64, error) {
	return r.store.GetFloat64(key)
}

// GetBool returns the path parameter's value as bool, based on its key.
// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
// or "0" or "f" or "F" or "FALSE" or "false" or "False".
// Any other value returns an error.
func (r RequestParams) GetBool(key string) (bool, error) {
	return r.store.GetBool(key)
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
	// current handler index without change the current index.rns that index, useless return value.
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
	// NextHandler returns(but it is NOT executes) the next handler from the handlers chain.
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
	// URLParamDefault returns the get parameter from a request, if not found then "def" is returned.
	URLParamDefault(name string, def string) string
	// URLParam returns the get parameter from a request , if any.
	URLParam(name string) string
	// URLParamTrim returns the url query parameter with trailing white spaces removed from a request,
	// returns an error if parse failed.
	URLParamTrim(name string) string
	// URLParamTrim returns the escaped url query parameter from a request,
	// returns an error if parse failed.
	URLParamEscape(name string) string
	// URLParamIntDefault returns the url query parameter as int value from a request,
	// if not found then "def" is returned.
	// Returns an error if parse failed.
	URLParamIntDefault(name string, def int) (int, error)
	// URLParamInt returns the url query parameter as int value from a request,
	// returns an error if parse failed.
	URLParamInt(name string) (int, error)
	// URLParamInt64Default returns the url query parameter as int64 value from a request,
	// if not found then "def" is returned.
	// Returns an error if parse failed.
	URLParamInt64Default(name string, def int64) (int64, error)
	// URLParamInt64 returns the url query parameter as int64 value from a request,
	// returns an error if parse failed.
	URLParamInt64(name string) (int64, error)
	// URLParamFloat64Default returns the url query parameter as float64 value from a request,
	// if not found then "def" is returned.
	// Returns an error if parse failed.
	URLParamFloat64Default(name string, def float64) (float64, error)
	// URLParamFloat64 returns the url query parameter as float64 value from a request,
	// returns an error if parse failed.
	URLParamFloat64(name string) (float64, error)
	// URLParamBool returns the url query parameter as boolean value from a request,
	// returns an error if parse failed.
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
	// PostValueIntDefault returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as int.
	//
	// If not found returns the "def".
	PostValueIntDefault(name string, def int) (int, error)
	// PostValueInt returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as int.
	//
	// If not found returns 0.
	PostValueInt(name string) (int, error)
	// PostValueInt64Default returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as int64.
	//
	// If not found returns the "def".
	PostValueInt64Default(name string, def int64) (int64, error)
	// PostValueInt64 returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as float64.
	//
	// If not found returns 0.0.
	PostValueInt64(name string) (int64, error)
	// PostValueInt64Default returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as float64.
	//
	// If not found returns the "def".
	PostValueFloat64Default(name string, def float64) (float64, error)
	/// PostValueInt64Default returns the parsed form data from POST, PATCH,
	// or PUT body parameters based on a "name", as float64.
	//
	// If not found returns 0.0.
	PostValueFloat64(name string) (float64, error)
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

// Cache304 sends a `StatusNotModified` (304) whenever
// the "If-Modified-Since" request header (time) is before the
// time.Now() + expiresEvery (always compared to their UTC values).
// Use this `context#Cache304` instead of the "github.com/kataras/iris/cache" or iris.Cache
// for better performance.
// Clients that are compatible with the http RCF (all browsers are and tools like postman)
// will handle the caching.
// The only disadvantage of using that instead of server-side caching
// is that this method will send a 304 status code instead of 200,
// So, if you use it side by side with other micro services
// you have to check for that status code as well for a valid response.
//
// Developers are free to extend this method's behavior
// by watching system directories changes manually and use of the `ctx.WriteWithExpiration`
// with a "modtime" based on the file modified date,
// simillary to the `StaticWeb`(StaticWeb sends an OK(200) and browser disk caching instead of 304).
var Cache304 = func(expiresEvery time.Duration) Handler {
	return func(ctx Context) {
		now := time.Now()
		if modified, err := ctx.CheckIfModifiedSince(now.Add(-expiresEvery)); !modified && err == nil {
			ctx.WriteNotModified()
			return
		}

		ctx.SetLastModified(now)
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
	ctx.params.store = ctx.params.store[0:0]
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
var Next = DefaultNext ///TODO: add an example for this usecase, i.e describe handlers and skip only file handlers.

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
		if cType != ContentBinaryHeaderValue {
			cType += "; charset=" + ctx.Application().ConfigurationReadOnly().GetCharset()
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
	v := ctx.request.URL.Query().Get(name)
	if v == "" {
		return def
	}
	return v
}

// URLParam returns the get parameter from a request , if any.
func (ctx *context) URLParam(name string) string {
	return ctx.URLParamDefault(name, "")
}

// URLParamTrim returns the url query parameter with trailing white spaces removed from a request,
// returns an error if parse failed.
func (ctx *context) URLParamTrim(name string) string {
	return strings.TrimSpace(ctx.URLParam(name))
}

// URLParamTrim returns the escaped url query parameter from a request,
// returns an error if parse failed.
func (ctx *context) URLParamEscape(name string) string {
	return DecodeQuery(ctx.URLParam(name))
}

// URLParamIntDefault returns the url query parameter as int value from a request,
// if not found then "def" is returned.
// Returns an error if parse failed.
func (ctx *context) URLParamIntDefault(name string, def int) (int, error) {
	v := ctx.URLParam(name)
	if v == "" {
		return def, nil
	}
	return strconv.Atoi(v)
}

// URLParamInt returns the url query parameter as int value from a request,
// returns an error if parse failed.
func (ctx *context) URLParamInt(name string) (int, error) {
	return ctx.URLParamIntDefault(name, 0)
}

// URLParamInt64Default returns the url query parameter as int64 value from a request,
// if not found then "def" is returned.
// Returns an error if parse failed.
func (ctx *context) URLParamInt64Default(name string, def int64) (int64, error) {
	v := ctx.URLParam(name)
	if v == "" {
		return def, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

// URLParamInt64 returns the url query parameter as int64 value from a request,
// returns an error if parse failed.
func (ctx *context) URLParamInt64(name string) (int64, error) {
	return ctx.URLParamInt64Default(name, 0.0)
}

// URLParamFloat64Default returns the url query parameter as float64 value from a request,
// if not found then "def" is returned.
// Returns an error if parse failed.
func (ctx *context) URLParamFloat64Default(name string, def float64) (float64, error) {
	v := ctx.URLParam(name)
	if v == "" {
		return def, nil
	}
	return strconv.ParseFloat(v, 64)
}

// URLParamFloat64 returns the url query parameter as float64 value from a request,
// returns an error if parse failed.
func (ctx *context) URLParamFloat64(name string) (float64, error) {
	return ctx.URLParamFloat64Default(name, 0.0)
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

// PostValueIntDefault returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as int.
//
// If not found returns the "def".
func (ctx *context) PostValueIntDefault(name string, def int) (int, error) {
	v := ctx.PostValue(name)
	if v == "" {
		return def, nil
	}
	return strconv.Atoi(v)
}

// PostValueInt returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as int.
//
// If not found returns 0.
func (ctx *context) PostValueInt(name string) (int, error) {
	return ctx.PostValueIntDefault(name, 0)
}

// PostValueInt64Default returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as int64.
//
// If not found returns the "def".
func (ctx *context) PostValueInt64Default(name string, def int64) (int64, error) {
	v := ctx.PostValue(name)
	if v == "" {
		return def, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

// PostValueInt64 returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as float64.
//
// If not found returns 0.0.
func (ctx *context) PostValueInt64(name string) (int64, error) {
	return ctx.PostValueInt64Default(name, 0.0)
}

// PostValueInt64Default returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as float64.
//
// If not found returns the "def".
func (ctx *context) PostValueFloat64Default(name string, def float64) (float64, error) {
	v := ctx.PostValue(name)
	if v == "" {
		return def, nil
	}
	return strconv.ParseFloat(v, 64)
}

// PostValueInt64Default returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as float64.
//
// If not found returns 0.0.
func (ctx *context) PostValueFloat64(name string) (float64, error) {
	return ctx.PostValueFloat64Default(name, 0.0)
}

// PostValueInt64Default returns the parsed form data from POST, PATCH,
// or PUT body parameters based on a "name", as bool.
//
// If not found or value is false, then it returns false, otherwise true.
func (ctx *context) PostValueBool(name string) (bool, error) {
	return strconv.ParseBool(ctx.PostValue(name))
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
func (ctx *context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	// we don't have access to see if the request is body stream
	// and then the ParseMultipartForm can be useless
	// here but do it in order to apply the post limit,
	// the internal request.FormFile will not do it if that's filled
	// and it's not a stream body.
	ctx.request.ParseMultipartForm(ctx.Application().ConfigurationReadOnly().GetPostMaxMemory())
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

	// or dec := formbinder.NewDecoder(&formbinder.DecoderOptions{TagName: "form"})
	// somewhere at the app level. I did change the tagName to "form"
	// inside its source code, so it's not needed for now.
	return errReadBody.With(formbinder.Decode(values, formObject))
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
		ctx.Header(lastModifiedHeaderKey, FormatTime(ctx, modtime.UTC())) // or modtime.UTC()?
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
	ims := ctx.GetHeader(ifModifiedSinceHeaderKey)
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
	delete(h, contentTypeHeaderKey)
	delete(h, contentLengthHeaderKey)
	if h.Get("Etag") != "" {
		delete(h, lastModifiedHeaderKey)
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

	optimize := ctx.shouldOptimize()

	ctx.ContentType(ContentJSONHeaderValue)

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
	if modified, err := ctx.CheckIfModifiedSince(modtime); !modified && err == nil {
		ctx.WriteNotModified()
		return nil
	}

	ctx.ContentType(filename)
	ctx.SetLastModified(modtime)
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
	c.Value = url.QueryEscape(value)
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
	value, _ := url.QueryUnescape(cookie.Value)
	return value
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
