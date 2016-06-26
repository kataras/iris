/*
Context.go  Implements: ./context/context.go ,
files: context_renderer.go, context_storage.go, context_request.go, context_response.go
*/

package iris

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"net"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iris-contrib/errors"
	"github.com/iris-contrib/formBinder"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions/store"
	"github.com/kataras/iris/utils"
	"github.com/klauspost/compress/gzip"
	"github.com/valyala/fasthttp"
)

const (
	// DefaultUserAgent default to 'iris' but it is not used anywhere yet
	defaultUserAgent = "iris"
	// ContentType represents the header["Content-Type"]
	contentType = "Content-Type"
	// ContentLength represents the header["Content-Length"]
	contentLength = "Content-Length"
	// ContentHTML is the  string of text/html response headers
	contentHTML = "text/html"
	// ContentBINARY is the string of application/octet-stream response headers
	contentBINARY = "application/octet-stream"

	// LastModified "Last-Modified"
	lastModified = "Last-Modified"
	// IfModifiedSince "If-Modified-Since"
	ifModifiedSince = "If-Modified-Since"
	// ContentDisposition "Content-Disposition"
	contentDisposition = "Content-Disposition"

	// stopExecutionPosition used inside the Context, is the number which shows us that the context's middleware manualy stop the execution
	stopExecutionPosition = 255
)

// this pool is used everywhere needed in the iris for example inside party-> Static
var gzipWriterPool = sync.Pool{New: func() interface{} { return &gzip.Writer{} }}

// errors

var (
	errTemplateExecute  = errors.New("Unable to execute a template. Trace: %s")
	errFlashNotFound    = errors.New("Unable to get flash message. Trace: Cookie does not exists")
	errSessionNil       = errors.New("Unable to set session, Config().Session.Provider is nil, please refer to the docs!")
	errNoForm           = errors.New("Request has no any valid form")
	errWriteJSON        = errors.New("Before JSON be written to the body, JSON Encoder returned an error. Trace: %s")
	errRenderMarshalled = errors.New("Before +type Rendering, MarshalIndent returned an error. Trace: %s")
	errReadBody         = errors.New("While trying to read %s from the request body. Trace %s")
	errServeContent     = errors.New("While trying to serve content to the client. Trace %s")
)

type (
	// Map is just a conversion for a map[string]interface{}
	Map map[string]interface{}
	// Context is resetting every time a request is coming to the server
	// it is not good practice to use this object in goroutines, for these cases use the .Clone()
	Context struct {
		*fasthttp.RequestCtx
		Params    PathParameters
		framework *Framework
		//keep track all registed middleware (handlers)
		middleware   Middleware
		sessionStore store.IStore
		// pos is the position number of the Context, look .Next to understand
		pos uint8
	}
)

var _ context.IContext = &Context{}

// GetRequestCtx returns the current fasthttp context
func (ctx *Context) GetRequestCtx() *fasthttp.RequestCtx {
	return ctx.RequestCtx
}

// Reset resets the Context with a given domain.Response and domain.Request
// the context is ready-to-use after that, just like a new Context
// I use it for zero rellocation memory
func (ctx *Context) Reset(reqCtx *fasthttp.RequestCtx) {
	ctx.Params = ctx.Params[0:0]
	ctx.sessionStore = nil
	ctx.middleware = nil
	ctx.RequestCtx = reqCtx
}

// Clone use that method if you want to use the context inside a goroutine
func (ctx *Context) Clone() context.IContext {
	var cloneContext = *ctx
	cloneContext.pos = 0

	//copy params
	p := ctx.Params
	cpP := make(PathParameters, len(p))
	copy(cpP, p)
	cloneContext.Params = cpP
	//copy middleware
	m := ctx.middleware
	cpM := make(Middleware, len(m))
	copy(cpM, m)
	cloneContext.middleware = cpM

	// we don't copy the sessionStore for more than one reasons...
	return &cloneContext
}

// Do calls the first handler only, it's like Next with negative pos, used only on Router&MemoryRouter
func (ctx *Context) Do() {
	ctx.pos = 0
	ctx.middleware[0].Serve(ctx)
}

