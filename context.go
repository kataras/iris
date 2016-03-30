// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type IContext interface {
	Reset(http.ResponseWriter, *http.Request)
	Do()
	Redo(http.ResponseWriter, *http.Request)
	Next()
	GetResponseWriter() IMemoryWriter
	GetRequest() *http.Request
	GetMemoryResponseWriter() MemoryWriter
	SetMemoryResponseWriter(MemoryWriter)
	Param(key string) string
	ParamInt(key string) (int, error)
	URLParam(key string) string
	URLParamInt(key string) (int, error)
	Get(key string) interface{}
	GetString(key string) (value string)
	GetInt(key string) (value int)
	Set(key string, value interface{})
	Write(format string, a ...interface{})
	ServeFile(path string)
	GetCookie(name string) string
	SetCookie(name string, value string)
	// Errors
	NotFound()
	Panic()
	EmitError(statusCode int)
	StopExecution()
	//
	Redirect(path string, statusHeader ...int) error
	SendStatus(statusCode int, message string)
	RequestIP() string
	Close()
	End()
	IsStopped() bool
	Clone() *Context ///todo IContext again
	RenderFile(file string, pageContext interface{}) error
	Render(pageContext interface{}) error
	//
	// WriteStatus writes http status code to the header
	WriteStatus(statusCode int)
	// SetContentType sets the "Content-Type" header, receives the values
	SetContentType(s []string)
	// SetHeader sets the response headers first parameter is the key, second is the values
	SetHeader(k string, s []string)
	//
	WriteHTML(httpStatus int, htmlContents string)
	HTML(htmlContents string)
	WriteData(httpStatus int, binaryData []byte)
	Data(binaryData []byte)
	WriteText(httpStatus int, text string)
	Text(text string)
	RenderJSON(httpStatus int, jsonStructs ...interface{}) error
	ReadJSON(jsonObject interface{}) error
	WriteJSON(httpStatus int, jsonObjectOrArray interface{}) error
	JSON(jsonObjectOrArray interface{}) error
	WriteXML(httpStatus int, xmlB []byte) error
	XML(xmlB []byte) error
	RenderXML(hhttpStatus int, xmlStructs ...interface{}) error
	ReadXML(xmlObject interface{}) error
	GetHandlerName() string
}

// Charset is defaulted to UTF-8, you can change it
// all render methods will have this charset
var Charset = DefaultCharset

const (
	// DefaultCharset represents the default charset for content headers
	DefaultCharset = "UTF-8"
	// ContentType represents the header["Content-Type"]
	ContentType = "Content-Type"
	// ContentLength represents the header["Content-Length"]
	ContentLength = "Content-Length"
	// ContentHTML is the  string of text/html response headers
	ContentHTML = "text/html"
	// ContentJSON is the  string of application/json response headers
	ContentJSON = "application/json"
	// ContentJSONP is the  string of application/javascript response headers
	ContentJSONP = "application/javascript"
	// ContentBINARY is the  string of "application/octet-stream response headers
	ContentBINARY = "application/octet-stream"
	// ContentTEXT is the  string of text/plain response headers
	ContentTEXT = "text/plain"
	// ContentXML is the  string of application/xml response headers
	ContentXML = "application/xml"
	// ContentXMLText is the  string of text/xml response headers
	ContentXMLText = "text/xml"

	// stopExecutionPosition the number which shows us that the context's middleware manualy stop the execution
	stopExecutionPosition = 255 // is the biggest uint8
)

// Context is resetting every time a request is coming to the server,
// it holds a pointer to the http.Request, the ResponseWriter
// the Named Parameters (if any) of the requested path and an underline Renderer.
//
type Context struct {
	memoryResponseWriter MemoryWriter
	ResponseWriter       IMemoryWriter
	Request              *http.Request
	Params               PathParameters
	station              *Station
	//keep track all registed middleware (handlers)
	middleware Middleware
	// pos is the position number of the Context, look .Next to understand
	pos uint8
	// these values are reseting on each request, are useful only between middleware,
	// use iris/sessions for cookie/filesystem storage
	values map[string]interface{}
	mu     sync.Mutex
}

var _ IContext = &Context{}

