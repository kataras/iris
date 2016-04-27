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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Third party package "github.com/monoculum/formam" is protected by the Apache License

package iris

import (
	"bufio"
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
	"time"

	"github.com/monoculum/formam"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
)

// Charset is defaulted to UTF-8, you can change it
// all render methods will have this charset
var Charset = DefaultCharset

type (

	// IContext is the interface for the iris/Context
	// it's implements the /x/net/context, the fasthttp's RequestCtx
	// and some other useful methods
	IContext interface {
		context.Context
		Reset(*fasthttp.RequestCtx)
		Next()
		Param(string) string
		ParamInt(string) (int, error)
		URLParam(string) string
		URLParamInt(string) (int, error)
		URLParams() map[string][]string
		MethodString() string
		HostString() string
		PathString() string
		Get(interface{}) interface{}
		GetString(interface{}) string
		GetInt(interface{}) int
		Set(interface{}, interface{})
		Write(string, ...interface{})
		GetCookie(string) string
		SetCookie(string, string)
		AddCookie(*fasthttp.Cookie)
		// Errors
		NotFound()
		Panic()
		EmitError(int)
		StopExecution()
		//

		Redirect(string, ...int)
		RequestIP() string
		IsStopped() bool
		Clone() *Context
		ExecuteTemplate(*template.Template, interface{}) error
		RenderFile(string, interface{}) error
		// SetContentType sets the "Content-Type" header, receives the values
		SetContentType([]string)
		// SetHeader sets the response headers first parameter is the key, second is the values
		SetHeader(string, []string)
		RequestHeader(k string) string
		//
		WriteHTML(int, string)
		HTML(string)
		WriteData(int, []byte)
		Data([]byte)
		WriteText(int, string)
		Text(string)
		RenderJSON(int, ...interface{}) error
		ReadJSON(interface{}) error
		WriteJSON(int, interface{}) error
		JSON(interface{}) error
		WriteXML(int, []byte) error
		XML([]byte) error
		RenderXML(int, ...interface{}) error
		ReadXML(interface{}) error
		ReadForm(formObject interface{}) error
		ServeContent(io.ReadSeeker, string, time.Time) error
		ServeFile(string) error
		SendFile(filename string, destinationName string) error
		Stream(func(*bufio.Writer))
		//
		PostFormValue(string) string
		//
		GetHandlerName() string
	}

	// Context is resetting every time a request is coming to the server
	// it is not good practice to use this object in goroutines, for these cases use the .Clone()
	Context struct {
		*fasthttp.RequestCtx
		Params  PathParameters
		station *Station
		//keep track all registed middleware (handlers)
		middleware Middleware
		// pos is the position number of the Context, look .Next to understand
		pos uint8
		// these values are reseting on each request, are useful only between middleware,
		// use iris/sessions for cookie/filesystem storage
		values map[interface{}]interface{}
	}
)

var _ IContext = &Context{}

// Implement the golang.org/x/net/context , as requested by the community, which is used inside app engine
// also this will give me the ability to use appengine's memcache with this context, if this needed.

// Deadline returns the time when this Context will be canceled, if any.
func (ctx *Context) Deadline() (deadline time.Time, ok bool) {
	return
}

// Done returns a channel that is closed when this Context is canceled
// or times out.
func (ctx *Context) Done() <-chan struct{} {
	return nil
}

// Err indicates why this context was canceled, after the Done channel
// is closed.
func (ctx *Context) Err() error {
	return nil
}

// Value returns the value associated with key or nil if none.
func (ctx *Context) Value(key interface{}) interface{} {
	if key == 0 {
		return ctx.Request
	}
	if keyAsString, ok := key.(string); ok {
		val := ctx.GetString(keyAsString)
		return val
	}
	return nil
}

//
// For PathParameters note:
//  here we return their values as pointer, so be careful

// Param returns the string representation of the key's path named parameter's value
func (ctx *Context) Param(key string) string {
	return ctx.Params.Get(key)
}

