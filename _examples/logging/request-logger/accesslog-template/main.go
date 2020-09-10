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
	// to customize the look & feel of a log.
	// You could also use its `Tmpl` field to
	// set a *template.Template instance.
	ac.SetFormatter(&accesslog.Template{
		Text: `{{.Now.Format .TimeFormat}} {{.Path}} {{.Code}} {{.Fields.Get "IP" }}
`,
	})
	// Example Output:
	// 2020-09-10 21:38:13 / 200 ::1

	app.UseRouter(ac.Handler)

	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.WriteString("Index")
}