// Next calls all the next handler from the middleware stack, it used inside a middleware
func (ctx *Context) Next() {
	//set position to the next
	ctx.pos++
	midLen := uint8(len(ctx.middleware))
	//run the next
	if ctx.pos < midLen {
		ctx.middleware[ctx.pos].Serve(ctx)
	}

}

// StopExecution just sets the .pos to 255 in order to  not move to the next middlewares(if any)
func (ctx *Context) StopExecution() {
	ctx.pos = stopExecutionPosition
}

//

// IsStopped checks and returns true if the current position of the Context is 255, means that the StopExecution has called
func (ctx *Context) IsStopped() bool {
	return ctx.pos == stopExecutionPosition
}

// GetHandlerName as requested returns the stack-name of the function which the Middleware is setted from
func (ctx *Context) GetHandlerName() string {
	return runtime.FuncForPC(reflect.ValueOf(ctx.middleware[len(ctx.middleware)-1]).Pointer()).Name()
}

/* Request */

// Param returns the string representation of the key's path named parameter's value
func (ctx *Context) Param(key string) string {
	return ctx.Params.Get(key)
}

// ParamInt returns the int representation of the key's path named parameter's value
func (ctx *Context) ParamInt(key string) (int, error) {
	return strconv.Atoi(ctx.Param(key))
}

// URLParam returns the get parameter from a request , if any
func (ctx *Context) URLParam(key string) string {
	return string(ctx.RequestCtx.Request.URI().QueryArgs().Peek(key))
}

// URLParams returns a map of a list of each url(query) parameter
func (ctx *Context) URLParams() map[string]string {
	urlparams := make(map[string]string)
	ctx.RequestCtx.Request.URI().QueryArgs().VisitAll(func(key, value []byte) {
		urlparams[string(key)] = string(value)
	})
	return urlparams
}

// URLParamInt returns the url query parameter as int value from a request ,  returns error on parse fail
func (ctx *Context) URLParamInt(key string) (int, error) {
	return strconv.Atoi(ctx.URLParam(key))
}

// URLParamInt64 returns the url query parameter as int64 value from a request ,  returns error on parse fail
func (ctx *Context) URLParamInt64(key string) (int64, error) {
	return strconv.ParseInt(ctx.Param(key), 10, 64)
}

// MethodString returns the HTTP Method
func (ctx *Context) MethodString() string {
	return utils.BytesToString(ctx.Method())
}

// HostString returns the Host of the request( the url as string )
func (ctx *Context) HostString() string {
	return utils.BytesToString(ctx.Host())
}

// VirtualHostname returns the hostname that user registers, host path maybe differs from the real which is HostString, which taken from a net.listener
func (ctx *Context) VirtualHostname() string {
	realhost := ctx.HostString()
	virtualhost := ctx.framework.HTTPServer.VirtualHostname()
	hostname := strings.Replace(realhost, "127.0.0.1", virtualhost, 1)
	hostname = strings.Replace(realhost, "localhost", virtualhost, 1)
	if portIdx := strings.IndexByte(hostname, ':'); portIdx > 0 {
		hostname = hostname[0:portIdx]
	}
	return hostname
}

// PathString returns the full escaped path as string
// for unescaped use: ctx.RequestCtx.RequestURI() or RequestPath(escape bool)
func (ctx *Context) PathString() string {
	return ctx.RequestPath(true)
}

// RequestPath returns the requested path
func (ctx *Context) RequestPath(escape bool) string {
	if escape {
		return utils.BytesToString(ctx.RequestCtx.Path())
	}
	return utils.BytesToString(ctx.RequestCtx.RequestURI())
}

// RequestIP gets just the Remote Address from the client.
func (ctx *Context) RequestIP() string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(ctx.RequestCtx.RemoteAddr().String())); err == nil {
		return ip
	}
	return ""
}

