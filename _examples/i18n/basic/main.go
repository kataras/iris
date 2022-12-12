package main

import (
	"fmt"
	"html/template"

	"github.com/kataras/iris/v12"
)

/*
	See i18n-template for a more advanced translation key-values.
*/

func newApp() *iris.Application {
	app := iris.New()

	// Configure i18n.
	//
	// app.I18n.Subdomain = false to disable resolve lang code from subdomain.
	// app.I18n.LoadAssets for go-bindata.

	// Default values:
	// app.I18n.URLParameter = "lang"
	// app.I18n.Subdomain = true
	//
	// Set to false to disallow path (local) redirects,
	// see https://github.com/kataras/iris/issues/1369.
	// app.I18n.PathRedirect = true
	//
	// See `app.I18n.ExtractFunc = func(ctx iris.Context) string` or
	// `ctx.SetLanguage(langCode string)` to change the extracted language from a request.
	//
	// Use DefaultMessageFunc to customize the return value of a not found key or lang.
	// All language inputs fallback to the default locale if not matched.
	// This is why this one accepts both input and matched languages,
	// so the caller can be more expressful knowing those.
	// Defaults to nil.
	app.I18n.DefaultMessageFunc = func(langInput, langMatched, key string, args ...interface{}) string {
		msg := fmt.Sprintf("user language input: %s: matched as: %s: not found key: %s: args: %v", langInput, langMatched, key, args)
		app.Logger().Warn(msg)
		return msg
	}
	// Load i18n when customizations are set in place.
	//
	// First parameter: Glob filpath patern,
	// Second variadic parameter: Optional language tags, the first one is the default/fallback one.
	err := app.I18n.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
	if err != nil {
		panic(err)
	}

	app.Get("/not-matched", func(ctx iris.Context) {
		text := ctx.Tr("not_found_key", "some", "values", 42)
		ctx.WriteString(text)
		// user language input: en-gb: matched as: en-US: not found key: not_found_key: args: [some values 42]
	})

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
		if err := ctx.View("index.html", iris.Map{
			"tr": ctx.Tr, // word, arguments... {call .tr "hi" "iris"}}
			"trUnsafe": func(message string, args ...interface{}) template.HTML {
				return template.HTML(ctx.Tr(message, args...))
			},
		}); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}

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
	app.Listen(":8080", iris.WithSitemap("http://localhost:8080"))
}
