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
	"github.com/klauspost/compress/gzip"
)

// Write writes a string via the context's ResponseWriter
func (ctx *Context) Write(format string, a ...interface{}) {
	//this doesn't work with gzip, so just write the []byte better |ctx.ResponseWriter.WriteString(fmt.Sprintf(format, a...))
	ctx.RequestCtx.WriteString(fmt.Sprintf(format, a...))
}

// WriteHTML writes html string with a http status
func (ctx *Context) WriteHTML(httpStatus int, htmlContents string) {
	ctx.SetContentType(ContentHTML + ctx.station.rest.CompiledCharset)
	ctx.RequestCtx.SetStatusCode(httpStatus)
	ctx.RequestCtx.WriteString(htmlContents)
}

// Data writes out the raw bytes as binary data.
func (ctx *Context) Data(status int, v []byte) error {
	return ctx.station.rest.Data(ctx.RequestCtx, status, v)
}

// HTML builds up the response from the specified template and bindings.
// Note: parameter layout has meaning only when using the iris.HTMLTemplate
func (ctx *Context) HTML(status int, name string, binding interface{}, layout ...string) error {
	ctx.SetStatusCode(status)
	return ctx.station.templates.Render(ctx, name, binding, layout...)
}

// Render same as .HTML but with status to iris.StatusOK (200)
func (ctx *Context) Render(name string, binding interface{}, layout ...string) error {
	return ctx.HTML(StatusOK, name, binding, layout...)
}

// RenderStrings accepts a template filename, its context data and returns the result of the parsed template (string)
func (ctx *Context) RenderString(name string, binding interface{}, layout ...string) (result string, err error) {
	return ctx.station.templates.RenderString(name, binding, layout...)
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

// MarkdownString parses the (dynamic) markdown string and returns the converted html string
func (ctx *Context) MarkdownString(markdown string) string {
	return ctx.station.rest.Markdown([]byte(markdown))
}

// Markdown parses and renders to the client a particular (dynamic) markdown string
// accepts two parameters
// first is the http status code
// second is the markdown string
func (ctx *Context) Markdown(status int, markdown string) {
	ctx.WriteHTML(status, ctx.MarkdownString(markdown))
}

// ExecuteTemplate executes a simple html template, you can use that if you already have the cached templates
// the recommended way to render is to use iris.Templates("./templates/path/*.html") and ctx.RenderFile("filename.html",struct{})
// accepts 2 parameters
// the first parameter is the template (*template.Template)
// the second parameter is the page context (interfac{})
// returns an error if any errors occurs while executing this template
func (ctx *Context) ExecuteTemplate(tmpl *template.Template, pageContext interface{}) error {
	ctx.RequestCtx.SetContentType(ContentHTML + ctx.station.rest.CompiledCharset)
	return ErrTemplateExecute.With(tmpl.Execute(ctx.RequestCtx.Response.BodyWriter(), pageContext))
}

// ServeContent serves content, headers are autoset
// receives three parameters, it's low-level function, instead you can use .ServeFile(string)
//
// You can define your own "Content-Type" header also, after this function call
func (ctx *Context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) (err error) {
	if t, err := time.Parse(TimeFormat, ctx.RequestHeader(IfModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		ctx.RequestCtx.Response.Header.Del(ContentType)
		ctx.RequestCtx.Response.Header.Del(ContentLength)
		ctx.RequestCtx.SetStatusCode(StatusNotModified)
		return nil
	}

	ctx.RequestCtx.Response.Header.Set(ContentType, utils.TypeByExtension(filename))
	ctx.RequestCtx.Response.Header.Set(LastModified, modtime.UTC().Format(TimeFormat))
	ctx.RequestCtx.SetStatusCode(StatusOK)
	var out io.Writer
	if gzipCompression {
		ctx.RequestCtx.Response.Header.Add("Content-Encoding", "gzip")
		gzipWriter := ctx.station.gzipWriterPool.Get().(*gzip.Writer)
		gzipWriter.Reset(ctx.RequestCtx.Response.BodyWriter())
		defer gzipWriter.Close()
		defer ctx.station.gzipWriterPool.Put(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.RequestCtx.Response.BodyWriter()

	}
	_, err = io.Copy(out, content)
	return ErrServeContent.With(err)
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

	ctx.RequestCtx.Response.Header.Set(ContentDisposition, "attachment;filename="+destinationName)
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
