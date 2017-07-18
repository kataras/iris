package controllers

import (
	"github.com/kataras/iris/_examples/http_responsewriter/quicktemplate/templates"

	"github.com/kataras/iris/context"
)

// ExecuteTemplate renders a "tmpl" partial template to the `context#ResponseWriter`.
func ExecuteTemplate(ctx context.Context, tmpl templates.Partial) {
	ctx.Gzip(true)
	ctx.ContentType("text/html")
	templates.WriteTemplate(ctx.ResponseWriter(), tmpl)
}
