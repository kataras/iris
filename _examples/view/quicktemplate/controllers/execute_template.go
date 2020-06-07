package controllers

import (
	"github.com/kataras/iris/v12/_examples/response-writer/quicktemplate/templates"

	"github.com/kataras/iris/v12"
)

// ExecuteTemplate renders a "tmpl" partial template to the `context#ResponseWriter`.
func ExecuteTemplate(ctx iris.Context, tmpl templates.Partial) {
	ctx.Gzip(true)
	ctx.ContentType("text/html")
	templates.WriteTemplate(ctx.ResponseWriter(), tmpl)
}
