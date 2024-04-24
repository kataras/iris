package controllers

import (
	"github.com/kataras/iris/v12/_examples/view/quicktemplate/templates"

	"github.com/kataras/iris/v12"
)

// ExecuteTemplate renders a "tmpl" partial template to the `Context.ResponseWriter`.
func ExecuteTemplate(ctx iris.Context, tmpl templates.Partial) {
	ctx.CompressWriter(true)
	ctx.ContentType("text/html")
	templates.WriteTemplate(ctx, tmpl)
}
