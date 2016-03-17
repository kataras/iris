package iris

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

const (
	// DefaultCharset represents the default charset for content headers
	DefaultCharset = "UTF-8"
	// ContentType represents the header["Content-Type"]
	ContentType = "Content-Type"
	// ContentLength represents the header["Content-Length"]
	ContentLength = "Content-Length"
	// ContentHTML is the  string of text/html response headers
	ContentHTML = "text/html" + "; " + DefaultCharset
	// ContentJSON is the  string of application/json response headers
	ContentJSON = "application/json" + "; " + DefaultCharset
	// ContentJSONP is the  string of application/javascript response headers
	ContentJSONP = "application/javascript"
	// ContentBINARY is the  string of "application/octet-stream response headers
	ContentBINARY = "application/octet-stream"
	// ContentTEXT is the  string of text/plain response headers
	ContentTEXT = "text/plain" + "; " + DefaultCharset
	// ContentXML is the  string of text/xml response headers
	ContentXML = "text/xml" + "; " + DefaultCharset
)

// Context is created every time a request is coming to the server,
// it holds a pointer to the http.Request, the ResponseWriter
// the Named Parameters (if any) of the requested path and an underline Renderer.
//
// Context is transferring to the frontend dev via the ContextedHandlerFunc at the handler.go,
// from the route.go 's Prepare -> convert handler as middleware and use route.run -> ServeHTTP.
type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params         PathParameters
	station        *Station
	//keep track all registed middleware (handlers)
	middleware Middleware
	// pos is the position number of the Context, look .Next to understand
	pos uint8
	// these values are reseting on each request, are useful only between middleware,
	// use iris/sessions for cookie/filesystem storage
	values map[string]interface{}
}

// Param returns the string representation of the key's path named parameter's value
func (ctx *Context) Param(key string) string {
	return ctx.Params.Get(key)
}

// ParamInt returns the int representation of the key's path named parameter's value
func (ctx *Context) ParamInt(key string) (int, error) {
	val, err := strconv.Atoi(ctx.Params.Get(key))
	return val, err
}

// URLParam returns the get parameter from a request , if any
func (ctx *Context) URLParam(key string) string {
	return URLParam(ctx.Request, key)
}

// URLParamInt returns the get parameter int value from a request , if any
func (ctx *Context) URLParamInt(key string) (int, error) {
	return strconv.Atoi(URLParam(ctx.Request, key))
}

// Write writes a string via the context's ResponseWriter
func (ctx *Context) Write(format string, a ...interface{}) {

	io.WriteString(ctx.ResponseWriter, fmt.Sprintf(format, a...))
}

// ServeFile is used to serve a file, via the http.ServeFile
func (ctx *Context) ServeFile(path string) {
	http.ServeFile(ctx.ResponseWriter, ctx.Request, path)
}

// GetCookie returns cookie's value by it's name
func (ctx *Context) GetCookie(name string) string {
	//thanks to  @wsantos fix cookieName to name
	_cookie, _err := ctx.Request.Cookie(name)
	if _err != nil {
		return ""
	}
	return _cookie.Value
}

// SetCookie adds a cookie to the request
func (ctx *Context) SetCookie(name string, value string) {
	c := &http.Cookie{Name: name, Value: value}
	ctx.Request.AddCookie(c)
}

// NotFound emits an error 404 to the client, using the custom http errors
// if no custom errors provided then use the default http.NotFound
// which is already registed nothing special to do here
func (ctx *Context) NotFound() {
	ctx.station.Errors().Emit(404, ctx)
}

func (ctx *Context) SendStatus(statusCode int, message string) {
	r := ctx.ResponseWriter
	r.Header().Set("Content-Type", "text/plain; charset=utf-8")
	r.Header().Set("X-Content-Type-Options", "nosniff")
	r.WriteHeader(statusCode)
	io.WriteString(r, message)
}

