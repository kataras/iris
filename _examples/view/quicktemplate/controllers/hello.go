package controllers

import (
	"github.com/kataras/iris/v12/_examples/view/quicktemplate/templates"

	"github.com/kataras/iris/v12"
)

// Hello renders our ../templates/hello.qtpl file using the compiled ../templates/hello.qtpl.go file.
func Hello(ctx iris.Context) {
	// vars := make(map[string]any)
	// vars["message"] = "Hello World!"
	// vars["name"] = ctx.Params().Get("name")
	// [...]
	// &templates.Hello{ Vars: vars }
	// [...]

	// However, as an alternative, we recommend that you should the `ctx.ViewData(key, value)`
	// in order to be able modify the `templates.Hello#Vars` from a middleware(other handlers) as well.
	ctx.ViewData("message", "Hello World!")
	ctx.ViewData("name", ctx.Params().Get("name"))

	// set view data to the `Vars` template's field
	tmpl := &templates.Hello{
		Vars: ctx.GetViewData(),
	}

	// render the template
	ExecuteTemplate(ctx, tmpl)
}