// ParamInt returns the int representation of the key's path named parameter's value
func (ctx *Context) ParamInt(key string) (int, error) {
	val, err := strconv.Atoi(ctx.Param(key))
	return val, err
}

// URLParam returns the get parameter from a request , if any
func (ctx *Context) URLParam(key string) string {
	return string(ctx.RequestCtx.Request.URI().QueryArgs().Peek(key))
}

// URLParams returns a map of a list of each url(query) parameter
func (ctx *Context) URLParams() map[string][]string {
	urlparams := make(map[string][]string)
	ctx.RequestCtx.Request.URI().QueryArgs().VisitAll(func(key, value []byte) {
		urlparams[string(key)] = []string{string(value)}
	})
	return urlparams
}

// URLParamInt returns the get parameter int value from a request , if any
func (ctx *Context) URLParamInt(key string) (int, error) {
	return strconv.Atoi(ctx.URLParam(key))
}

// MethodString returns the HTTP Method
func (ctx *Context) MethodString() string {
	return BytesToString(ctx.Method())
}

// HostString returns the Host of the request( the url as string )
func (ctx *Context) HostString() string {
	return BytesToString(ctx.Host())
}

// PathString returns the full path as string
func (ctx *Context) PathString() string {
	return BytesToString(ctx.Path())
}

// GetCookie returns cookie's value by it's name
func (ctx *Context) GetCookie(name string) string {
	return string(ctx.RequestCtx.Request.Header.Cookie(name))
}

// SetCookie adds a cookie to the request
func (ctx *Context) SetCookie(name string, value string) {
	ctx.RequestCtx.Request.Header.SetCookie(name, value)
}