// RemoteAddr is like RequestIP but it checks for proxy servers also, tries to get the real client's request IP
func (ctx *Context) RemoteAddr() string {
	header := string(ctx.RequestCtx.Request.Header.Peek("X-Real-Ip"))
	realIP := strings.TrimSpace(header)
	if realIP != "" {
		return realIP
	}
	realIP = string(ctx.RequestCtx.Request.Header.Peek("X-Forwarded-For"))
	idx := strings.IndexByte(realIP, ',')
	if idx >= 0 {
		realIP = realIP[0:idx]
	}
	realIP = strings.TrimSpace(realIP)
	if realIP != "" {
		return realIP
	}
	return ctx.RequestIP()

}

// RequestHeader returns the request header's value
// accepts one parameter, the key of the header (string)
// returns string
func (ctx *Context) RequestHeader(k string) string {
	return utils.BytesToString(ctx.RequestCtx.Request.Header.Peek(k))
}

// PostFormValue returns a single value from post request's data
func (ctx *Context) PostFormValue(name string) string {
	return string(ctx.RequestCtx.PostArgs().Peek(name))
}

// PostFormMulti returns a slice of string from post request's data
func (ctx *Context) PostFormMulti(name string) []string {
	arrBytes := ctx.PostArgs().PeekMulti(name)
	arrStr := make([]string, len(arrBytes))
	for i, v := range arrBytes {
		arrStr[i] = string(v)
	}
	return arrStr
}

// Subdomain returns the subdomain (string) of this request, if any
func (ctx *Context) Subdomain() (subdomain string) {
	host := ctx.HostString()
	if index := strings.IndexByte(host, '.'); index > 0 {
		subdomain = host[0:index]
	}

	return
}

// URLEncode returns the path encoded as url
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
/* Credits to Manish Singh @kryptodev for URLEncode */
func URLEncode(path string) string {
	if path == "" {
		return ""
	}
	u := fasthttp.AcquireURI()
	u.SetPath(path)
	encodedPath := u.String()[8:]
	fasthttp.ReleaseURI(u)
	return encodedPath
}

// ReadJSON reads JSON from request's body
func (ctx *Context) ReadJSON(jsonObject interface{}) error {
	data := ctx.RequestCtx.Request.Body()

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	err := decoder.Decode(jsonObject)

	//err != nil fix by @shiena
	if err != nil && err != io.EOF {
		return errReadBody.Format("JSON", err.Error())
	}

	return nil
}

// ReadXML reads XML from request's body
func (ctx *Context) ReadXML(xmlObject interface{}) error {
	data := ctx.RequestCtx.Request.Body()

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	err := decoder.Decode(xmlObject)
	//err != nil fix by @shiena
	if err != nil && err != io.EOF {
		return errReadBody.Format("XML", err.Error())
	}

	return nil
}

// ReadForm binds the formObject  with the form data
// it supports any kind of struct
func (ctx *Context) ReadForm(formObject interface{}) error {
	reqCtx := ctx.RequestCtx
	// first check if we have multipart form
	multipartForm, err := reqCtx.MultipartForm()
	if err == nil {
		//we have multipart form
		return errReadBody.With(formBinder.Decode(multipartForm.Value, formObject))
	}
	// if no multipart and post arguments ( means normal form)

	if reqCtx.PostArgs().Len() == 0 && reqCtx.QueryArgs().Len() == 0 {
		return errReadBody.With(errNoForm.Return())
	}

	form := make(map[string][]string, reqCtx.PostArgs().Len()+reqCtx.QueryArgs().Len())

	reqCtx.PostArgs().VisitAll(func(k []byte, v []byte) {
		key := string(k)
		value := string(v)
		// for slices
		if form[key] != nil {
			form[key] = append(form[key], value)
		} else {
			form[key] = []string{value}
		}

	})

	reqCtx.QueryArgs().VisitAll(func(k []byte, v []byte) {
		key := string(k)
		value := string(v)
		// for slices
		if form[key] != nil {
			form[key] = append(form[key], value)
		} else {
			form[key] = []string{value}
		}
	})

	return errReadBody.With(formBinder.Decode(form, formObject))
}

/* Response */

// SetContentType sets the response writer's header key 'Content-Type' to a given value(s)
func (ctx *Context) SetContentType(s string) {
	ctx.RequestCtx.Response.Header.Set(contentType, s)
}

