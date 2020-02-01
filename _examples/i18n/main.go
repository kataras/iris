package main

import (
	"github.com/kataras/iris/v12"
)

func newApp() *iris.Application {
	app := iris.New()

	// Configure i18n.
	// First parameter: Glob filpath patern,
	// Second variadic parameter: Optional language tags, the first one is the default/fallback one.
	app.I18n.Load("./locales/*/*.ini", "en-US", "el-GR", "zh-CN")
	// app.I18n.LoadAssets for go-bindata.

	// Default values:
	// app.I18n.URLParameter = "lang"
	// app.I18n.Subdomain = true
	//
	// Set to false to disallow path (local) redirects,
	// see https://github.com/kataras/iris/issues/1369.
	// app.I18n.PathRedirect = true

	app.Get("/", func(ctx iris.Context) {
		hi := ctx.Tr("hi", "iris")

		locale := ctx.GetLocale()

		ctx.Writef("From the language %s translated output: %s", locale.Language(), hi)
	})

	app.Get("/some-path", func(ctx iris.Context) {
		ctx.Writef("%s", ctx.Tr("hi", "iris"))
	})

	app.Get("/other", func(ctx iris.Context) {
		language := ctx.GetLocale().Language()

		fromFirstFileValue := ctx.Tr("key1")
		fromSecondFileValue := ctx.Tr("key2")
		ctx.Writef("From the language: %s, translated output:\n%s=%s\n%s=%s",
			language, "key1", fromFirstFileValue,
			"key2", fromSecondFileValue)
	})

	// using in inside your views:
	view := iris.HTML("./views", ".html")
	app.RegisterView(view)

	app.Get("/templates", func(ctx iris.Context) {
		ctx.View("index.html", iris.Map{
			"tr": ctx.Tr, // word, arguments... {call .tr "hi" "iris"}}
		})

		// Note that,
		// Iris automatically adds a "tr" global template function as well,
		// the only difference is the way you call it inside your templates and
		// that it accepts a language code as its first argument: {{ tr "el-GR" "hi" "iris"}}
	})
	//

	return app
}

func main() {
	app := newApp()

	// go to http://localhost:8080/el-gr/some-path
	// ^ (by path prefix)
	//
	// or http://el.mydomain.com8080/some-path
	// ^ (by subdomain - test locally with the hosts file)
	//
	// or http://localhost:8080/zh-CN/templates
	// ^ (by path prefix with uppercase)
	//
	// or http://localhost:8080/some-path?lang=el-GR
	// ^ (by url parameter)
	//
	// or http://localhost:8080 (default is en-US)
	// or http://localhost:8080/?lang=zh-CN
	//
	// go to http://localhost:8080/other?lang=el-GR
	// or http://localhost:8080/other (default is en-US)
	// or http://localhost:8080/other?lang=en-US
	//
	// or use cookies to set the language.
	app.Run(iris.Addr(":8080"), iris.WithSitemap("http://localhost:8080"))
}