// AddCookie sets a specific cookie to the response header
func (ctx *Context) AddCookie(cookie *fasthttp.Cookie) {
	s := fmt.Sprintf("%s=%s", string(cookie.Key()), string(cookie.Value()))
	if c := string(ctx.RequestCtx.Request.Header.Peek("Cookie")); c != "" {
		ctx.RequestCtx.Request.Header.Set("Cookie", c+"; "+s)
	} else {
		ctx.RequestCtx.Request.Header.Set("Cookie", s)
	}
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

// StopExecution just sets the .pos to 255 in order to  not move to the next middlewares(if any)
func (ctx *Context) StopExecution() {
	ctx.pos = stopExecutionPosition
}

//

// Redirect redirect sends a redirect response the client
// accepts 2 parameters string and an optional int
// first parameter is the url to redirect
// second parameter is the http status should send, default is 302 (Temporary redirect), you can set it to 301 (Permant redirect), if that's nessecery
func (ctx *Context) Redirect(urlToRedirect string, statusHeader ...int) {
	httpStatus := 302 // temporary redirect
	if statusHeader != nil && len(statusHeader) > 0 && statusHeader[0] > 0 {
		httpStatus = statusHeader[0]
	}

	ctx.RequestCtx.Redirect(urlToRedirect, httpStatus)
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

// IsStopped checks and returns true if the current position of the Context is 255, means that the StopExecution has called
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

// Do calls the first handler only, it's like Next with negative pos, used only on Router&MemoryRouter
func (ctx *Context) Do() {
	ctx.pos = 0
	ctx.middleware[0].Serve(ctx)
}

// Reset resets the Context with a given domain.Response and domain.Request
// the context is ready-to-use after that, just like a new Context
// I use it for zero rellocation memory
func (ctx *Context) Reset(reqCtx *fasthttp.RequestCtx) {
	ctx.Params = ctx.Params[0:0]
	ctx.middleware = nil
	ctx.RequestCtx = reqCtx
}

// Clone use that method if you want to use the context inside a goroutine
func (ctx *Context) Clone() *Context {
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
	return &cloneContext
}

// Get returns a value from a key
// if doesn't exists returns nil
func (ctx *Context) Get(key interface{}) interface{} {
	if ctx.values == nil {
		return nil
	}

	return ctx.values[key]
}

// GetFmt returns a value which has this format: func(format string, args ...interface{}) string
// if doesn't exists returns nil
func (ctx *Context) GetFmt(key interface{}) func(format string, args ...interface{}) string {
	if ctx.values == nil {
		return nil
	}

	return ctx.values[key].(func(format string, args ...interface{}) string)
}

// GetString same as Get but returns the value as string
func (ctx *Context) GetString(key interface{}) (value string) {
	if v := ctx.Get(key); v != nil {
		value = v.(string)
	}

	return
}

// GetInt same as Get but returns the value as int
func (ctx *Context) GetInt(key interface{}) (value int) {
	if v := ctx.Get(key); v != nil {
		value = v.(int)
	}

	return
}

// Set sets a value to a key in the values map
func (ctx *Context) Set(key interface{}, value interface{}) {
	if ctx.values == nil {
		ctx.values = make(map[interface{}]interface{})
	}

	ctx.values[key] = value
}

// SetContentType sets the response writer's header key 'Content-Type' to a given value(s)
func (ctx *Context) SetContentType(s []string) {
	for _, hv := range s {
		ctx.RequestCtx.Response.Header.Set(ContentType, hv)
	}

}

// SetHeader write to the response writer's header to a given key the given value(s)
func (ctx *Context) SetHeader(k string, s []string) {
	for _, hv := range s {
		ctx.RequestCtx.Response.Header.Set(k, hv)
	}
}

// RequestHeader returns the request header's value
// accepts one parameter, the key of the header (string)
// returns string
func (ctx *Context) RequestHeader(k string) string {
	return BytesToString(ctx.RequestCtx.Request.Header.Peek(k))
}

/* RENDERER */

// ExecuteTemplate executes a simple html template, you can use that if you already have the cached templates
// the recommended way to render is to use iris.Templates("./templates/path/*.html") and ctx.RenderFile("filename.html",struct{})
// accepts 2 parameters
// the first parameter is the template (*template.Template)
// the second parameter is the page context (interfac{})
// returns an error if any errors occurs while executing this template
func (ctx *Context) ExecuteTemplate(tmpl *template.Template, pageContext interface{}) error {
	ctx.RequestCtx.SetContentType(ContentHTML + " ;charset=" + Charset)
	return ErrTemplateExecute.With(tmpl.Execute(ctx.RequestCtx.Response.BodyWriter(), pageContext))
}

// RenderFile renders a file by its path and a context passed to the function
func (ctx *Context) RenderFile(file string, pageContext interface{}) error {
	ctx.RequestCtx.SetContentType(ContentHTML + " ;charset=" + Charset)
	return ErrTemplateExecute.With(ctx.station.GetTemplates().Templates.ExecuteTemplate(ctx.RequestCtx.Response.BodyWriter(), file, pageContext))

}

// WriteHTML writes html string with a http status
func (ctx *Context) WriteHTML(httpStatus int, htmlContents string) {
	ctx.SetContentType([]string{ContentHTML + " ;charset=" + Charset})
	ctx.RequestCtx.SetStatusCode(httpStatus)
	ctx.RequestCtx.WriteString(htmlContents)
}

//HTML calls the WriteHTML with the 200 http status ok
func (ctx *Context) HTML(htmlContents string) {
	ctx.WriteHTML(StatusOK, htmlContents)
}

// WriteData writes binary data with a http status
func (ctx *Context) WriteData(httpStatus int, binaryData []byte) {
	ctx.SetHeader(ContentType, []string{ContentBINARY + " ;charset=" + Charset})
	ctx.SetHeader(ContentLength, []string{strconv.Itoa(len(binaryData))})
	ctx.RequestCtx.SetStatusCode(httpStatus)
	ctx.RequestCtx.Write(binaryData)
}

//Data calls the WriteData with the 200 http status ok
func (ctx *Context) Data(binaryData []byte) {
	ctx.WriteData(StatusOK, binaryData)
}

// Write writes a string via the context's ResponseWriter
func (ctx *Context) Write(format string, a ...interface{}) {
	//this doesn't work with gzip, so just write the []byte better |ctx.ResponseWriter.WriteString(fmt.Sprintf(format, a...))
	ctx.RequestCtx.WriteString(fmt.Sprintf(format, a...))
}

// WriteText writes text with a http status
func (ctx *Context) WriteText(httpStatus int, text string) {
	ctx.SetContentType([]string{ContentTEXT + " ;charset=" + Charset})
	ctx.RequestCtx.SetStatusCode(httpStatus)

	ctx.RequestCtx.Write([]byte(text))
}

//Text calls the WriteText with the 200 http status ok
func (ctx *Context) Text(text string) {
	ctx.WriteText(StatusOK, text)
}

// WriteJSON writes JSON which is encoded from a single json object or array with no Indent
func (ctx *Context) WriteJSON(httpStatus int, jsonObjectOrArray interface{}) error {
	ctx.SetContentType([]string{ContentJSON + " ;charset=" + Charset})
	ctx.RequestCtx.SetStatusCode(httpStatus)
	return ErrWriteJSON.With(json.NewEncoder(ctx.Response.BodyWriter()).Encode(jsonObjectOrArray))
}

//JSON calls the WriteJSON with the 200 http status ok if no previous status code setted
func (ctx *Context) JSON(jsonObjectOrArray interface{}) error {
	statusCode := ctx.Response.StatusCode()
	if statusCode <= 0 {
		statusCode = StatusOK
	}
	return ctx.WriteJSON(statusCode, jsonObjectOrArray)
}

// RenderJSON renders json objects with indent
func (ctx *Context) RenderJSON(httpStatus int, jsonStructs ...interface{}) error {
	var _json []byte

	for _, jsonStruct := range jsonStructs {

		theJSON, err := json.MarshalIndent(jsonStruct, "", "  ")
		if err != nil {
			return ErrRenderMarshalled.Format("JSON", err.Error())
		}
		_json = append(_json, theJSON...)
	}

	ctx.SetContentType([]string{ContentJSON + " ;charset=" + Charset})
	ctx.RequestCtx.SetStatusCode(httpStatus)

	ctx.RequestCtx.Write(_json)

	return nil
}

// ReadJSON reads JSON from request's body
func (ctx *Context) ReadJSON(jsonObject interface{}) error {

	data := ctx.RequestCtx.Request.Body()

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	err := decoder.Decode(jsonObject)

	//err != nil fix by @shiena
	if err != nil && err != io.EOF {
		return ErrReadBody.Format("JSON", err.Error())
	}

	return nil
}

// WriteXML writes xml which from []byte
func (ctx *Context) WriteXML(httpStatus int, xmlB []byte) error {
	ctx.RequestCtx.SetStatusCode(httpStatus)
	ctx.SetContentType([]string{ContentXML + " ;charset=" + Charset})

	ctx.RequestCtx.Write(xmlB)
	return nil
}

//XML calls the WriteXML with the 200 http status ok if no previous status setted
func (ctx *Context) XML(xmlBytes []byte) error {
	statusCode := ctx.Response.StatusCode()
	if statusCode <= 0 {
		statusCode = StatusOK
	}
	return ctx.WriteXML(statusCode, xmlBytes)
}

// ReadXML reads XML from request's body
func (ctx *Context) ReadXML(xmlObject interface{}) error {
	data := ctx.RequestCtx.Request.Body()

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	err := decoder.Decode(xmlObject)
	//err != nil fix by @shiena
	if err != nil && err != io.EOF {
		return ErrReadBody.Format("XML", err.Error())
	}

	return nil
}

// RenderXML writes xml which is converted from struct(s) with a http status which they passed to the function via parameters
func (ctx *Context) RenderXML(httpStatus int, xmlStructs ...interface{}) error {
	var _xmlDoc []byte
	for _, xmlStruct := range xmlStructs {
		theDoc, err := xml.MarshalIndent(xmlStruct, "", "  ")
		if err != nil {
			return ErrRenderMarshalled.Format("XML", err.Error())
		}
		_xmlDoc = append(_xmlDoc, theDoc...)
	}
	ctx.RequestCtx.SetStatusCode(httpStatus)
	ctx.SetContentType([]string{ContentXMLText + " ;charset=" + Charset})

	ctx.RequestCtx.Write(_xmlDoc)
	return nil
}

// ReadForm binds the formObject  with the form data
// it supports any kind of struct
func (ctx *Context) ReadForm(formObject interface{}) error {

	// first check if we have multipart form
	form, err := ctx.RequestCtx.MultipartForm()
	if err == nil {
		//we have multipart form

		return ErrReadBody.With(formam.Decode(form.Value, formObject))
	}
	// if no multipart and post arguments ( means normal form)
	if ctx.RequestCtx.PostArgs().Len() > 0 {
		form := make(map[string][]string, ctx.RequestCtx.PostArgs().Len()+ctx.RequestCtx.QueryArgs().Len())
		ctx.RequestCtx.PostArgs().VisitAll(func(k []byte, v []byte) {
			form[BytesToString(k)] = []string{BytesToString(v)}
		})
		ctx.RequestCtx.QueryArgs().VisitAll(func(k []byte, v []byte) {
			form[BytesToString(k)] = []string{BytesToString(v)}
		})

		return ErrReadBody.With(formam.Decode(form, formObject))
	}

	return ErrReadBody.With(ErrNoForm.Return())
}

// ServeContent serves content, headers are autoset
// receives three parameters, it's low-level function, instead you can use .ServeFile(string)
//
// You can define your own "Content-Type" header also, after this function call
func (ctx *Context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time) error {
	if t, err := time.Parse(TimeFormat, ctx.RequestHeader(IfModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.RequestCtx.Response.Header.Del(ContentType)
		ctx.RequestCtx.Response.Header.Del(ContentLength)
		ctx.RequestCtx.SetStatusCode(304) //NotModified
		return nil
	}

	ctx.RequestCtx.Response.Header.Set(ContentType, TypeByExtension(filename))
	ctx.RequestCtx.Response.Header.Set(LastModified, modtime.UTC().Format(TimeFormat))
	ctx.RequestCtx.SetStatusCode(200)
	_, err := io.Copy(ctx.RequestCtx.Response.BodyWriter(), content)
	return ErrServeContent.With(err)
}

// ServeFile serves a view file, to send a file ( zip for example) to the client you should use the SendFile(serverfilename,clientfilename)
// receives one parameter
// filename (string)
//
// You can define your own "Content-Type" header also, after this function call
func (ctx *Context) ServeFile(filename string) error {
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
	return ctx.ServeContent(f, fi.Name(), fi.ModTime())
}

// SendFile sends file for force-download to the client
//
// You can define your own "Content-Type" header also, after this function call
// for example: ctx.Response.Header.Set("Content-Type","thecontent/type")
func (ctx *Context) SendFile(filename string, destinationName string) error {
	err := ctx.ServeFile(filename)
	if err != nil {
		return err
	}

	ctx.RequestCtx.Response.Header.Set(ContentDisposition, "attachment;filename="+destinationName)
	return nil
}

// Stream use that to do data steaming
func (ctx *Context) Stream(cb func(writer *bufio.Writer)) {
	ctx.RequestCtx.SetBodyStreamWriter(cb)
}

/* END OF RENDERER */

// PostFormValue returns a single value from post request's data
func (ctx *Context) PostFormValue(name string) string {
	return string(ctx.RequestCtx.PostArgs().Peek(name))
}

// GetHandlerName as requested returns the stack-name of the function which the Middleware is setted from
func (ctx *Context) GetHandlerName() string {
	return runtime.FuncForPC(reflect.ValueOf(ctx.middleware[len(ctx.middleware)-1]).Pointer()).Name()
}
