package iris

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/iris-contrib/formBinder"
	"github.com/kataras/go-errors"
	"github.com/kataras/go-fs"
	"github.com/kataras/go-sessions"
)

const (
	// ContentType represents the header["Content-Type"]
	contentType = "Content-Type"
	// ContentLength represents the header["Content-Length"]
	contentLength = "Content-Length"
	// contentEncodingHeader represents the header["Content-Encoding"]
	contentEncodingHeader = "Content-Encoding"
	// varyHeader represents the header "Vary"
	varyHeader = "Vary"
	// acceptEncodingHeader represents the header key & value "Accept-Encoding"
	acceptEncodingHeader = "Accept-Encoding"
	// ContentHTML is the  string of text/html response headers
	contentHTML = "text/html"
	// ContentBinary header value for binary data.
	contentBinary = "application/octet-stream"
	// ContentJSON header value for JSON data.
	contentJSON = "application/json"
	// ContentJSONP header value for JSONP & Javascript data.
	contentJSONP = "application/javascript"
	// ContentJavascript header value for Javascript/JSONP
	// conversional
	contentJavascript = "application/javascript"
	// ContentText header value for Text data.
	contentText = "text/plain"
	// ContentXML header value for XML data.
	contentXML = "text/xml"

	// contentMarkdown custom key/content type, the real is the text/html
	contentMarkdown = "text/markdown"

	// LastModified "Last-Modified"
	lastModified = "Last-Modified"
	// IfModifiedSince "If-Modified-Since"
	ifModifiedSince = "If-Modified-Since"
	// ContentDisposition "Content-Disposition"
	contentDisposition = "Content-Disposition"
	// CacheControl "Cache-Control"
	cacheControl = "Cache-Control"

	// stopExecutionPosition used inside the Context, is the number which shows us that the context's middleware manualy stop the execution
	stopExecutionPosition = 255
)

// errors

var (
	errTemplateExecute = errors.New("Unable to execute a template. Trace: %s")
	errFlashNotFound   = errors.New("Unable to get flash message. Trace: Cookie does not exists")
	errReadBody        = errors.New("While trying to read %s from the request body. Trace %s")
	errServeContent    = errors.New("While trying to serve content to the client. Trace %s")
)

type (
	requestValue struct {
		key   []byte
		value interface{}
	}
	requestValues []requestValue
)

func (r *requestValues) Set(key string, value interface{}) {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if string(kv.key) == key {
			kv.value = value
			return
		}
	}

	c := cap(args)
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.key = append(kv.key[:0], key...)
		kv.value = value
		*r = args
		return
	}

	kv := requestValue{}
	kv.key = append(kv.key[:0], key...)
	kv.value = value
	*r = append(args, kv)
}

func (r *requestValues) Get(key string) interface{} {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if string(kv.key) == key {
			return kv.value
		}
	}
	return nil
}

func (r *requestValues) Reset() {
	*r = (*r)[:0]
}

type (
	// Map is just a conversion for a map[string]interface{}
	// should not be used inside Render when PongoEngine is used.
	Map map[string]interface{}

	// Context is resetting every time a request is coming to the server
	// it is not good practice to use this object in goroutines, for these cases use the .Clone()
	Context struct {
		ResponseWriter *ResponseWriter
		Request        *http.Request
		values         requestValues
		framework      *Framework
		//keep track all registed middleware (handlers)
		Middleware Middleware //  exported because is useful for debugging
		session    sessions.Session
		// Pos is the position number of the Context, look .Next to understand
		Pos int // exported because is useful for debugging
	}
)

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------------------------Handler(s) Execution------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// Do calls the first handler only, it's like Next with negative pos, used only on Router&MemoryRouter
func (ctx *Context) Do() {
	ctx.Pos = 0
	ctx.Middleware[0].Serve(ctx)
}

// Next calls all the next handler from the middleware stack, it used inside a middleware
func (ctx *Context) Next() {
	//set position to the next
	ctx.Pos++
	//run the next
	if ctx.Pos < len(ctx.Middleware) {
		ctx.Middleware[ctx.Pos].Serve(ctx)
	}
}

// StopExecution just sets the .pos to 255 in order to  not move to the next middlewares(if any)
func (ctx *Context) StopExecution() {
	ctx.Pos = stopExecutionPosition
}

// IsStopped checks and returns true if the current position of the Context is 255, means that the StopExecution has called
func (ctx *Context) IsStopped() bool {
	return ctx.Pos == stopExecutionPosition
}

