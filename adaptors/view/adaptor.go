// Package view is the adaptor of the 5 template engines
// as written by me at https://github.com/kataras/go-template
package view

import (
	"io"
	"strings"

	"github.com/kataras/go-template"
	"gopkg.in/kataras/iris.v6"
)

// Adaptor contains the common actions
// that all template engines share.
//
// We need to export that as it is without an interface
// because it may be used as a wrapper for a template engine
// that is not exists yet but community can create.
type Adaptor struct {
	dir       string
	extension string
	// for a .go template file lives inside the executable
	assetFn func(name string) ([]byte, error)
	namesFn func() []string

	reload bool

	engine template.Engine // used only on Adapt, we could make
	//it as adaptEngine and pass a second parameter there but this would break the pattern.
}

// NewAdaptor returns a new general template engine policy adaptor.
func NewAdaptor(directory string, extension string, e template.Engine) *Adaptor {
	return &Adaptor{
		dir:       directory,
		extension: extension,
		engine:    e,
	}
}

// Binary optionally, use it when template files are distributed
// inside the app executable (.go generated files).
func (h *Adaptor) Binary(assetFn func(name string) ([]byte, error), namesFn func() []string) *Adaptor {
	h.assetFn = assetFn
	h.namesFn = namesFn
	return h
}

// Reload if setted to true the templates are reloading on each call,
// use it when you're in development and you're boring of restarting
// the whole app when you edit a template file
func (h *Adaptor) Reload(developmentMode bool) *Adaptor {
	h.reload = developmentMode
	return h
}

// Adapt adapts a template engine to the main Iris' policies.
// this specific Adapt is a multi-policies adaptors
// we use  that instead of just return New() iris.RenderPolicy
// for two reasons:
// -  the user may need to edit the adaptor's fields
//   like Directory, Binary
// - we need to adapt an event policy to add the engine to the external mux
//   and load it.
func (h *Adaptor) Adapt(frame *iris.Policies) {
	mux := template.DefaultMux
	// on the build state in order to have the shared funcs also
	evt := iris.EventPolicy{
		Build: func(s *iris.Framework) {
			// mux has default to ./templates and .html ext
			// no need for any checks here.
			// the RenderPolicy will give a "no templates found on 'directory'"
			// if user tries to use the context.Render without having the template files
			// no need to panic here because we will use the html as the default template engine
			// even if the user doesn't asks for
			// or no? we had the defaults before... maybe better to give the user
			// the opportunity to learn about the template's configuration
			// (now 6.1.4 ) they are defaulted and users may don't know how and if they can change the configuration
			// even if the book and examples covers these things, many times they're asking me on chat.............
			// so no defaults? ok no defaults. This file then will be saved to /adaptors as with other template engines.
			// simple.
			mux.AddEngine(h.engine).
				Directory(h.dir, h.extension).
				Binary(h.assetFn, h.namesFn)

			mux.Reload = h.reload

			// notes for me: per-template engine funcs are setted by each template engine adaptor itself,
			// here we will set the template funcs policy'.
			// as I explain on the TemplateFuncsPolicy it exists in order to allow community to create plugins
			// even by adding custom template funcs to their behaviors.

			// We know that iris.TemplateFuncsPolicy is useless without this specific
			// adaptor. We also know that it is not a good idea to have two
			// policies with the same function or we can? wait. No we can't.
			// We can't because:
			// - the RenderPolicy should accept any type of render process, not only templates.
			// - I didn't design iris/policy.go to keep logic about implementation, this would make that very limited
			//    and I don't want to break that just for the templates.
			// - We want community to be able to create packages which can adapt template functions but
			//   not to worry about the rest of the template engine adaptor policy.
			//   And even don't worry if the user has registered or use a template engine at all.
			// So we should keep separate the TemplateFuncsPolicy(just map[string]interface{})
			//                            from the rest of the implementation.
			//
			// So when the community wants to create a template adaptor has two options:
			// - Use the RenderPolicy which is just a func
			// - Use the kataras/iris/adaptors/view.Adaptor adaptor wrapper for RenderPolicy with a combination of kataras/go-template/.Engine
			//
			//
			// So here is the only place we adapt the iris.TemplateFuncsPolicy to the templates, if and only if templates are used,
			//  otherwise they are just ignored without any creepy things.
			//
			// The TemplateFuncsPolicy will work in combination with the specific template adaptor's functions(see html.go and the rest)

			if len(frame.TemplateFuncsPolicy) > 0 {
				mux.SetFuncMapToEngine(frame.TemplateFuncsPolicy, h.engine)
			}

			if err := mux.Load(); err != nil {
				s.Log(iris.ProdMode, err.Error())
			}
		},
	}
	// adapt the build event to the main policies
	evt.Adapt(frame)

	r := iris.RenderPolicy(func(out io.Writer, file string, tmplContext interface{}, options ...map[string]interface{}) (bool, error) {
		// template mux covers that but maybe we have more than one RenderPolicy
		// and each of them carries a different mux on the new design.
		if strings.Contains(file, h.extension) {
			return true, mux.ExecuteWriter(out, file, tmplContext, options...)
		}
		return false, nil
	})

	r.Adapt(frame)
}
