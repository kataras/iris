package iris

import (
	"gopkg.in/kataras/go-fs.v0"
	"gopkg.in/kataras/go-template.v0"
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

	// PreRender is typeof func(*iris.Context, string, interface{},...map[string]interface{}) bool
	// PreRenders helps developers to pass middleware between
	// the route Handler and a context.Render (THAT can render 'rest' types also but the PreRender applies ONLY to template rendering(file or source)) call
	// all parameter receivers can be changed before passing it to the actual context's Render
	// so, you can change the filenameOrSource, the page binding, the options, and even add cookies, session value or a flash message through ctx
	// the return value of a PreRender is a boolean, if returns false then the next PreRender will not be executed, keep note
	// that the actual context's Render will be called at any case.
	PreRender func(ctx *Context, filenameOrSource string, binding interface{}, options ...map[string]interface{}) bool
)

// templateEngines just a wrapper of template.Mux in order to use it's execute without break the whole of the API
type templateEngines struct {
	*template.Mux
	prerenders []PreRender
}

func newTemplateEngines(sharedFuncs map[string]interface{}) *templateEngines {
	return &templateEngines{Mux: template.NewMux(sharedFuncs)}
}

// getGzipOption receives a default value and the render options map and returns if gzip is enabled for this render action
func getGzipOption(defaultValue bool, options map[string]interface{}) bool {
	gzipOpt := options["gzip"] // we only need that, so don't create new map to keep the options.
	if b, isBool := gzipOpt.(bool); isBool {
		return b
	}
	return defaultValue
}

// gtCharsetOption receives a default value and the render options  map and returns the correct charset for this render action
func getCharsetOption(defaultValue string, options map[string]interface{}) string {
	charsetOpt := options["charset"]
	if s, isString := charsetOpt.(string); isString {
		return s
	}
	return defaultValue
}

func (t *templateEngines) usePreRender(pre PreRender) {
	t.prerenders = append(t.prerenders, pre)
}

// render executes a template and write its result to the context's body
// options are the optional runtime options can be passed by user and catched by the template engine when render
// an example of this is the "layout"
// note that gzip option is an iris dynamic option which exists for all template engines
// the gzip and charset options are built'n with iris
// template is passed as file or souce
func (t *templateEngines) render(isFile bool, ctx *Context, filenameOrSource string, binding interface{}, options []map[string]interface{}) error {
	if ctx.framework.Config.DisableTemplateEngines {
		return errTemplateExecute.Format("Templates are disabled '.Config.DisableTemplatesEngines = true' please turn that to false, as defaulted.")
	}

	if len(t.prerenders) > 0 {
		for i := range t.prerenders {
			// I'm not making any checks here for performance reasons, means that
			// if binding is pointer it can be changed, otherwise not.
			if shouldContinue := t.prerenders[i](ctx, filenameOrSource, binding, options...); !shouldContinue {
				break
			}
		}
	}

	// we do all these because we don't want to initialize a new map for each execution...
	gzipEnabled := ctx.framework.Config.Gzip
	charset := ctx.framework.Config.Charset
	if len(options) > 0 {
		gzipEnabled = getGzipOption(gzipEnabled, options[0])
		charset = getCharsetOption(charset, options[0])
	}

	if isFile {
		ctxLayout := ctx.GetString(TemplateLayoutContextKey)
		if ctxLayout != "" {
			if len(options) > 0 {
				options[0]["layout"] = ctxLayout
			} else {
				options = []map[string]interface{}{{"layout": ctxLayout}}
			}
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

	if isFile {
		return t.ExecuteWriter(out, filenameOrSource, binding, options...)
	}
	return t.ExecuteRaw(filenameOrSource, out, binding)

}

func (t *templateEngines) renderFile(ctx *Context, filename string, binding interface{}, options ...map[string]interface{}) error {
	return t.render(true, ctx, filename, binding, options)
}

func (t *templateEngines) renderSource(ctx *Context, src string, binding interface{}, options ...map[string]interface{}) error {
	return t.render(false, ctx, src, binding, options)
}
