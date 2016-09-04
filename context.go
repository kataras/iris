/*
Context.go  Implements: ./context/context.go
*/

package iris

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
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

	"github.com/iris-contrib/formBinder"
	"github.com/kataras/go-errors"
	"github.com/kataras/go-fs"
	"github.com/kataras/go-sessions"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/context"
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

	// stopExecutionPosition used inside the Context, is the number which shows us that the context's middleware manualy stop the execution
	stopExecutionPosition = 255
	// used inside GetFlash to store the lifetime request flash messages
	flashMessagesStoreContextKey = "_iris_flash_messages_"
	flashMessageCookiePrefix     = "_iris_flash_message_"
	cookieHeaderID               = "Cookie: "
	cookieHeaderIDLen            = len(cookieHeaderID)
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
	// should not be used inside Render when PongoEngine is used.
	Map map[string]interface{}
	// Context is resetting every time a request is coming to the server
	// it is not good practice to use this object in goroutines, for these cases use the .Clone()
	Context struct {
		*fasthttp.RequestCtx
		Params    PathParameters
		framework *Framework
		//keep track all registed middleware (handlers)
		middleware Middleware
		session    sessions.Session
		// pos is the position number of the Context, look .Next to understand
		pos uint8
	}
)

var _ context.IContext = &Context{}

