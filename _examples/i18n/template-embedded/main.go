package main

import (
	"embed"
	"strings"
	"text/template"

	"github.com/kataras/iris/v12"
)

//go:embed embedded/locales/*
var embeddedFS embed.FS

func main() {
	app := newApp()
	// http://localhost:8080
	// http://localhost:8080?lang=el
	// http://localhost:8080?lang=el
	// http://localhost:8080?lang=el-GR
	// http://localhost:8080?lang=en
	// http://localhost:8080?lang=en-US
	//
	// http://localhost:8080/title
	// http://localhost:8080/title?lang=el-GR
	// ...
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	// Set custom functions per locale!
	app.I18n.Loader.Funcs = func(current iris.Locale) template.FuncMap {
		return template.FuncMap{
			"uppercase": func(word string) string {
				return strings.ToUpper(word)
			},
		}
	}

	// Instead of:
	// err := app.I18n.Load("./locales/*/*.ini", "en-US", "el-GR")
	// apply the below in order to build with embedded locales inside your executable binary.
	err := app.I18n.LoadFS(embeddedFS, "./embedded/locales/*/*.ini", "en-US", "el-GR")
	if err != nil {
		panic(err)
	} // OR to load all languages by filename:
	// app.I18n.LoadFS(embeddedFS, "./embedded/locales/*/*.ini")
	// Then set the default language using:
	// app.I18n.SetDefault("en-US")

	app.Get("/", func(ctx iris.Context) {
		text := ctx.Tr("forms.register") // en-US: prints "Become a MEMBER".
		ctx.WriteString(text)
	})

	app.Get("/title", func(ctx iris.Context) {
		text := ctx.Tr("user.connections.Title") // en-US: prints "Accounts Connections".
		ctx.WriteString(text)
	})

	return app
}
