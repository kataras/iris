package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
)

func main() {
	/*
		This example will show you how you can
		register custom fields and log them separately
		with a custom format through the Template formatter.
	*/

	app := iris.New()
	ac := accesslog.File("./access.log").AddOutput(app.Logger().Printer)
	ac.TimeFormat = "2006-01-02 15:04:05"

	// 1. Register a field.
	ac.AddFields(func(ctx iris.Context, fields *accesslog.Fields) {
		fields.Set("IP", ctx.RemoteAddr())
	})

	// 2. Use Template formatter's `Text` value
	// to define a log line format.
	ac.SetFormatter(&accesslog.Template{
		Text: `{{.Now.Format .TimeFormat}} {{.Path}} {{.Code}} {{.Fields.Get "IP" }}
`, /*             2020-09-10 21:38:13 / 200 ::1 */
	})
	// 3. Register the middleware. That's all.
	app.UseRouter(ac.Handler)

	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.WriteString("Index")
}

/* Use a custom *template.Template:

// 2.1 The log line format:
text := `{{.Now.Format .TimeFormat}} {{.Path}} {{.Code}} {{.Fields.Get "IP" }}
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