// GetRequestCtx returns the current fasthttp context
func (ctx *Context) GetRequestCtx() *fasthttp.RequestCtx {
	return ctx.RequestCtx
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

// ParamInt64 returns the int64 representation of the key's path named parameter's value
func (ctx *Context) ParamInt64(key string) (int64, error) {
	return strconv.ParseInt(ctx.Param(key), 10, 64)
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
	return strconv.ParseInt(ctx.URLParam(key), 10, 64)
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
	hostname := realhost
	virtualhost := ctx.framework.Servers.Main().Hostname()

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

// PathString returns the full escaped path as string
// for unescaped use: ctx.RequestCtx.RequestURI() or RequestPath(escape bool)
func (ctx *Context) PathString() string {
	return ctx.RequestPath(!ctx.framework.Config.DisablePathEscape)
}

// RequestPath returns the requested path
func (ctx *Context) RequestPath(escape bool) string {
	if escape {
		//	return utils.BytesToString(ctx.RequestCtx.Path())
		return utils.BytesToString(ctx.RequestCtx.URI().PathOriginal())
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

// FormValueString returns a single value, as string, from post request's data
func (ctx *Context) FormValueString(name string) string {
	return string(ctx.FormValue(name))
}

// FormValues returns a slice of string from post request's data
func (ctx *Context) FormValues(name string) []string {
	arrBytes := ctx.PostArgs().PeekMulti(name)
	arrStr := make([]string, len(arrBytes))
	for i, v := range arrBytes {
		arrStr[i] = string(v)
	}
	return arrStr
}

// PostValuesAll returns all post data values with their keys
// multipart, form data, get & post query arguments
func (ctx *Context) PostValuesAll() (valuesAll map[string][]string) {
	reqCtx := ctx.RequestCtx
	valuesAll = make(map[string][]string)
	// first check if we have multipart form
	multipartForm, err := reqCtx.MultipartForm()
	if err == nil {
		//we have multipart form
		return multipartForm.Value
	}
	// if no multipart and post arguments ( means normal form)

	if reqCtx.PostArgs().Len() == 0 && reqCtx.QueryArgs().Len() == 0 {
		return // no found
	}

	reqCtx.PostArgs().VisitAll(func(k []byte, v []byte) {
		key := string(k)
		value := string(v)
		// for slices
		if valuesAll[key] != nil {
			valuesAll[key] = append(valuesAll[key], value)
		} else {
			valuesAll[key] = []string{value}
		}

	})

	reqCtx.QueryArgs().VisitAll(func(k []byte, v []byte) {
		key := string(k)
		value := string(v)
		// for slices
		if valuesAll[key] != nil {
			valuesAll[key] = append(valuesAll[key], value)
		} else {
			valuesAll[key] = []string{value}
		}
	})

	return
}

// PostValues returns the post data values as []string of a single key/name
func (ctx *Context) PostValues(name string) []string {
	values := make([]string, 0)
	if v := ctx.PostValuesAll(); v != nil && len(v) > 0 {
		values = v[name]
	}
	return values
}

// PostValue returns the post data value of a single key/name
// returns an empty string if nothing found
func (ctx *Context) PostValue(name string) string {
	if v := ctx.PostValues(name); len(v) > 0 {
		return v[0]
	}
	return ""
}

// Subdomain returns the subdomain (string) of this request, if any
func (ctx *Context) Subdomain() (subdomain string) {
	host := ctx.HostString()
	if index := strings.IndexByte(host, '.'); index > 0 {
		subdomain = host[0:index]
	}

	return
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
		return errReadBody.With(errNoForm)
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
	httpStatus := StatusFound // a 'temporary-redirect-like' wich works better than for our purpose
	if statusHeader != nil && len(statusHeader) > 0 && statusHeader[0] > 0 {
		httpStatus = statusHeader[0]
	}
	ctx.RequestCtx.Redirect(urlToRedirect, httpStatus)

	/* you can use one of these if you want to customize the redirection:
		1.
			u := fasthttp.AcquireURI()
			ctx.URI().CopyTo(u)
			u.Update(urlToRedirect)
			ctx.SetHeader("Location", string(u.FullURI()))
			fasthttp.ReleaseURI(u)
			ctx.SetStatusCode(httpStatus)
	2.
			ctx.SetHeader("Location", urlToRedirect)
			ctx.SetStatusCode(httpStatus)
	*/

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

// Gzip accepts bytes, which are compressed to gzip format and sent to the client
func (ctx *Context) Gzip(b []byte, status int) {
	ctx.RequestCtx.Response.Header.Add(varyHeader, acceptEncodingHeader)

	if ctx.clientAllowsGzip() {
		_, err := fasthttp.WriteGzip(ctx.RequestCtx.Response.BodyWriter(), b)
		if err == nil {
			ctx.SetHeader(contentEncodingHeader, "gzip")
		}
	}
}

// RenderWithStatus builds up the response from the specified template or a response engine.
// Note: the options: "gzip" and "charset" are built'n support by Iris, so you can pass these on any template engine or response engine
func (ctx *Context) RenderWithStatus(status int, name string, binding interface{}, options ...map[string]interface{}) error {
	ctx.SetStatusCode(status)
	if strings.IndexByte(name, '.') > -1 { //we have template
		return ctx.framework.templates.render(ctx, name, binding, options...)
	}
	return ctx.framework.responses.getBy(name).render(ctx, binding, options...)
}

// Render same as .RenderWithStatus but with status to iris.StatusOK (200) if no previous status exists
// builds up the response from the specified template or a response engine.
// Note: the options: "gzip" and "charset" are built'n support by Iris, so you can pass these on any template engine or response engine
func (ctx *Context) Render(name string, binding interface{}, options ...map[string]interface{}) error {
	errCode := ctx.RequestCtx.Response.StatusCode()
	if errCode <= 0 {
		errCode = StatusOK
	}
	return ctx.RenderWithStatus(errCode, name, binding, options...)
}

// MustRender same as .Render but returns 500 internal server http status (error) if rendering fail
// builds up the response from the specified template or a response engine.
// Note: the options: "gzip" and "charset" are built'n support by Iris, so you can pass these on any template engine or response engine
func (ctx *Context) MustRender(name string, binding interface{}, options ...map[string]interface{}) {
	if err := ctx.Render(name, binding, options...); err != nil {
		ctx.Panic()
		ctx.framework.Logger.Dangerf("MustRender panics for client with IP: %s On template: %s.Trace: %s\n", ctx.RemoteAddr(), name, err)
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
		// if no response engine found for text/html
		ctx.SetContentType(contentHTML + "; charset=" + ctx.framework.Config.Charset)
		ctx.RequestCtx.SetStatusCode(status)
		ctx.RequestCtx.WriteString(htmlContents)
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
	return ctx.framework.ResponseString(contentMarkdown, markdownText)
}

// Markdown parses and renders to the client a particular (dynamic) markdown string
// accepts two parameters
// first is the http status code
// second is the markdown string
func (ctx *Context) Markdown(status int, markdown string) {
	ctx.HTML(status, ctx.MarkdownString(markdown))
}

// ServeContent serves content, headers are autoset
// receives three parameters, it's low-level function, instead you can use .ServeFile(string,bool)/SendFile(string,string)
//
// You can define your own "Content-Type" header also, after this function call
// Doesn't implements resuming (by range), use ctx.SendFile instead
func (ctx *Context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error {
	if t, err := time.Parse(config.TimeFormat, ctx.RequestHeader(ifModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.RequestCtx.Response.Header.Del(contentType)
		ctx.RequestCtx.Response.Header.Del(contentLength)
		ctx.RequestCtx.SetStatusCode(StatusNotModified)
		return nil
	}

	ctx.RequestCtx.Response.Header.Set(contentType, fs.TypeByExtension(filename))
	ctx.RequestCtx.Response.Header.Set(lastModified, modtime.UTC().Format(config.TimeFormat))
	ctx.RequestCtx.SetStatusCode(StatusOK)
	var out io.Writer
	if gzipCompression && ctx.clientAllowsGzip() {
		ctx.RequestCtx.Response.Header.Add(varyHeader, acceptEncodingHeader)
		ctx.SetHeader(contentEncodingHeader, "gzip")
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
// This function doesn't implement resuming (by range), use ctx.SendFile/fasthttp.ServeFileUncompressed(ctx.RequestCtx,path)/fasthttpServeFile(ctx.RequestCtx,path) instead
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
	ctx.RequestCtx.SendFile(filename)
	ctx.RequestCtx.Response.Header.Set(contentDisposition, "attachment;filename="+destinationName)
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

// VisitAllCookies takes a visitor which loops on each (request's) cookie key and value
//
// Note: the method ctx.Request.Header.VisitAllCookie by fasthttp, has a strange bug which I cannot solve at the moment.
// This is the reason which this function exists and should be used instead of fasthttp's built'n.
func (ctx *Context) VisitAllCookies(visitor func(key string, value string)) {
	// strange bug, this doesnt works also: 	cookieHeaderContent := ctx.Request.Header.Peek("Cookie")/User-Agent tested also
	headerbody := string(ctx.Request.Header.Header())
	headerlines := strings.Split(headerbody, "\n")
	for _, s := range headerlines {
		if len(s) > cookieHeaderIDLen {
			if s[0:cookieHeaderIDLen] == cookieHeaderID {
				contents := s[cookieHeaderIDLen:]
				values := strings.Split(contents, "; ")
				for _, s := range values {
					keyvalue := strings.SplitN(s, "=", 2)
					visitor(keyvalue[0], keyvalue[1])
				}
			}
		}
	}
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
	c := fasthttp.AcquireCookie()
	//	c := &fasthttp.Cookie{}
	c.SetKey(key)
	c.SetValue(value)
	c.SetHTTPOnly(true)
	c.SetExpire(time.Now().Add(time.Duration(120) * time.Minute))
	ctx.SetCookie(c)
	fasthttp.ReleaseCookie(c)
}

// RemoveCookie deletes a cookie by it's name/key
func (ctx *Context) RemoveCookie(name string) {
	ctx.Response.Header.DelCookie(name)

	cookie := fasthttp.AcquireCookie()
	//cookie := &fasthttp.Cookie{}
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	exp := time.Now().Add(-time.Duration(1) * time.Minute) //RFC says 1 second, but let's do it 1 minute to make sure is working...
	cookie.SetExpire(exp)
	ctx.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)
	// delete request's cookie also, which is temporarly available
	ctx.Request.Header.DelCookie(name)
}

// GetFlashes returns all the flash messages for available for this request
func (ctx *Context) GetFlashes() map[string]string {
	// if already taken at least one time, this will be filled
	if messages := ctx.Get(flashMessagesStoreContextKey); messages != nil {
		if m, isMap := messages.(map[string]string); isMap {
			return m
		}
	} else {
		flashMessageFound := false
		// else first time, get all flash cookie keys(the prefix will tell us which is a flash message), and after get all one-by-one using the GetFlash.
		flashMessageCookiePrefixLen := len(flashMessageCookiePrefix)
		ctx.VisitAllCookies(func(key string, value string) {
			if len(key) > flashMessageCookiePrefixLen {
				if key[0:flashMessageCookiePrefixLen] == flashMessageCookiePrefix {
					unprefixedKey := key[flashMessageCookiePrefixLen:]
					_, err := ctx.GetFlash(unprefixedKey) // this func will add to the list (flashMessagesStoreContextKey) also
					if err == nil {
						flashMessageFound = true
					}
				}

			}
		})
		// if we found at least one flash message then re-execute this function to return the list
		if flashMessageFound {
			return ctx.GetFlashes()
		}
	}
	return nil
}

func (ctx *Context) decodeFlashCookie(key string) (string, string) {
	cookieKey := flashMessageCookiePrefix + key
	cookieValue := string(ctx.RequestCtx.Request.Header.Cookie(cookieKey))

	if cookieValue != "" {
		v, e := base64.URLEncoding.DecodeString(cookieValue)
		if e == nil {
			return cookieKey, string(v)
		}
	}
	return "", ""
}

// GetFlash get a flash message by it's key
// returns the value as string and an error
//
// if the cookie doesn't exists the string is empty and the error is filled
// after the request's life the value is removed
func (ctx *Context) GetFlash(key string) (string, error) {

	// first check if flash exists from this request's lifetime, if yes return that else continue to get the cookie
	storeExists := false

	if messages := ctx.Get(flashMessagesStoreContextKey); messages != nil {
		m, isMap := messages.(map[string]string)
		if !isMap {
			return "", fmt.Errorf("Flash store is not a map[string]string. This suppose will never happen, please report this bug.")
		}
		storeExists = true // in order to skip the check later
		for k, v := range m {
			if k == key && v != "" {
				return v, nil
			}
		}
	}

	cookieKey, cookieValue := ctx.decodeFlashCookie(key)
	if cookieValue == "" {
		return "", errFlashNotFound
	}
	// store this flash message to the lifetime request's local storage,
	// I choose this method because no need to store it if not used at all
	if storeExists {
		ctx.Get(flashMessagesStoreContextKey).(map[string]string)[key] = cookieValue
	} else {
		flashStoreMap := make(map[string]string)
		flashStoreMap[key] = cookieValue
		ctx.Set(flashMessagesStoreContextKey, flashStoreMap)
	}

	//remove the real cookie, no need to have that, we stored it on lifetime request
	ctx.RemoveCookie(cookieKey)
	return cookieValue, nil
	//it should'b be removed until the next reload, so we don't do that: ctx.Request.Header.SetCookie(key, "")

}

// SetFlash sets a flash message, accepts 2 parameters the key(string) and the value(string)
// the value will be available on the NEXT request
func (ctx *Context) SetFlash(key string, value string) {
	cKey := flashMessageCookiePrefix + key
	cValue := base64.URLEncoding.EncodeToString([]byte(value))

	c := fasthttp.AcquireCookie()
	c.SetKey(cKey)
	c.SetValue(cValue)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	ctx.RequestCtx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)

	// if any bug on the future: this works, and the above:
	//ctx.RequestCtx.Request.Header.SetCookie(cKey, cValue)
	//ctx.RequestCtx.Response.Header.Add("Set-Cookie", cKey+"="+cValue+"; Path:/; HttpOnly")
	//

	/*c := &fasthttp.Cookie{}
	c.SetKey(cKey)
	c.SetValue(cValue)
	c.SetPath("/")
	c.SetHTTPOnly(true)
	ctx.SetCookie(c)*/

}

// Session returns the current session
func (ctx *Context) Session() sessions.Session {
	if ctx.framework.sessions == nil { // this should never return nil but FOR ANY CASE, on future changes.
		return nil
	}

	if ctx.session == nil {
		ctx.session = ctx.framework.sessions.Start(ctx.RequestCtx)
	}
	return ctx.session
}

// SessionDestroy destroys the whole session, calls the provider's destroy and remove the cookie
func (ctx *Context) SessionDestroy() {
	if sess := ctx.Session(); sess != nil {
		ctx.framework.sessions.Destroy(ctx.RequestCtx)
	}

}

// Log logs to the iris defined logger
func (ctx *Context) Log(format string, a ...interface{}) {
	ctx.framework.Logger.Printf(format, a...)
}