func (ctx *Context) GetResponseWriter() IMemoryWriter {
	return ctx.ResponseWriter
}

func (ctx *Context) GetRequest() *http.Request {
	return ctx.Request
}

func (ctx *Context) SetRequest(req *http.Request) {
	ctx.Request = req
}

func (ctx *Context) SetResponseWriter(res IMemoryWriter) {
	ctx.ResponseWriter = res
}

func (ctx *Context) GetMemoryResponseWriter() MemoryWriter {
	return ctx.memoryResponseWriter
}

func (ctx *Context) SetMemoryResponseWriter(res MemoryWriter) {
	ctx.memoryResponseWriter = res
	ctx.ResponseWriter = &ctx.memoryResponseWriter
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

// Error handling

// NotFound emits an error 404 to the client, using the custom http errors
// if no custom errors provided then it sends the default http.NotFound
func (ctx *Context) NotFound() {
	ctx.StopExecution()
	ctx.station.EmitError(404, ctx)
}

// Panic stops the executions of the context and returns the registed panic handler
// or if not, the default which is  500 http status to the client
//
// This function is useful when you use the recovery middleware, which is auto-executing the (custom, registed) 500 internal server error.
func (ctx *Context) Panic() {
	ctx.StopExecution()
	ctx.station.EmitError(500, ctx)
}

// EmitError executes the custom error by the http status code passed to the function
func (ctx *Context) EmitError(statusCode int) {
	ctx.station.EmitError(statusCode, ctx)
}

func (ctx *Context) StopExecution() {
	ctx.pos = stopExecutionPosition
}

//

func (ctx *Context) Redirect(urlToRedirect string, statusHeader ...int) error {
	httpStatus := 302 // temporary redirect
	if statusHeader != nil && len(statusHeader) > 0 && statusHeader[0] > 0 {
		httpStatus = statusHeader[0]
	}
	var u *url.URL
	var err error
	if u, err = url.Parse(urlToRedirect); err == nil {
		ctx.StopExecution()
		if u.Scheme == "" && u.Host == "" {
			//The http://yourserver is done automatically by all browsers today
			//so just clean the path
			trailing := strings.HasSuffix(urlToRedirect, "/")
			urlToRedirect = path.Clean(urlToRedirect)
			//check after clean if we had a slash but after we don't, we have to do that otherwise we will get forever redirects if path is /home but the registed is /home/
			if trailing && !strings.HasSuffix(urlToRedirect, "/") {
				urlToRedirect += "/"
			}

		}
		ctx.ResponseWriter.Header().Set("Location", urlToRedirect)
		ctx.ResponseWriter.WriteHeader(httpStatus)
	}
	return err
}

func (ctx *Context) Status(statusCode int) {
	ctx.memoryResponseWriter.WriteHeader(statusCode)
}

// SendStatus sends a http status to the client
// it receives status code (int) and a message (string)
func (ctx *Context) SendStatus(statusCode int, message string) {
	r := ctx.memoryResponseWriter
	r.Header().Set("Content-Type", "text/plain"+" ;charset="+Charset)
	r.Header().Set("X-Content-Type-Options", "nosniff")
	ctx.Status(statusCode)
	//r.WriteString(message)
	r.Write([]byte(message))
}

// RemoteAddr gets just the Remote Address from the client.
func (ctx *Context) RemoteAddr() string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(ctx.Request.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

// RequestIP is like RemoteAddr but it checks for proxy servers also, tries to get the real client's request IP
func (ctx *Context) RequestIP() string {
	header := ctx.Request.Header.Get("X-Real-Ip")
	realIP := strings.TrimSpace(header)
	if realIP != "" {
		return realIP
	}
	realIP = ctx.Request.Header.Get("X-Forwarded-For")
	idx := strings.IndexByte(realIP, ',')
	if idx >= 0 {
		realIP = realIP[0:idx]
	}
	realIP = strings.TrimSpace(realIP)
	if realIP != "" {
		return realIP
	} else {
		return ctx.RemoteAddr()
	}
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

func (ctx *Context) IsStopped() bool {
	return ctx.pos == stopExecutionPosition
}

// Next calls all the next handler from the middleware stack, it used inside a middleware
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
func (ctx *Context) Do() {
	ctx.pos = 0
	ctx.middleware[0].Serve(ctx)
}

func (ctx *Context) Reset(res http.ResponseWriter, req *http.Request) {
	ctx.Params = ctx.Params[0:0]
	ctx.middleware = nil
	ctx.memoryResponseWriter.Reset(res)
	ctx.ResponseWriter = &ctx.memoryResponseWriter
	ctx.Request = req
	/*if len(ctx.values) > 0 {
		for k, _ := range ctx.values {
			delete(ctx.values, k)
		}
	}*/

}

//no pointer don't change anything. it works with the 1kkk requests no multiple write header but we have cost at memory, I must find other way to solve that.
func (ctx *Context) Redo(res http.ResponseWriter, req *http.Request) {
	//ctx.memoryResponseWriter = MemoryWriter{res, -1, 200}
	ctx.memoryResponseWriter.Reset(res)
	ctx.ResponseWriter = &ctx.memoryResponseWriter
	ctx.Request = req
	ctx.Do()
	ctx.memoryResponseWriter.ForceHeader()

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
	var cloneContext = *ctx
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

	cloneContext.memoryResponseWriter.ResponseWriter = nil
	cloneContext.ResponseWriter = &cloneContext.memoryResponseWriter
	return &cloneContext
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
	return ctx.station.GetTemplates().ExecuteTemplate(ctx.GetResponseWriter(), file, pageContext)

}

// Render renders the template file html which is already registed to the template cache, with it's pageContext passed to the function
func (ctx *Context) Render(pageContext interface{}) error {
	return ctx.station.GetTemplates().Execute(ctx.GetResponseWriter(), pageContext)

}

// WriteHTML writes html string with a http status
///TODO or I will think to pass an interface on handlers as second parameter near to the Context, with developer's custom Renderer package .. I will think about it.
func (ctx *Context) WriteHTML(httpStatus int, htmlContents string) {
	//ctx.ResponseWriter.Header().Set(ContentType, ContentHTML+" ;charset="+Charset)
	//ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.SetContentType([]string{ContentHTML + " ;charset=" + Charset})
	ctx.WriteStatus(httpStatus)
	//io.WriteString(ctx.ResponseWriter, htmlContents)
	//ctx.ResponseWriter.WriteString(htmlContents)
	ctx.ResponseWriter.Write([]byte(htmlContents))
}

//HTML calls the WriteHTML with the 200 http status ok
func (ctx *Context) HTML(htmlContents string) {
	ctx.WriteHTML(http.StatusOK, htmlContents)
}

// WriteData writes binary data with a http status
func (ctx *Context) WriteData(httpStatus int, binaryData []byte) {
	//ctx.ResponseWriter.Header().Set(ContentType, ContentBINARY)
	//ctx.ResponseWriter.Header().Set(ContentLength, strconv.Itoa(len(binaryData)))
	//ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.SetHeader(ContentType, []string{ContentBINARY + " ;charset=" + Charset})
	ctx.SetHeader(ContentLength, []string{strconv.Itoa(len(binaryData))})
	ctx.WriteStatus(httpStatus)
	ctx.ResponseWriter.Write(binaryData)
}

//Data calls the WriteData with the 200 http status ok
func (ctx *Context) Data(binaryData []byte) {
	ctx.WriteData(http.StatusOK, binaryData)
}

// Write writes a string via the context's ResponseWriter
func (ctx *Context) Write(format string, a ...interface{}) {
	//this doesn't work with gzip, so just write the []byte better |ctx.ResponseWriter.WriteString(fmt.Sprintf(format, a...))
	ctx.ResponseWriter.Write([]byte(fmt.Sprintf(format, a...)))
}

//fix https://github.com/kataras/iris/issues/44

func (ctx *Context) WriteStatus(statusCode int) {
	ctx.memoryResponseWriter.WriteHeader(statusCode)
}

func (ctx *Context) SetContentType(s []string) {
	ctx.mu.Lock()
	h := ctx.ResponseWriter.Header()
	if ss := h[ContentType]; len(ss) == 0 {
		h[ContentType] = s
	}
	ctx.mu.Unlock()
}

func (ctx *Context) SetHeader(k string, s []string) {
	ctx.mu.Lock()
	h := ctx.ResponseWriter.Header()
	if ss := h[k]; len(ss) == 0 {
		h[k] = s
	}
	ctx.mu.Unlock()
}

//
// WriteText writes text with a http status
func (ctx *Context) WriteText(httpStatus int, text string) {

	//ctx.ResponseWriter.Header().Set(ContentType, ContentTEXT+" ;charset="+Charset)
	//ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.SetContentType([]string{ContentTEXT + " ;charset=" + Charset})
	ctx.WriteStatus(httpStatus)
	//io.WriteString(ctx.ResponseWriter, text)
	//ctx.ResponseWriter.WriteString(text)
	ctx.ResponseWriter.Write([]byte(text))
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
	//ctx.ResponseWriter.Header().Set(ContentType, ContentJSON+" ;charset="+Charset)
	//ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.SetContentType([]string{ContentJSON + " ;charset=" + Charset})
	ctx.WriteStatus(httpStatus)
	ctx.ResponseWriter.Write(_json)

	return nil
}

// ReadJSON reads JSON from request's body
func (ctx *Context) ReadJSON(jsonObject interface{}) error {
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return err
	}
	defer ctx.Close()

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	err = decoder.Decode(jsonObject)

	if err != io.EOF {
		return err
	}

	return nil
}

// WriteJSON writes JSON which is encoded from a single json object or array with no Indent
func (ctx *Context) WriteJSON(httpStatus int, jsonObjectOrArray interface{}) error {
	//ctx.ResponseWriter.Header().Set(ContentType, ContentJSON)
	//ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.SetContentType([]string{ContentJSON + " ;charset=" + Charset})
	ctx.WriteStatus(httpStatus)
	return json.NewEncoder(ctx.ResponseWriter).Encode(jsonObjectOrArray)
}

//JSON calls the WriteJSON with the 200 http status ok
func (ctx *Context) JSON(jsonObjectOrArray interface{}) error {
	return ctx.WriteJSON(http.StatusOK, jsonObjectOrArray)
}

// ReadXML reads XML from request's body
func (ctx Context) ReadXML(xmlObject interface{}) error {
	data, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return err
	}
	defer ctx.Close()

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	err = decoder.Decode(xmlObject)

	if err != io.EOF {
		return err
	}

	return nil
}

//XML calls the WriteXML with the 200 http status ok
func (ctx *Context) XML(xmlBytes []byte) error {
	return ctx.WriteXML(http.StatusOK, xmlBytes)
}

// WriteXML writes xml which from []byte
func (ctx *Context) WriteXML(httpStatus int, xmlB []byte) error {
	ctx.WriteStatus(httpStatus)
	ctx.SetContentType([]string{ContentXML + " ;charset=" + Charset})

	ctx.ResponseWriter.Write(xmlB)
	return nil
	//This is maybe better but it doesn't works as I want, so let it for other func at the future return xml.NewEncoder(ctx.ResponseWriter).Encode(xmlB)
}

// RenderXML writes xml which is converted from struct(s) with a http status which they passed to the function via parameters
func (ctx *Context) RenderXML(httpStatus int, xmlStructs ...interface{}) error {
	var _xmlDoc []byte
	for _, xmlStruct := range xmlStructs {
		theDoc, err := xml.MarshalIndent(xmlStruct, "", "  ")
		if err != nil {
			return err
		}
		_xmlDoc = append(_xmlDoc, theDoc...)
	}
	//ctx.ResponseWriter.Header().Set(ContentType, ContentXML+" ;charset="+Charset)
	//ctx.ResponseWriter.WriteHeader(httpStatus)
	ctx.WriteStatus(httpStatus)
	ctx.SetContentType([]string{ContentXMLText + " ;charset=" + Charset})

	ctx.ResponseWriter.Write(_xmlDoc)
	//xml.NewEncoder(w).Encode(r.Data)
	return nil
}

/* END OF RENDERER */

func (ctx *Context) GetHandlerName() string {
	return runtime.FuncForPC(reflect.ValueOf(ctx.middleware[len(ctx.middleware)-1]).Pointer()).Name()

}
