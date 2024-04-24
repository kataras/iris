package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/kataras/iris/v12/middleware/requestid"
)

func main() {
	/*
		This example will show you how you can
		register custom fields and log them separately
		with a custom format through the Template formatter.
	*/

	app := iris.New()
	ac := accesslog.File("./access.log").AddOutput(app.Logger().Printer)
	defer ac.Close()

	// 1. Register a field.
	ac.AddFields(func(ctx iris.Context, fields *accesslog.Fields) {
		fields.Set("Request ID", ctx.GetID())
	})

	// 2. Use Template formatter's `Text` value
	// to define a log line format.
	ac.SetFormatter(&accesslog.Template{
		Text: `{{.Now.Format .TimeFormat}} {{.Path}} {{.Code}} {{.IP}} {{.Fields.Get "Request ID" }}
`, /*             2020-09-11 09:30:10 / 200 ::1 050a0979-c5e4-4c2b-9f08-cb456628edb1 */
	})
	// 3. Register the middleware. That's all.
	app.UseRouter(ac.Handler)
	// Register the request id middleware, after the logger,  this maps the Context.GetID().
	// Remember: the accesslog runs the next handlers before itself to provide some fields.
	app.UseRouter(requestid.New())

	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.WriteString("Index")
}

/* Use a custom *template.Template:

// 2.1 The log line format:
text := `{{.Now.Format .TimeFormat}} {{.Path}} {{.Code}} {{.IP}} {{.Fields.Get "Request ID" }}
`
//
// 2.2 Parse the template, optionally using custom Template Functions.
tmpl := template.Must(template.New("").Funcs(template.FuncMap{
	// Custom functions you may want to use inside "text",
	// e.g. prefixFields .Fields "my_prefix"
	// to get a slice of fields starts with "my_prefix"
	// and later, in the template, loop through them and render their values.
	// "key": func(input) string { return ... }
}).Parse(text))
//
// 3. Use Template formatter's `Text` value
// or the `Tmpl` field to customize the look & feel of a log.
ac.SetFormatter(&accesslog.Template{
	Tmpl: tmpl,
})

*/