// GetHandlerName as requested returns the stack-name of the function which the Middleware is setted from
func (ctx *Context) GetHandlerName() string {
	return runtime.FuncForPC(reflect.ValueOf(ctx.Middleware[len(ctx.Middleware)-1]).Pointer()).Name()
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------------------------Request URL, Method, IP & Headers getters---------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// Method returns the http request method
// same as *http.Request.Method
func (ctx *Context) Method() string {
	return ctx.Request.Method
}

// Host returns the host part of the current url
func (ctx *Context) Host() string {
	return ctx.Request.URL.Host
}

// ServerHost returns the server host taken by *http.Request.Host
func (ctx *Context) ServerHost() string {
	return ctx.Request.Host
}

// Subdomain returns the subdomain (string) of this request, if any
func (ctx *Context) Subdomain() (subdomain string) {
	host := ctx.Host()
	if index := strings.IndexByte(host, '.'); index > 0 {
		subdomain = host[0:index]
	}

	return
}

// VirtualHostname returns the hostname that user registers, host path maybe differs from the real which is HostString, which taken from a net.listener
func (ctx *Context) VirtualHostname() string {
	realhost := ctx.Host()
	hostname := realhost
	virtualhost := ctx.framework.mux.hostname

	if portIdx := strings.IndexByte(hostname, ':'); portIdx > 0 {
		hostname = hostname[0:portIdx]
	}
	if idxDotAnd := strings.LastIndexByte(hostname, '.'); idxDotAnd > 0 {
		s := hostname[idxDotAnd:]
		// means that we have the request's host mymachine.com or 127.0.0.1/0.0.0.0, but for the second option we will need to replace it with the hostname that the dev was registered
		// this needed to parse correct the {{ url }} iris global template engine's function
		if s == ".1" {
			hostname = strings.Replace(hostname, "127.0.0.1", virtualhost, 1)
		} else if s == ".0" {
			hostname = strings.Replace(hostname, "0.0.0.0", virtualhost, 1)
		}
		//
	} else {
		hostname = strings.Replace(hostname, "localhost", virtualhost, 1)
	}

	return hostname
}

// Path returns the full escaped path as string
// for unescaped use: ctx.RequestCtx.RequestURI() or RequestPath(escape bool)
func (ctx *Context) Path() string {
	return ctx.RequestPath(!ctx.framework.Config.DisablePathEscape)
}

// RequestPath returns the requested path
func (ctx *Context) RequestPath(escape bool) string {
	if escape {
		return ctx.Request.URL.EscapedPath()
	}
	return ctx.Request.RequestURI
}

// RemoteAddr tries to return the real client's request IP
func (ctx *Context) RemoteAddr() string {
	header := ctx.RequestHeader("X-Real-Ip")
	realIP := strings.TrimSpace(header)
	if realIP != "" {
		return realIP
	}
	realIP = ctx.RequestHeader("X-Forwarded-For")
	idx := strings.IndexByte(realIP, ',')
	if idx >= 0 {
		realIP = realIP[0:idx]
	}
	realIP = strings.TrimSpace(realIP)
	if realIP != "" {
		return realIP
	}
	addr := strings.TrimSpace(ctx.Request.RemoteAddr)
	if len(addr) == 0 {
		return ""
	}
	// if addr has port use the net.SplitHostPort otherwise(error occurs) take as it is
	if ip, _, err := net.SplitHostPort(addr); err == nil {
		return ip
	}
	return addr
}

// RequestHeader returns the request header's value
// accepts one parameter, the key of the header (string)
// returns string
func (ctx *Context) RequestHeader(k string) string {
	return ctx.Request.Header.Get(k)
}

// IsAjax returns true if this request is an 'ajax request'( XMLHttpRequest)
//
// Read more at: http://www.w3schools.com/ajax/
func (ctx *Context) IsAjax() bool {
	return ctx.RequestHeader("HTTP_X_REQUESTED_WITH") == "XMLHttpRequest"
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------------------------GET & POST arguments------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// URLParam returns the get parameter from a request , if any
func (ctx *Context) URLParam(key string) string {
	return ctx.Request.URL.Query().Get(key)
}

// URLParams returns a map of GET query parameters seperated by comma if more than one
// it returns an empty map if nothing founds
func (ctx *Context) URLParams() map[string]string {
	values := map[string]string{}

	q := ctx.URLParamsAsMulti()
	if q != nil {
		for k, v := range q {
			values[k] = strings.Join(v, ",")
		}
	}

	return values
}

// URLParamsAsMulti returns a map of list contains the url get parameters
func (ctx *Context) URLParamsAsMulti() map[string][]string {
	return ctx.Request.URL.Query()
}

// URLParamInt returns the url query parameter as int value from a request ,  returns error on parse fail
func (ctx *Context) URLParamInt(key string) (int, error) {
	return strconv.Atoi(ctx.URLParam(key))
}

// URLParamInt64 returns the url query parameter as int64 value from a request ,  returns error on parse fail
func (ctx *Context) URLParamInt64(key string) (int64, error) {
	return strconv.ParseInt(ctx.URLParam(key), 10, 64)
}

func (ctx *Context) askParseForm() error {
	if ctx.Request.Form == nil {
		if err := ctx.Request.ParseForm(); err != nil {
			return err
		}
	}
	return nil
}

// FormValues returns all post data values with their keys
// form data, get, post & put query arguments
//
// NOTE: A check for nil is necessary for zero results
func (ctx *Context) FormValues() map[string][]string {
	//  we skip the check of multipart form, takes too much memory, if user wants it can do manually now.
	if err := ctx.askParseForm(); err != nil {
		return nil
	}
	return ctx.Request.Form // nothing more to do, it's already contains both query and post & put args.
}

// FormValue returns a single form value by its name/key
func (ctx *Context) FormValue(name string) string {
	return ctx.Request.FormValue(name)
}

// PostValue returns a form's only-post value by its name
// same as Request.PostFormValue
func (ctx *Context) PostValue(name string) string {
	return ctx.Request.PostFormValue(name)
}

// FormFile returns the first file for the provided form key.
// FormFile calls ctx.Request.ParseMultipartForm and ParseForm if necessary.
//
// same as Request.FormFile
func (ctx *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return ctx.Request.FormFile(key)
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------------------------Request Body Binders/Readers----------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

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
// for ReadJSON is the encoding/json and for ReadXML is the encoding/xml
type BodyDecoder interface {
	Decode(data []byte) error
}

// Unmarshaler is the interface implemented by types that can unmarshal any raw data
// TIP INFO: Any v object which implements the BodyDecoder can be override the unmarshaler
type Unmarshaler interface {
	Unmarshal(data []byte, v interface{}) error
}

// UnmarshalerFunc a shortcut for the Unmarshaler interface
//
// See 'Unmarshaler' and 'BodyDecoder' for more
type UnmarshalerFunc func(data []byte, v interface{}) error

// Unmarshal parses the X-encoded data and stores the result in the value pointed to by v.
// Unmarshal uses the inverse of the encodings that Marshal uses, allocating maps,
// slices, and pointers as necessary.
func (u UnmarshalerFunc) Unmarshal(data []byte, v interface{}) error {
	return u(data, v)
}

// UnmarshalBody reads the request's body and binds it to a value or pointer of any type
// Examples of usage: context.ReadJSON, context.ReadXML
func (ctx *Context) UnmarshalBody(v interface{}, unmarshaler Unmarshaler) error {
	if ctx.Request.Body == nil {
		return errors.New("Empty body, please send request body!")
	}

	rawData, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return err
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

// ReadJSON reads JSON from request's body and binds it to a value of any json-valid type
func (ctx *Context) ReadJSON(jsonObject interface{}) error {
	return ctx.UnmarshalBody(jsonObject, UnmarshalerFunc(json.Unmarshal))
}

// ReadXML reads XML from request's body and binds it to a value of any xml-valid type
func (ctx *Context) ReadXML(xmlObject interface{}) error {
	return ctx.UnmarshalBody(xmlObject, UnmarshalerFunc(xml.Unmarshal))
}

// ReadForm binds the formObject  with the form data
// it supports any kind of struct
func (ctx *Context) ReadForm(formObject interface{}) error {
	values := ctx.FormValues()
	if values == nil {
		return errors.New("An empty form passed on context.ReadForm")
	}
	return errReadBody.With(formBinder.Decode(values, formObject))
}

// ResetBody resets the body of the response
func (ctx *Context) ResetBody() {
	ctx.ResponseWriter.ResetBody()
}

/* Response */

// SetContentType sets the response writer's header key 'Content-Type' to a given value(s)
func (ctx *Context) SetContentType(s string) {
	ctx.ResponseWriter.Header().Set(contentType, s)
}

// SetHeader write to the response writer's header to a given key the given value
func (ctx *Context) SetHeader(k string, v string) {
	ctx.ResponseWriter.Header().Add(k, v)
}

// SetStatusCode sets the status code header to the response
//
// NOTE: Iris takes cares of multiple header writing
func (ctx *Context) SetStatusCode(statusCode int) {
	ctx.ResponseWriter.WriteHeader(statusCode)
}

// it used only inside Redirect,
// keep it here for allocations reason
var httpsSchemeOnlyBytes = []byte("https")

// Redirect redirect sends a redirect response the client
// accepts 2 parameters string and an optional int
// first parameter is the url to redirect
// second parameter is the http status should send, default is 302 (StatusFound),
// you can set it to 301 (Permant redirect), if that's nessecery
func (ctx *Context) Redirect(urlToRedirect string, statusHeader ...int) {
	ctx.StopExecution()

	httpStatus := StatusFound // a 'temporary-redirect-like' which works better than for our purpose
	if statusHeader != nil && len(statusHeader) > 0 && statusHeader[0] > 0 {
		httpStatus = statusHeader[0]
	}

	if urlToRedirect == ctx.Path() {
		if ctx.framework.Config.IsDevelopment {
			ctx.Log("Trying to redirect to itself. FROM: %s TO: %s", ctx.Path(), urlToRedirect)
		}
	}
	http.Redirect(ctx.ResponseWriter.ResponseWriter, ctx.Request, urlToRedirect, httpStatus)
}

// RedirectTo does the same thing as Redirect but instead of receiving a uri or path it receives a route name
func (ctx *Context) RedirectTo(routeName string, args ...interface{}) {
	s := ctx.framework.URL(routeName, args...)
	if s != "" {
		ctx.Redirect(s, StatusFound)
	}
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------------------------(Custom) Errors-----------------------------------------
// ----------------------Look iris.OnError/EmitError for more---------------------------
// -------------------------------------------------------------------------------------

// NotFound emits an error 404 to the client, using the custom http errors
// if no custom errors provided then it sends the default error message
func (ctx *Context) NotFound() {
	ctx.framework.EmitError(StatusNotFound, ctx)
}

// Panic emits an error 500 to the client, using the custom http errors
// if no custom errors rpovided then it sends the default error message
func (ctx *Context) Panic() {
	ctx.framework.EmitError(StatusInternalServerError, ctx)
}

// EmitError executes the custom error by the http status code passed to the function
func (ctx *Context) EmitError(statusCode int) {
	ctx.framework.EmitError(statusCode, ctx)
	ctx.StopExecution()
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------------------------Raw write methods---------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// Write writes the contents to the response writer.
//
// Returns the number of bytes written and any write error encountered
func (ctx *Context) Write(contents []byte) (n int, err error) {
	return ctx.ResponseWriter.Write(contents)
}

// Writef formats according to a format specifier and writes to the response.
//
// Returns the number of bytes written and any write error encountered
func (ctx *Context) Writef(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(ctx.ResponseWriter, format, a...)
}

// WriteString writes a simple string to the response.
//
// Returns the number of bytes written and any write error encountered
func (ctx *Context) WriteString(s string) (n int, err error) {
	return io.WriteString(ctx.ResponseWriter, s)
}

// SetBodyString writes a simple string to the response.
func (ctx *Context) SetBodyString(s string) {
	ctx.ResponseWriter.SetBodyString(s)
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------Context's gzip inline response writer ----------------------
// ---------------------Look template.go & iris.go for more options---------------------
// -------------------------------------------------------------------------------------

var (
	errClientDoesNotSupportGzip = errors.New("Client doesn't supports gzip compression")
)

func (ctx *Context) clientAllowsGzip() bool {
	if h := ctx.RequestHeader(acceptEncodingHeader); h != "" {
		for _, v := range strings.Split(h, ";") {
			if strings.Contains(v, "gzip") { // we do Contains because sometimes browsers has the q=, we don't use it atm. || strings.Contains(v,"deflate"){
				return true
			}
		}
	}
	return false
}

// WriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
// returns the number of bytes written and an error ( if the client doesn' supports gzip compression)
func (ctx *Context) WriteGzip(b []byte) (int, error) {
	if ctx.clientAllowsGzip() {
		ctx.ResponseWriter.Header().Add(varyHeader, acceptEncodingHeader)

		gzipWriter := fs.AcquireGzipWriter(ctx.ResponseWriter)
		n, err := gzipWriter.Write(b)
		fs.ReleaseGzipWriter(gzipWriter)

		if err == nil {
			ctx.SetHeader(contentEncodingHeader, "gzip")
		} // else write the contents as it is? no let's create a new func for this
		return n, err
	}
	return 0, errClientDoesNotSupportGzip
}

// TryWriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
// If client does not supprots gzip then the contents are written as they are, uncompressed.
func (ctx *Context) TryWriteGzip(b []byte) (int, error) {
	n, err := ctx.WriteGzip(b)
	if err != nil {
		// check if the error came from gzip not allowed and not the writer itself
		if _, ok := err.(*errors.Error); ok {
			// client didn't supported gzip, write them uncompressed:
			return ctx.ResponseWriter.Write(b)
		}
	}
	return n, err
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------------------------Render and powerful content negotiation-----------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// renderSerialized renders contents with a serializer with status OK which you can change using RenderWithStatus or ctx.SetStatusCode(iris.StatusCode)
func (ctx *Context) renderSerialized(contentType string, obj interface{}, options ...map[string]interface{}) error {
	s := ctx.framework.serializers
	finalResult, err := s.Serialize(contentType, obj, options...)
	if err != nil {
		return err
	}
	gzipEnabled := ctx.framework.Config.Gzip
	charset := ctx.framework.Config.Charset
	if len(options) > 0 {
		gzipEnabled = getGzipOption(gzipEnabled, options[0]) // located to the template.go below the RenderOptions
		charset = getCharsetOption(charset, options[0])
	}
	ctype := contentType

	if ctype == contentMarkdown { // remember the text/markdown is just a custom internal iris content type, which in reallity renders html
		ctype = contentHTML
	}

	if ctype != contentBinary { // set the charset only on non-binary data
		ctype += "; charset=" + charset
	}
	ctx.SetContentType(ctype)
	if gzipEnabled {
		ctx.TryWriteGzip(finalResult)
	} else {
		ctx.ResponseWriter.Write(finalResult)
	}
	ctx.SetStatusCode(StatusOK)
	return nil
}

// RenderTemplateSource serves a template source(raw string contents) from  the first template engines which supports raw parsing returns its result as string
func (ctx *Context) RenderTemplateSource(status int, src string, binding interface{}, options ...map[string]interface{}) error {
	err := ctx.framework.templates.renderSource(ctx, src, binding, options...)
	if err == nil {
		ctx.SetStatusCode(status)
	}

	return err
}

// RenderWithStatus builds up the response from the specified template or a serialize engine.
// Note: the options: "gzip" and "charset" are built'n support by Iris, so you can pass these on any template engine or serialize engines
func (ctx *Context) RenderWithStatus(status int, name string, binding interface{}, options ...map[string]interface{}) (err error) {
	if strings.IndexByte(name, '.') > -1 { //we have template
		err = ctx.framework.templates.renderFile(ctx, name, binding, options...)
	} else {
		err = ctx.renderSerialized(name, binding, options...)
	}

	if err == nil {
		ctx.SetStatusCode(status)
	}

	return
}

// Render same as .RenderWithStatus but with status to iris.StatusOK (200) if no previous status exists
// builds up the response from the specified template or a serialize engine.
// Note: the options: "gzip" and "charset" are built'n support by Iris, so you can pass these on any template engine or serialize engine
func (ctx *Context) Render(name string, binding interface{}, options ...map[string]interface{}) error {
	errCode := ctx.ResponseWriter.StatusCode()
	if errCode <= 0 {
		errCode = StatusOK
	}
	return ctx.RenderWithStatus(errCode, name, binding, options...)
}

// MustRender same as .Render but returns 503 service unavailable http status with a (html) message if render failed
// Note: the options: "gzip" and "charset" are built'n support by Iris, so you can pass these on any template engine or serialize engine
func (ctx *Context) MustRender(name string, binding interface{}, options ...map[string]interface{}) {
	if err := ctx.Render(name, binding, options...); err != nil {
		ctx.HTML(StatusServiceUnavailable, fmt.Sprintf("<h2>Template: %s</h2><b>%s</b>", name, err.Error()))
		if ctx.framework.Config.IsDevelopment {
			ctx.framework.Logger.Printf("MustRender panics on template: %s.Trace: %s\n", name, err)
		}
	}
}

// TemplateString accepts a template filename, its context data and returns the result of the parsed template (string)
// if any error returns empty string
func (ctx *Context) TemplateString(name string, binding interface{}, options ...map[string]interface{}) string {
	return ctx.framework.TemplateString(name, binding, options...)
}

// HTML writes html string with a http status
func (ctx *Context) HTML(status int, htmlContents string) {
	if err := ctx.RenderWithStatus(status, contentHTML, htmlContents); err != nil {
		// if no serialize engine found for text/html
		ctx.SetContentType(contentHTML + "; charset=" + ctx.framework.Config.Charset)
		ctx.SetStatusCode(status)
		ctx.WriteString(htmlContents)
	}
}

// Data writes out the raw bytes as binary data.
func (ctx *Context) Data(status int, v []byte) error {
	return ctx.RenderWithStatus(status, contentBinary, v)
}

// JSON marshals the given interface object and writes the JSON response.
func (ctx *Context) JSON(status int, v interface{}) error {
	return ctx.RenderWithStatus(status, contentJSON, v)
}

// JSONP marshals the given interface object and writes the JSON response.
func (ctx *Context) JSONP(status int, callback string, v interface{}) error {
	return ctx.RenderWithStatus(status, contentJSONP, v, map[string]interface{}{"callback": callback})
}

// Text writes out a string as plain text.
func (ctx *Context) Text(status int, v string) error {
	return ctx.RenderWithStatus(status, contentText, v)
}

// XML marshals the given interface object and writes the XML response.
func (ctx *Context) XML(status int, v interface{}) error {
	return ctx.RenderWithStatus(status, contentXML, v)
}

// MarkdownString parses the (dynamic) markdown string and returns the converted html string
func (ctx *Context) MarkdownString(markdownText string) string {
	return ctx.framework.SerializeToString(contentMarkdown, markdownText)
}

// Markdown parses and renders to the client a particular (dynamic) markdown string
// accepts two parameters
// first is the http status code
// second is the markdown string
func (ctx *Context) Markdown(status int, markdown string) {
	ctx.HTML(status, ctx.MarkdownString(markdown))
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------Static content serve by context implementation-------------------
// --------------------Look iris.go for more useful Static web system methods-----------
// -------------------------------------------------------------------------------------

// staticCachePassed checks the IfModifiedSince header and
// returns true if (client-side) duration has expired
func (ctx *Context) staticCachePassed(modtime time.Time) bool {
	if t, err := time.Parse(ctx.framework.Config.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && modtime.Before(t.Add(StaticCacheDuration)) {
		ctx.ResponseWriter.Header().Del(contentType)
		ctx.ResponseWriter.Header().Del(contentLength)
		ctx.SetStatusCode(StatusNotModified)
		return true
	}
	return false
}

// SetClientCachedBody like SetBody but it sends with an expiration datetime
// which is managed by the client-side (all major browsers supports this feature)
func (ctx *Context) SetClientCachedBody(status int, bodyContent []byte, cType string, modtime time.Time) {
	if ctx.staticCachePassed(modtime) {
		return
	}

	modtimeFormatted := modtime.UTC().Format(ctx.framework.Config.TimeFormat)

	ctx.ResponseWriter.Header().Set(contentType, cType)
	ctx.ResponseWriter.Header().Set(lastModified, modtimeFormatted)
	ctx.SetStatusCode(status)

	ctx.ResponseWriter.Write(bodyContent)
}

// ServeContent serves content, headers are autoset
// receives three parameters, it's low-level function, instead you can use .ServeFile(string,bool)/SendFile(string,string)
//
// You can define your own "Content-Type" header also, after this function call
// Doesn't implements resuming (by range), use ctx.SendFile instead
func (ctx *Context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error {
	if t, err := time.Parse(ctx.framework.Config.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.ResponseWriter.Header().Del(contentType)
		ctx.ResponseWriter.Header().Del(contentLength)
		ctx.SetStatusCode(StatusNotModified)
		return nil
	}

	ctx.ResponseWriter.Header().Set(contentType, fs.TypeByExtension(filename))
	ctx.ResponseWriter.Header().Set(lastModified, modtime.UTC().Format(ctx.framework.Config.TimeFormat))
	ctx.SetStatusCode(StatusOK)
	var out io.Writer
	if gzipCompression && ctx.clientAllowsGzip() {
		ctx.ResponseWriter.Header().Add(varyHeader, acceptEncodingHeader)
		ctx.SetHeader(contentEncodingHeader, "gzip")

		gzipWriter := fs.AcquireGzipWriter(ctx.ResponseWriter)
		defer fs.ReleaseGzipWriter(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.ResponseWriter
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
// Use it when you want to serve css/js/... files to the client, for bigger files and 'force-download' use the SendFile
func (ctx *Context) ServeFile(filename string, gzipCompression bool) error {
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
// Use this instead of ServeFile to 'force-download' bigger files to the client
func (ctx *Context) SendFile(filename string, destinationName string) {
	ctx.ServeFile(filename, false)
	ctx.ResponseWriter.Header().Set(contentDisposition, "attachment;filename="+destinationName)
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------Storage----------------------------------------------
// -----------------------User Values &  Path parameters--------------------------------
// -------------------------------------------------------------------------------------

// ValuesLen returns the total length of the user values storage, some of them maybe path parameters
func (ctx *Context) ValuesLen() (n int) {
	return len(ctx.values)
}

// Get returns the user's value from a key
// if doesn't exists returns nil
func (ctx *Context) Get(key string) interface{} {
	return ctx.values.Get(key)
}

// GetFmt returns a value which has this format: func(format string, args ...interface{}) string
// if doesn't exists returns nil
func (ctx *Context) GetFmt(key string) func(format string, args ...interface{}) string {
	if v, ok := ctx.Get(key).(func(format string, args ...interface{}) string); ok {
		return v
	}
	return func(format string, args ...interface{}) string { return "" }

}

// GetString same as Get but returns the value as string
// if nothing founds returns empty string ""
func (ctx *Context) GetString(key string) string {
	if v, ok := ctx.Get(key).(string); ok {
		return v
	}

	return ""
}

var errIntParse = errors.New("Unable to find or parse the integer, found: %#v")

// GetInt same as Get but tries to convert the return value as integer
// if nothing found or canno be parsed to integer it returns an error
func (ctx *Context) GetInt(key string) (int, error) {
	v := ctx.Get(key)
	if vint, ok := v.(int); ok {
		return vint, nil
	} else if vstring, sok := v.(string); sok {
		return strconv.Atoi(vstring)
	}

	return -1, errIntParse.Format(v)
}

// Set sets a value to a key in the values map
func (ctx *Context) Set(key string, value interface{}) {
	ctx.values.Set(key, value)
}

// VisitValues calls visitor for each existing context's temp values.
//
// visitor must not retain references to key and value after returning.
// Make key and/or value copies if you need storing them after returning.
func (ctx *Context) VisitValues(visitor func([]byte, interface{})) {
	for i, n := 0, len(ctx.values); i < n; i++ {
		kv := &ctx.values[i]
		visitor(kv.key, kv.value)
	}
}

// ParamsLen tries to return all the stored values which values are string, probably most of them will be the path parameters
func (ctx *Context) ParamsLen() (n int) {
	ctx.VisitValues(func(kb []byte, vg interface{}) {
		if _, ok := vg.(string); ok {
			n++
		}

	})
	return
}

// Param returns the string representation of the key's path named parameter's value
// same as GetString
func (ctx *Context) Param(key string) string {
	return ctx.GetString(key)
}

// ParamDecoded returns a url-query-decoded path parameter's value
func (ctx *Context) ParamDecoded(key string) string {
	return DecodeQuery(DecodeQuery(ctx.Param(key)))
}

// ParamInt returns the int representation of the key's path named parameter's value
// same as GetInt
func (ctx *Context) ParamInt(key string) (int, error) {
	return ctx.GetInt(key)
}

// ParamInt64 returns the int64 representation of the key's path named parameter's value
func (ctx *Context) ParamInt64(key string) (int64, error) {
	return strconv.ParseInt(ctx.Param(key), 10, 64)
}

// ParamsSentence returns a string implementation of all parameters that this context  keeps
// hasthe form of key1=value1,key2=value2...
func (ctx *Context) ParamsSentence() string {
	var buff bytes.Buffer
	ctx.VisitValues(func(kb []byte, vg interface{}) {
		v, ok := vg.(string)
		if !ok {
			return
		}
		k := string(kb)
		buff.WriteString(k)
		buff.WriteString("=")
		buff.WriteString(v)
		// we don't know where that (yet) stops so...
		buff.WriteString(",")

	})
	result := buff.String()
	if len(result) < 2 {
		return ""
	}

	return result[0 : len(result)-1]

}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// -----------https://github.com/golang/net/blob/master/context/context.go--------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// Deadline returns the time when work done on behalf of this context
// should be canceled.  Deadline returns ok==false when no deadline is
// set.  Successive calls to Deadline return the same results.
func (ctx *Context) Deadline() (deadline time.Time, ok bool) {
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
func (ctx *Context) Done() <-chan struct{} {
	return nil
}

// Err returns a non-nil error value after Done is closed.  Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed.  No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (ctx *Context) Err() error {
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
func (ctx *Context) Value(key interface{}) interface{} {
	if key == 0 {
		return ctx.Request
	}
	if k, ok := key.(string); ok {
		return ctx.GetString(k)
	}
	return nil
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------Session & Cookies------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// VisitAllCookies takes a visitor which loops on each (request's) cookie key and value
func (ctx *Context) VisitAllCookies(visitor func(key string, value string)) {
	for _, cookie := range ctx.Request.Cookies() {
		visitor(cookie.Name, cookie.Value)
	}
}

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found
func (ctx *Context) GetCookie(name string) string {
	cookie, err := ctx.Request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetCookie adds a cookie
func (ctx *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.ResponseWriter, cookie)
}

// SetCookieKV adds a cookie, receives just a key(string) and a value(string)
//
// Expires on 2 hours by default(unchable)
// use ctx.SetCookie or http.SetCookie instead for more control.
func (ctx *Context) SetCookieKV(name, value string) {
	c := &http.Cookie{}
	c.Name = name
	c.Value = value
	c.HttpOnly = true
	c.Expires = time.Now().Add(time.Duration(120) * time.Minute)
	ctx.SetCookie(c)
}

// RemoveCookie deletes a cookie by it's name/key
func (ctx *Context) RemoveCookie(name string) {
	c := &http.Cookie{}
	c.Name = name
	c.Value = ""
	c.Path = "/"
	c.HttpOnly = true
	exp := time.Now().Add(-time.Duration(1) * time.Minute) //RFC says 1 second, but let's do it 1 minute to make sure is working...
	c.Expires = exp
	c.MaxAge = -1
	ctx.SetCookie(c)
	// delete request's cookie also, which is temporarly available
	ctx.Request.Header.Set("Cookie", "")
}

// Session returns the current session ( && flash messages )
func (ctx *Context) Session() sessions.Session {
	if ctx.framework.sessions == nil { // this should never return nil but FOR ANY CASE, on future changes.
		return nil
	}

	if ctx.session == nil {
		ctx.session = ctx.framework.sessions.Start(ctx.ResponseWriter, ctx.Request)
	}
	return ctx.session
}

// SessionDestroy destroys the whole session, calls the provider's destroy and remove the cookie
func (ctx *Context) SessionDestroy() {
	if sess := ctx.Session(); sess != nil {
		ctx.framework.sessions.Destroy(ctx.ResponseWriter, ctx.Request)
	}

}

var maxAgeExp = regexp.MustCompile(`maxage=(\d+)`)

// MaxAge returns the "cache-control" request header's value
// seconds as int64
// if header not found or parse failed then it returns -1
func (ctx *Context) MaxAge() int64 {
	header := ctx.RequestHeader(cacheControl)
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

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------Transactions-----------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// skipTransactionsContextKey set this to any value to stop executing next transactions
// it's a context-key in order to be used from anywhere, set it by calling the SkipTransactions()
const skipTransactionsContextKey = "__IRIS_TRANSACTIONS_SKIP___"

// SkipTransactions if called then skip the rest of the transactions
// or all of them if called before the first transaction
func (ctx *Context) SkipTransactions() {
	ctx.Set(skipTransactionsContextKey, 1)
}

// TransactionsSkipped returns true if the transactions skipped or canceled at all.
func (ctx *Context) TransactionsSkipped() bool {
	if n, err := ctx.GetInt(skipTransactionsContextKey); err == nil && n == 1 {
		return true
	}
	return false
}

// non-detailed error log for transacton unexpected panic
var errTransactionInterrupted = errors.New("Transaction Interrupted, recovery from panic:\n%s")

// BeginTransaction starts a scoped transaction.
//
// Can't say a lot here because it will take more than 200 lines to write about.
// You can search third-party articles or books on how Business Transaction works (it's quite simple, especialy here).
//
// Note that this is unique and new
// (=I haver never seen any other examples or code in Golang on this subject, so far, as with the most of iris features...)
// it's not covers all paths,
// such as databases, this should be managed by the libraries you use to make your database connection,
// this transaction scope is only for context's response.
// Transactions have their own middleware ecosystem also, look iris.go:UseTransaction.
//
// See https://github.com/iris-contrib/examples/tree/master/transactions for more
func (ctx *Context) BeginTransaction(pipe func(transaction *Transaction)) {
	// SILLY NOTE: use of manual pipe type in order of TransactionFunc
	// in order to help editors complete the sentence here...

	// do NOT begin a transaction when the previous transaction has been failed
	// and it was requested scoped or SkipTransactions called manually.
	if ctx.TransactionsSkipped() {
		return
	}
	// get a transaction scope from the pool by passing the temp context/
	t := newTransaction(ctx)
	defer func() {
		if err := recover(); err != nil {
			if ctx.framework.Config.IsDevelopment {
				ctx.Log(errTransactionInterrupted.Format(err).Error())
			}
			// complete (again or not , doesn't matters) the scope without loud
			t.Complete(nil)
			// we continue as normal, no need to return here*
		}

		// write the temp contents to the original writer
		t.Context.ResponseWriter.writeTo(ctx.ResponseWriter)
		// give back to the transaction the original writer (SetBeforeFlush works this way and only this way)
		// this is tricky but nessecery if we want ctx.EmitError to work inside transactions
		t.Context.ResponseWriter = ctx.ResponseWriter
	}()

	// run the worker with its context clone inside.
	pipe(t)
}

// Log logs to the iris defined logger
func (ctx *Context) Log(format string, a ...interface{}) {
	ctx.framework.Logger.Printf(format, a...)
}

// Framework returns the Iris instance, containing the configuration and all other fields
func (ctx *Context) Framework() *Framework {
	return ctx.framework
}
