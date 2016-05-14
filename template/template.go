package template

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/template/engine"
	"github.com/kataras/iris/template/engine/pongo"
	"github.com/kataras/iris/template/engine/standar"
)

type (
	Template struct {
		Engine engine.Engine

		IsDevelopment bool
		Gzip          bool
		ContentType   string
		Layout        string
	}

	// TemplateOptions the options to create a Template instance
	//
	// Options and no Config because this struct is not live inside a Template instance
	TemplateOptions struct {
		Engine        engine.EngineType
		engine.Config // contains common configs for both standar & pongo
		// [ENGINE-1]
		Standar standar.StandarConfig // contains specific configs for standar html/template
		Pongo   pongo.PongoConfig     // contains specific configs for pongo2
	}
)

func New(opt TemplateOptions) *Template {

	var e engine.Engine
	// [ENGINE-2]
	switch opt.Engine {
	case engine.Pongo:
		e = pongo.New(pongo.WrapConfig(opt.Config, opt.Pongo))
	default:
		e = standar.New(standar.WrapConfig(opt.Config, opt.Standar)) // default to standar
	}

	if err := e.BuildTemplates(); err != nil { // first build the templates, if error panic because this is called before server's run
		panic(err)
	}

	compiledContentType := opt.ContentType + "; charset=" + opt.Charset

	return &Template{
		Engine:        e,
		IsDevelopment: opt.IsDevelopment,
		Gzip:          opt.Gzip,
		ContentType:   compiledContentType,
		Layout:        opt.Layout,
	}

}

func (t *Template) Render(ctx context.IContext, name string, bindings interface{}, layout ...string) error {
	// build templates again on each render if IsDevelopment.
	if t.IsDevelopment {
		if err := t.Engine.BuildTemplates(); err != nil {
			return err
		}
	}
	ctx.GetRequestCtx().Response.Header.Set("Content-Type", t.ContentType)
	// I don't like this, something feels wrong
	_layout := ""
	if len(layout) > 0 {
		_layout = layout[0]
	}
	if _layout == "" {
		_layout = t.Layout
	}

	//

	if t.Gzip {
		return t.Engine.ExecuteGzip(ctx, name, bindings, _layout)
	}

	return t.Engine.Execute(ctx, name, bindings, _layout)

}
