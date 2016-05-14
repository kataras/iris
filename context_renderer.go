package iris

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"time"

	"github.com/kataras/iris/utils"
)

// Write writes a string via the context's ResponseWriter
func (ctx *Context) Write(format string, a ...interface{}) {
	//this doesn't work with gzip, so just write the []byte better |ctx.ResponseWriter.WriteString(fmt.Sprintf(format, a...))
	ctx.RequestCtx.WriteString(fmt.Sprintf(format, a...))
}

// WriteHTML writes html string with a http status
func (ctx *Context) WriteHTML(httpStatus int, htmlContents string) {
	ctx.SetContentType([]string{ContentHTML + " ;charset=" + DefaultCharset})
	ctx.RequestCtx.SetStatusCode(httpStatus)
	ctx.RequestCtx.WriteString(htmlContents)
}

// Data writes out the raw bytes as binary data.
func (ctx *Context) Data(status int, v []byte) error {
	return ctx.station.rest.Data(ctx.RequestCtx, status, v)
}

// HTML builds up the response from the specified template and bindings.
// Note: parameter layout has meaning only when using the iris.StandarEngine
func (ctx *Context) HTML(status int, name string, binding interface{}, layout ...string) error {
	ctx.SetStatusCode(status)
	return ctx.station.templates.Render(ctx, name, binding, layout...)
}

// Render same as .HTML but with status to iris.StatusOK (200)
func (ctx *Context) Render(name string, binding interface{}, layout ...string) error {
	return ctx.HTML(StatusOK, name, binding, layout...)
}

// JSON marshals the given interface object and writes the JSON response.
func (ctx *Context) JSON(status int, v interface{}) error {
	return ctx.station.rest.JSON(ctx.RequestCtx, status, v)
}

// JSONP marshals the given interface object and writes the JSON response.
func (ctx *Context) JSONP(status int, callback string, v interface{}) error {
	return ctx.station.rest.JSONP(ctx.RequestCtx, status, callback, v)
}

// Text writes out a string as plain text.
func (ctx *Context) Text(status int, v string) error {
	return ctx.station.rest.Text(ctx.RequestCtx, status, v)
}

// XML marshals the given interface object and writes the XML response.
func (ctx *Context) XML(status int, v interface{}) error {
	return ctx.station.rest.XML(ctx.RequestCtx, status, v)
}

// ExecuteTemplate executes a simple html template, you can use that if you already have the cached templates
// the recommended way to render is to use iris.Templates("./templates/path/*.html") and ctx.RenderFile("filename.html",struct{})
// accepts 2 parameters
// the first parameter is the template (*template.Template)
// the second parameter is the page context (interfac{})
// returns an error if any errors occurs while executing this template
func (ctx *Context) ExecuteTemplate(tmpl *template.Template, pageContext interface{}) error {
	ctx.RequestCtx.SetContentType(ContentHTML + " ;charset=" + DefaultCharset)
	return ErrTemplateExecute.With(tmpl.Execute(ctx.RequestCtx.Response.BodyWriter(), pageContext))
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