// RequestIP gets just the Remote Address from the client.
func (ctx *Context) RequestIP() string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(ctx.Request.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

// Close is used to close the body of the request
///TODO: CHECK FOR REQUEST CLOSED IN ORDER TO FIX SOME ERRORS HERE
func (ctx *Context) Close() {
	ctx.Request.Body.Close()
}

// End same as Close, end the response process.
func (ctx *Context) End() {
	ctx.Request.Body.Close()
}

// Clone before we had (c Context) inscope and  (c *Context) for outscope like goroutines
// now we have (c *Context) for both sittuations ,and call .Clone() if we need to pass the context in a gorotoune or to a time func
// example:
// api.Get("/user/:id", func(ctx *iris.Context) {
//		c:= ctx.Clone()
//		time.AfterFunc(20 * time.Second, func() {
//			println(" 20 secs after: from user with id:", c.Param("id"), " context req path:", c.Request.URL.Path)
//		})
//	})
func (ctx *Context) Clone() *Context {
	cloneContext := *ctx
	cloneContext.pos = 0

	//copy params
	params := cloneContext.Params
	cpP := make(PathParameters, len(params))
	copy(cpP, params)
	//copy middleware
	middleware := ctx.middleware
	cpM := make(Middleware, len(middleware))
	copy(cpM, middleware)
	cloneContext.middleware = middleware
	return &cloneContext
}

// Next calls all the  remeaning handlers from the middleware stack, it used inside a middleware
func (ctx *Context) Next() {
	//set position to the next
	ctx.pos++
	midLen := uint8(len(ctx.middleware)) // max 255 handlers, we don't except more than these logically ...
	//run the next
	if ctx.pos < midLen {
		ctx.middleware[ctx.pos].Serve(ctx)
	}

}

// do calls the first handler only, it's like Next with negative pos, used only on Router&MemoryRouter
func (ctx *Context) do() {
	ctx.pos = 0 //reset the position to re-run
	ctx.middleware[0].Serve(ctx)
}

func (ctx *Context) clear() {
	ctx.Params = ctx.Params[0:0]
	ctx.middleware = nil
	ctx.pos = 0
}

// Get returns a value from a key
// if doesn't exists returns nil
func (ctx *Context) Get(key string) interface{} {
	if ctx.values == nil {
		return nil
	}

	return ctx.values[key]
}

// GetString same as Get but returns the value as string
func (ctx *Context) GetString(key string) (value string) {
	if v := ctx.Get(key); v != nil {
		value = v.(string)
	}

	return
}

// GetInt same as Get but returns the value as int
func (ctx *Context) GetInt(key string) (value int) {
	if v := ctx.Get(key); v != nil {
		value = v.(int)
	}

	return
}

// Set sets a value to a key in the values map
func (ctx *Context) Set(key string, value interface{}) {
	if ctx.values == nil {
		ctx.values = make(map[string]interface{})
	}

	ctx.values[key] = value
}

/* RENDERER */

// RenderFile renders a file by its path and a context passed to the function
func (ctx *Context) RenderFile(file string, pageContext interface{}) error {
	return ctx.station.templates.ExecuteTemplate(ctx.ResponseWriter, file, pageContext)

}

// Render renders the template file html which is already registed to the template cache, with it's pageContext passed to the function
func (ctx *Context) Render(pageContext interface{}) error {
	return ctx.station.templates.Execute(ctx.ResponseWriter, pageContext)

}

// WriteHTML writes html string with a http status
///TODO or I will think to pass an interface on handlers as second parameter near to the Context, with developer's custom Renderer package .. I will think about it.
func (ctx *Context) WriteHTML(httpStatus int, htmlContents string) {
	ctx.ResponseWriter.Header().Set(ContentType, ContentHTML)
	ctx.ResponseWriter.WriteHeader(httpStatus)
	io.WriteString(ctx.ResponseWriter, htmlContents)
}

//HTML calls the WriteHTML with the 200 http status ok
func (ctx *Context) HTML(htmlContents string) {
	ctx.WriteHTML(http.StatusOK, htmlContents)
}

// WriteData writes binary data with a http status
func (ctx *Context) WriteData(httpStatus int, binaryData []byte) {
	ctx.ResponseWriter.Header().Set(ContentType, ContentBINARY)
	ctx.ResponseWriter.Header().Set(ContentLength, strconv.Itoa(len(binaryData)))
	ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.ResponseWriter.Write(binaryData)
}

//Data calls the WriteData with the 200 http status ok
func (ctx *Context) Data(binaryData []byte) {
	ctx.WriteData(http.StatusOK, binaryData)
}

// WriteText writes text with a http status
func (ctx *Context) WriteText(httpStatus int, text string) {
	ctx.ResponseWriter.Header().Set(ContentType, ContentTEXT)
	ctx.ResponseWriter.WriteHeader(httpStatus)
	io.WriteString(ctx.ResponseWriter, text)
}

//Text calls the WriteText with the 200 http status ok
func (ctx *Context) Text(text string) {
	ctx.WriteText(http.StatusOK, text)
}

// RenderJSON renders json objects with indent
func (ctx *Context) RenderJSON(httpStatus int, jsonStructs ...interface{}) error {
	var _json []byte

	for _, jsonStruct := range jsonStructs {

		theJSON, err := json.MarshalIndent(jsonStruct, "", "  ")
		if err != nil {
			return err
		}
		_json = append(_json, theJSON...)
	}

	//keep in mind http.DetectContentType(data)
	ctx.ResponseWriter.Header().Set(ContentType, ContentJSON)
	ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.ResponseWriter.Write(_json)

	return nil
}

// WriteJSON writes JSON which is encoded from a single json object or array with no Indent
func (ctx *Context) WriteJSON(httpStatus int, jsonObjectOrArray interface{}) error {
	ctx.ResponseWriter.Header().Set(ContentType, ContentJSON)
	ctx.ResponseWriter.WriteHeader(httpStatus)

	return json.NewEncoder(ctx.ResponseWriter).Encode(jsonObjectOrArray)
}

//JSON calls the WriteJSON with the 200 http status ok
func (ctx *Context) JSON(jsonObjectOrArray interface{}) error {
	return ctx.WriteJSON(http.StatusOK, jsonObjectOrArray)
}

// WriteXML writes xml which is converted from struct(s) with a http status which they passed to the function via parameters
func (ctx *Context) WriteXML(httpStatus int, xmlStructs ...interface{}) error {
	var _xmlDoc []byte
	for _, xmlStruct := range xmlStructs {
		theDoc, err := xml.MarshalIndent(xmlStruct, "", "  ")
		if err != nil {
			return err
		}
		_xmlDoc = append(_xmlDoc, theDoc...)
	}
	ctx.ResponseWriter.Header().Set(ContentType, ContentXML)
	ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.ResponseWriter.Write(_xmlDoc)
	return nil
}

//XML calls the WriteXML with the 200 http status ok
func (ctx *Context) XML(xmlStructs ...interface{}) error {
	return ctx.WriteXML(http.StatusOK, xmlStructs...)
}

/* END OF RENDERER */
