package iris

import (
	"github.com/kataras/go-fs"
	"github.com/kataras/go-template"
	"io"
)

var (
	builtinFuncs = [...]string{"url", "urlpath"}
)

const (
	// NoLayout to disable layout for a particular template file
	NoLayout = template.NoLayout
	// TemplateLayoutContextKey is the name of the user values which can be used to set a template layout from a middleware and override the parent's
	TemplateLayoutContextKey = "templateLayout"
)

type (
	// RenderOptions is a helper type for  the optional runtime options can be passed by user when Render
	// an example of this is the "layout" or "gzip" option
	// same as Map but more specific name
	RenderOptions map[string]interface{}
)

// templateEngines just a wrapper of template.Mux in order to use it's execute without break the whole of the API
type templateEngines struct {
	*template.Mux
}

func newTemplateEngines(sharedFuncs map[string]interface{}) *templateEngines {
	return &templateEngines{Mux: template.NewMux(sharedFuncs)}
}

// render executes a template and write its result to the context's body
// options are the optional runtime options can be passed by user and catched by the template engine when render
// an example of this is the "layout"
// note that gzip option is an iris dynamic option which exists for all template engines
// the gzip and charset options are built'n with iris
func (t *templateEngines) render(ctx *Context, filename string, binding interface{}, options ...map[string]interface{}) (err error) {
	// we do all these because we don't want to initialize a new map for each execution...
	gzipEnabled := ctx.framework.Config.Gzip
	charset := ctx.framework.Config.Charset
	if len(options) > 0 {
		gzipEnabled = template.GetGzipOption(gzipEnabled, options[0])
		charset = template.GetCharsetOption(charset, options[0])
	}

	ctxLayout := ctx.GetString(TemplateLayoutContextKey)
	if ctxLayout != "" {
		if len(options) > 0 {
			options[0]["layout"] = ctxLayout
		} else {
			options = []map[string]interface{}{map[string]interface{}{"layout": ctxLayout}}
		}
	}

	ctx.SetContentType(contentHTML + "; charset=" + charset)

	var out io.Writer
	if gzipEnabled && ctx.clientAllowsGzip() {
		ctx.RequestCtx.Response.Header.Add(varyHeader, acceptEncodingHeader)
		ctx.SetHeader(contentEncodingHeader, "gzip")

		gzipWriter := fs.AcquireGzipWriter(ctx.Response.BodyWriter())
		defer fs.ReleaseGzipWriter(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.Response.BodyWriter()
	}

	err = t.ExecuteWriter(out, filename, binding, options...)
	return err
}

// renderSource executes a template source raw contents (string) and write its result to the context's body
// note that gzip option is an iris dynamic option which exists for all template engines
// the gzip and charset options are built'n with iris
func (t *templateEngines) renderSource(ctx *Context, src string, binding interface{}, options ...map[string]interface{}) (err error) {
	// we do all these because we don't want to initialize a new map for each execution...
	gzipEnabled := ctx.framework.Config.Gzip
	charset := ctx.framework.Config.Charset
	if len(options) > 0 {
		gzipEnabled = template.GetGzipOption(gzipEnabled, options[0])
		charset = template.GetCharsetOption(charset, options[0])
	}

	ctx.SetContentType(contentHTML + "; charset=" + charset)

	var out io.Writer
	if gzipEnabled && ctx.clientAllowsGzip() {
		ctx.RequestCtx.Response.Header.Add(varyHeader, acceptEncodingHeader)
		ctx.SetHeader(contentEncodingHeader, "gzip")

		gzipWriter := fs.AcquireGzipWriter(ctx.Response.BodyWriter())
		defer fs.ReleaseGzipWriter(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.Response.BodyWriter()
	}
	return t.ExecuteRaw(src, out, binding)
}
