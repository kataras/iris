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

package iris

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	_template "github.com/kataras/iris/template"

	"github.com/kataras/iris/utils"
)

type (
	IContextRenderer interface {
		Write(string, ...interface{})
		WriteHTML(int, string)
		HTML(string)
		WriteData(int, []byte)
		Data([]byte)
		WriteText(int, string)
		Text(string)
		RenderJSON(int, ...interface{}) error
		WriteJSON(int, interface{}) error
		JSON(interface{}) error
		WriteXML(int, []byte) error
		XML([]byte) error
		RenderXML(int, ...interface{}) error

		ExecuteTemplate(*template.Template, interface{}) error
		Render(string, interface{}) error
		RenderNS(namespace string, file string, pageContext interface{}) error
		ServeContent(io.ReadSeeker, string, time.Time) error
		ServeFile(string) error
		SendFile(filename string, destinationName string) error
		Stream(func(*bufio.Writer))
	}
)

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

// ExecuteTemplate executes a simple html template, you can use that if you already have the cached templates
// the recommended way to render is to use iris.Templates("./templates/path/*.html") and ctx.RenderFile("filename.html",struct{})
// accepts 2 parameters
// the first parameter is the template (*template.Template)
// the second parameter is the page context (interfac{})
// returns an error if any errors occurs while executing this template
func (ctx *Context) ExecuteTemplate(tmpl *template.Template, pageContext interface{}) error {
	ctx.RequestCtx.SetContentType(ContentHTML + " ;charset=" + Charset)
	return _template.ErrTemplateExecute.With(tmpl.Execute(ctx.RequestCtx.Response.BodyWriter(), pageContext))
}

// Render  renders a file by its path and a page context passed to the function
func (ctx *Context) Render(file string, pageContext interface{}) error {
	ctx.RequestCtx.SetContentType(ContentHTML + " ;charset=" + Charset)
	return _template.ErrTemplateExecute.With(ctx.station.Templates.Templates.ExecuteTemplate(ctx.RequestCtx.Response.BodyWriter(), file, pageContext))
}

// RenderNS  renders a file by its namespace and path, a page context passed to the function
func (ctx *Context) RenderNS(namespace string, file string, pageContext interface{}) error {
	ctx.RequestCtx.SetContentType(ContentHTML + " ;charset=" + Charset)
	return _template.ErrTemplateExecute.With(ctx.station.Templates.Templates.Lookup(namespace).ExecuteTemplate(ctx.RequestCtx.Response.BodyWriter(), file, pageContext))
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

	ctx.RequestCtx.Response.Header.Set(ContentType, utils.TypeByExtension(filename))
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
