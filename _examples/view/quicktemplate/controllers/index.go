package controllers

import (
	"github.com/kataras/iris/v12/_examples/view/quicktemplate/templates"

	"github.com/kataras/iris/v12"
)

// Index renders our ../templates/index.qtpl file using the compiled ../templates/index.qtpl.go file.
func Index(ctx iris.Context) {
	tmpl := &templates.Index{}

	// render the template
	ExecuteTemplate(ctx, tmpl)
}