// SetHeader write to the response writer's header to a given key the given value(s)
//
// Note: If you want to send a multi-line string as header's value use: strings.TrimSpace first.
func (ctx *Context) SetHeader(k string, v string) {
	//v = strings.TrimSpace(v)
	ctx.RequestCtx.Response.Header.Set(k, v)
}

// Redirect redirect sends a redirect response the client
// accepts 2 parameters string and an optional int
// first parameter is the url to redirect
// second parameter is the http status should send, default is 302 (StatusFound), you can set it to 301 (Permant redirect), if that's nessecery
func (ctx *Context) Redirect(urlToRedirect string, statusHeader ...int) {
	httpStatus := StatusFound // temporary redirect
	if statusHeader != nil && len(statusHeader) > 0 && statusHeader[0] > 0 {
		httpStatus = statusHeader[0]
	}
	ctx.RequestCtx.Redirect(urlToRedirect, httpStatus)
	ctx.StopExecution()
}

// RedirectTo does the same thing as Redirect but instead of receiving a uri or path it receives a route name
func (ctx *Context) RedirectTo(routeName string, args ...interface{}) {
	s := ctx.framework.URL(routeName, args...)
	if s != "" {
		ctx.Redirect(s, StatusFound)
	}
}

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

// Write writes a string to the client, something like fmt.Printf but for the web
func (ctx *Context) Write(format string, a ...interface{}) {
	//this doesn't work with gzip, so just write the []byte better |ctx.ResponseWriter.WriteString(fmt.Sprintf(format, a...))
	ctx.RequestCtx.WriteString(fmt.Sprintf(format, a...))
}

// HTML writes html string with a http status
func (ctx *Context) HTML(httpStatus int, htmlContents string) {
	ctx.SetContentType(contentHTML + ctx.framework.rest.CompiledCharset)
	ctx.RequestCtx.SetStatusCode(httpStatus)
	ctx.RequestCtx.WriteString(htmlContents)
}

// Data writes out the raw bytes as binary data.
func (ctx *Context) Data(status int, v []byte) error {
	return ctx.framework.rest.Data(ctx.RequestCtx, status, v)
}

// RenderWithStatus builds up the response from the specified template and bindings.
// Note: parameter layout has meaning only when using the iris.HTMLTemplate
func (ctx *Context) RenderWithStatus(status int, name string, binding interface{}, layout ...string) error {
	ctx.SetStatusCode(status)
	return ctx.framework.templates.Render(ctx, name, binding, layout...)
}

// Render same as .RenderWithStatus but with status to iris.StatusOK (200)
func (ctx *Context) Render(name string, binding interface{}, layout ...string) error {
	return ctx.RenderWithStatus(StatusOK, name, binding, layout...)
}

// MustRender same as .Render but returns 500 internal server http status (error) if rendering fail
func (ctx *Context) MustRender(name string, binding interface{}, layout ...string) {
	if err := ctx.Render(name, binding, layout...); err != nil {
		ctx.Panic()
		ctx.framework.Logger.Dangerf("MustRender panics for client with IP: %s On template: %s", ctx.RemoteAddr(), name)
	}
}

// TemplateString accepts a template filename, its context data and returns the result of the parsed template (string)
// if any error returns empty string
func (ctx *Context) TemplateString(name string, binding interface{}, layout ...string) string {
	return ctx.framework.TemplateString(name, binding, layout...)
}

// JSON marshals the given interface object and writes the JSON response.
func (ctx *Context) JSON(status int, v interface{}) error {
	return ctx.framework.rest.JSON(ctx.RequestCtx, status, v)
}

// JSONP marshals the given interface object and writes the JSON response.
func (ctx *Context) JSONP(status int, callback string, v interface{}) error {
	return ctx.framework.rest.JSONP(ctx.RequestCtx, status, callback, v)
}

// Text writes out a string as plain text.
func (ctx *Context) Text(status int, v string) error {
	return ctx.framework.rest.Text(ctx.RequestCtx, status, v)
}

// XML marshals the given interface object and writes the XML response.
func (ctx *Context) XML(status int, v interface{}) error {
	return ctx.framework.rest.XML(ctx.RequestCtx, status, v)
}

