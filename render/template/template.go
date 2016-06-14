package template

import (
	"fmt"
	"io"

	"github.com/klauspost/compress/gzip"

	"sync"

	"github.com/kataras/iris/config"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/render/template/engine/amber"
	"github.com/kataras/iris/render/template/engine/html"
	"github.com/kataras/iris/render/template/engine/jade"
	"github.com/kataras/iris/render/template/engine/markdown"
	"github.com/kataras/iris/render/template/engine/pongo"
	"github.com/kataras/iris/utils"
)

type (
	// Engine the interface that all template engines must inheritance
	Engine interface {
		// BuildTemplates builds the templates for a directory
		BuildTemplates() error
		// ExecuteWriter finds and execute a template and write its result to the out writer
		ExecuteWriter(out io.Writer, name string, binding interface{}, layout string) error
	}

	// Template the internal configs for the common configs for the template engines
	Template struct {
		// Engine the type of the Engine
		Engine Engine
		// Gzip enable gzip compression
		// default is false
		Gzip bool
		// IsDevelopment re-builds the templates on each request
		// default is false
		IsDevelopment bool
		// Directory the system path which the templates live
		// default is ./templates
		Directory string
		// Extensions the allowed file extension
		// default is []string{".html"}
		Extensions []string
		// ContentType is the Content-Type response header
		// default is text/html but you can change if if needed
		ContentType string
		// Layout the template file ( with its extension) which is the mother of all
		// use it to have it as a root file, and include others with {{ yield }}, refer  the docs
		Layout string

		buffer         *utils.BufferPool // this is used only for RenderString
		gzipWriterPool sync.Pool
	}
)

// sharedFuncs the funcs should be exists in all supported view template engines
var sharedFuncs map[string]interface{}

// we do this because we don't want to override the user's funcs
func setSharedFuncs(source map[string]interface{}, target map[string]interface{}) {
	if source == nil {
		return
	}

	if target == nil {
		target = make(map[string]interface{}, len(source))
	}

	for k, v := range source {
		if target[k] == nil {
			target[k] = v
		}
	}
}

// New creates and returns a Template instance which keeps the Template Engine and helps with render
func New(c config.Template) *Template {
	defer func() {
		sharedFuncs = nil
	}()

	var e Engine
	// [ENGINE-2]
	switch c.Engine {
	case config.HTMLEngine:
		setSharedFuncs(sharedFuncs, c.HTMLTemplate.Funcs)
		e = html.New(c) //  HTMLTemplate
	case config.JadeEngine:
		setSharedFuncs(sharedFuncs, c.Jade.Funcs)
		e = jade.New(c) // Jade
	case config.PongoEngine:
		setSharedFuncs(sharedFuncs, c.Pongo.Globals)
		e = pongo.New(c) // Pongo2
	case config.MarkdownEngine:
		e = markdown.New(c) // Markdown
	case config.AmberEngine:
		setSharedFuncs(sharedFuncs, c.Amber.Funcs)
		e = amber.New(c) // Amber
	default: // config.NoEngine
		return nil
	}

	if err := e.BuildTemplates(); err != nil { // first build the templates, if error then panic because this is called before server's run
		panic(err)
	}

	compiledContentType := c.ContentType + "; charset=" + c.Charset

	t := &Template{
		Engine:        e,
		IsDevelopment: c.IsDevelopment,
		Gzip:          c.Gzip,
		ContentType:   compiledContentType,
		Layout:        c.Layout,
		buffer:        utils.NewBufferPool(64),
		gzipWriterPool: sync.Pool{New: func() interface{} {
			return &gzip.Writer{}
		}},
	}

	return t

}

// RegisterSharedFunc registers a functionality that should be inherited from all supported template engines
func RegisterSharedFunc(name string, fn interface{}) {
	if sharedFuncs == nil {
		sharedFuncs = make(map[string]interface{})
	}
	sharedFuncs[name] = fn
}

// RegisterSharedFuncs registers functionalities that should be inherited from all supported template engines
func RegisterSharedFuncs(theFuncs map[string]interface{}) {
	if sharedFuncs == nil || len(sharedFuncs) == 0 {
		sharedFuncs = theFuncs
		return
	}
	for k, v := range theFuncs {
		sharedFuncs[k] = v
	}

}

// Render renders a template using the context's writer
func (t *Template) Render(ctx context.IContext, name string, binding interface{}, layout ...string) (err error) {

	if t == nil { // No engine was given but .Render was called
		ctx.HTML(403, "<b> Iris </b> <br/> Templates are disabled via config.NoEngine, check your iris' configuration please.")
		return fmt.Errorf("[IRIS TEMPLATES] Templates are disabled via config.NoEngine, check your iris' configuration please.\n")
	}

	// build templates again on each render if IsDevelopment.
	if t.IsDevelopment {
		if err = t.Engine.BuildTemplates(); err != nil {
			return
		}
	}

	// I don't like this, something feels wrong
	_layout := ""
	if len(layout) > 0 {
		_layout = layout[0]
	} else if layoutFromCtx := ctx.GetString(config.TemplateLayoutContextKey); layoutFromCtx != "" {
		_layout = layoutFromCtx
	}
	if _layout == "" {
		_layout = t.Layout
	}

	//
	ctx.GetRequestCtx().Response.Header.Set("Content-Type", t.ContentType)

	var out io.Writer
	if t.Gzip {
		ctx.GetRequestCtx().Response.Header.Add("Content-Encoding", "gzip")
		gzipWriter := t.gzipWriterPool.Get().(*gzip.Writer)
		gzipWriter.Reset(ctx.GetRequestCtx().Response.BodyWriter())
		defer gzipWriter.Close()
		defer t.gzipWriterPool.Put(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.GetRequestCtx().Response.BodyWriter()
	}

	err = t.Engine.ExecuteWriter(out, name, binding, _layout)

	return
}

// RenderString executes a template and returns its contents result (string)
func (t *Template) RenderString(name string, binding interface{}, layout ...string) (result string, err error) {

	if t == nil { // No engine was given but .Render was called
		err = fmt.Errorf("[IRIS TEMPLATES] Templates are disabled via config.NoEngine, check your iris' configuration please.\n")
		return
	}

	// build templates again on each render if IsDevelopment.
	if t.IsDevelopment {
		if err = t.Engine.BuildTemplates(); err != nil {
			return
		}
	}

	// I don't like this, something feels wrong
	_layout := ""
	if len(layout) > 0 {
		_layout = layout[0]
	}
	if _layout == "" {
		_layout = t.Layout
	}

	out := t.buffer.Get()
	// if we have problems later consider that -> out.Reset()
	defer t.buffer.Put(out)
	err = t.Engine.ExecuteWriter(out, name, binding, _layout)
	if err == nil {
		result = out.String()
	}
	return
}