// MarkdownString parses the (dynamic) markdown string and returns the converted html string
func (ctx *Context) MarkdownString(markdownText string) string {
	return ctx.framework.rest.Markdown([]byte(markdownText))
}

// Markdown parses and renders to the client a particular (dynamic) markdown string
// accepts two parameters
// first is the http status code
// second is the markdown string
func (ctx *Context) Markdown(status int, markdown string) {
	ctx.HTML(status, ctx.MarkdownString(markdown))
}

// ExecuteTemplate executes a simple html template, you can use that if you already have the cached templates
// the recommended way to render is to use iris.Templates("./templates/path/*.html") and ctx.RenderFile("filename.html",struct{})
// accepts 2 parameters
// the first parameter is the template (*template.Template)
// the second parameter is the page context (interfac{})
// returns an error if any errors occurs while executing this template
func (ctx *Context) ExecuteTemplate(tmpl *template.Template, pageContext interface{}) error {
	ctx.RequestCtx.SetContentType(contentHTML + ctx.framework.rest.CompiledCharset)
	return errTemplateExecute.With(tmpl.Execute(ctx.RequestCtx.Response.BodyWriter(), pageContext))
}

// ServeContent serves content, headers are autoset
// receives three parameters, it's low-level function, instead you can use .ServeFile(string)
//
// You can define your own "Content-Type" header also, after this function call
func (ctx *Context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error {
	if t, err := time.Parse(config.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.RequestCtx.Response.Header.Del(contentType)
		ctx.RequestCtx.Response.Header.Del(contentLength)
		ctx.RequestCtx.SetStatusCode(StatusNotModified)
		return nil
	}

	ctx.RequestCtx.Response.Header.Set(contentType, utils.TypeByExtension(filename))
	ctx.RequestCtx.Response.Header.Set(lastModified, modtime.UTC().Format(config.TimeFormat))
	ctx.RequestCtx.SetStatusCode(StatusOK)
	var out io.Writer
	if gzipCompression {
		ctx.RequestCtx.Response.Header.Add("Content-Encoding", "gzip")
		gzipWriter := gzipWriterPool.Get().(*gzip.Writer)
		gzipWriter.Reset(ctx.RequestCtx.Response.BodyWriter())
		defer gzipWriter.Close()
		defer gzipWriterPool.Put(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.RequestCtx.Response.BodyWriter()

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
// You can define your own "Content-Type" header also, after this function call
// for example: ctx.Response.Header.Set("Content-Type","thecontent/type")
func (ctx *Context) SendFile(filename string, destinationName string) error {
	err := ctx.ServeFile(filename, false)
	if err != nil {
		return err
	}

	ctx.RequestCtx.Response.Header.Set(contentDisposition, "attachment;filename="+destinationName)
	return nil
}

// Stream same as StreamWriter
func (ctx *Context) Stream(cb func(writer *bufio.Writer)) {
	ctx.StreamWriter(cb)
}

// StreamWriter registers the given stream writer for populating
// response body.
//
//
// This function may be used in the following cases:
//
//     * if response body is too big (more than 10MB).
//     * if response body is streamed from slow external sources.
//     * if response body must be streamed to the client in chunks.
//     (aka `http server push`).
func (ctx *Context) StreamWriter(cb func(writer *bufio.Writer)) {
	ctx.RequestCtx.SetBodyStreamWriter(cb)
}

// StreamReader sets response body stream and, optionally body size.
//
// If bodySize is >= 0, then the bodyStream must provide exactly bodySize bytes
// before returning io.EOF.
//
// If bodySize < 0, then bodyStream is read until io.EOF.
//
// bodyStream.Close() is called after finishing reading all body data
// if it implements io.Closer.
//
// See also StreamReader.
func (ctx *Context) StreamReader(bodyStream io.Reader, bodySize int) {
	ctx.RequestCtx.Response.SetBodyStream(bodyStream, bodySize)
}

/* Storage */

// Get returns the user's value from a key
// if doesn't exists returns nil
func (ctx *Context) Get(key string) interface{} {
	return ctx.RequestCtx.UserValue(key)
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

// GetInt same as Get but returns the value as int
// if nothing founds returns -1
func (ctx *Context) GetInt(key string) int {
	if v, ok := ctx.Get(key).(int); ok {
		return v
	}

	return -1
}

// Set sets a value to a key in the values map
func (ctx *Context) Set(key string, value interface{}) {
	ctx.RequestCtx.SetUserValue(key, value)
}

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found
func (ctx *Context) GetCookie(name string) (val string) {
	bcookie := ctx.RequestCtx.Request.Header.Cookie(name)
	if bcookie != nil {
		val = string(bcookie)
	}
	return
}

// SetCookie adds a cookie
func (ctx *Context) SetCookie(cookie *fasthttp.Cookie) {
	ctx.RequestCtx.Response.Header.SetCookie(cookie)
}

// SetCookieKV adds a cookie, receives just a key(string) and a value(string)
func (ctx *Context) SetCookieKV(key, value string) {
	c := fasthttp.AcquireCookie() // &fasthttp.Cookie{}
	c.SetKey(key)
	c.SetValue(value)
	c.SetHTTPOnly(true)
	c.SetExpire(time.Now().Add(time.Duration(120) * time.Minute))
	ctx.SetCookie(c)
	fasthttp.ReleaseCookie(c)
}

// RemoveCookie deletes a cookie by it's name/key
func (ctx *Context) RemoveCookie(name string) {
	cookie := fasthttp.AcquireCookie()
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	exp := time.Now().Add(-time.Duration(1) * time.Minute) //RFC says 1 second, but make sure 1 minute because we are using fasthttp
	cookie.SetExpire(exp)
	ctx.Response.Header.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)
}

// GetFlash get a flash message by it's key
// after this action the messages is removed
// returns string, if the cookie doesn't exists the string is empty
func (ctx *Context) GetFlash(key string) string {
	val, err := ctx.GetFlashBytes(key)
	if err != nil {
		return ""
	}
	return string(val)
}

// GetFlashBytes get a flash message by it's key
// after this action the messages is removed
// returns []byte along with an error if the cookie doesn't exists or decode fails
func (ctx *Context) GetFlashBytes(key string) (value []byte, err error) {
	cookieValue := string(ctx.RequestCtx.Request.Header.Cookie(key))
	if cookieValue == "" {
		err = errFlashNotFound.Return()
	} else {
		value, err = base64.URLEncoding.DecodeString(cookieValue)
		//remove the message
		ctx.RemoveCookie(key)
		//it should'b be removed until the next reload, so we don't do that: ctx.Request.Header.SetCookie(key, "")
	}
	return
}

// SetFlash sets a flash message, accepts 2 parameters the key(string) and the value(string)
func (ctx *Context) SetFlash(key string, value string) {
	ctx.SetFlashBytes(key, utils.StringToBytes(value))
}

// SetFlashBytes sets a flash message, accepts 2 parameters the key(string) and the value([]byte)
func (ctx *Context) SetFlashBytes(key string, value []byte) {
	c := fasthttp.AcquireCookie()
	c.SetKey(key)
	c.SetValue(base64.URLEncoding.EncodeToString(value))
	c.SetPath("/")
	c.SetHTTPOnly(true)
	ctx.RequestCtx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)
}

// Session returns the current session store, returns nil if provider is ""
func (ctx *Context) Session() store.IStore {
	if ctx.framework.sessions == nil || ctx.framework.Config.Sessions.Provider == "" { //the second check can be changed on runtime, users are able to  turn off the sessions by setting provider to  ""
		return nil
	}

	if ctx.sessionStore == nil {
		ctx.sessionStore = ctx.framework.sessions.Start(ctx)
	}
	return ctx.sessionStore
}

// SessionDestroy destroys the whole session, calls the provider's destroy and remove the cookie
func (ctx *Context) SessionDestroy() {
	if ctx.framework.sessions != nil {
		if store := ctx.Session(); store != nil {
			ctx.framework.sessions.Destroy(ctx)
		}
	}

}

// Log logs to the iris defined logger
func (ctx *Context) Log(format string, a ...interface{}) {
	ctx.framework.Logger.Printf(format, a...)
}
