package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/i18n"
)

func newApp() *iris.Application {
	app := iris.New()
	app.Logger().SetLevel("debug")

	i18nConfig := i18n.Config{
		Default:       "en-US",
		URLParameter:  "lang",
		PathParameter: "lang",
		Languages: map[string]string{
			"en-US": "./locales/locale_en-US.ini",
			"el-GR": "./locales/locale_el-GR.ini",
			"zh-CN": "./locales/locale_zh-CN.ini",
		},
		Alternatives: map[string]string{"greek": "el-GR"},
	}

	// See https://github.com/kataras/iris/issues/1369
	// if you want to enable this (SEO) feature.
	app.WrapRouter(i18n.NewWrapper(i18nConfig))

	i18nMiddleware := i18n.New(i18nConfig)
	app.Use(i18nMiddleware)

	app.Get("/", func(ctx iris.Context) {
		// Ir tries to find the language by:
		// ctx.Values().GetString("language")
		// if that was empty then
		// it tries to find from the URLParameter set on the configuration
		// if not found then
		// it tries to find the language by the "language" cookie
		// if didn't found then it it set to the Default set on the configuration

		// hi is the key, 'iris' is the %s on the .ini file
		// the second parameter is optional

		// hi := ctx.Translate("hi", "iris")
		// or:
		hi := i18n.Translate(ctx, "hi", "iris")

		// GetTranslateLanguageContextKey() == "language"
		language := ctx.Values().GetString(ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey())
		// return is form of 'en-US'

		// The first succeed language found saved at the cookie with name ("language"),
		// you can change that by changing the value of the:  iris.TranslateLanguageContextKey
		ctx.Writef("From the language %s translated output: %s", language, hi)
	})

	app.Get("/some-path", func(ctx iris.Context) {
		ctx.Writef("%s", ctx.Translate("hi", "iris"))
	})

	app.Get("/sitemap.xml", func(ctx iris.Context) {
		ctx.WriteString("sitemap")
	})

	multiLocale := i18n.New(i18n.Config{
		Default:      "en-US",
		URLParameter: "lang",
		Languages: map[string]string{
			"en-US": "./locales/locale_multi_first_en-US.ini, ./locales/locale_multi_second_en-US.ini",
			"el-GR": "./locales/locale_multi_first_el-GR.ini, ./locales/locale_multi_second_el-GR.ini",
		},
	})

	app.Get("/multi", multiLocale, func(ctx iris.Context) {
		language := ctx.Values().GetString(ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey())

		fromFirstFileValue := i18n.Translate(ctx, "key1")
		fromSecondFileValue := i18n.Translate(ctx, "key2")
		ctx.Writef("From the language: %s, translated output:\n%s=%s\n%s=%s",
			language, "key1", fromFirstFileValue,
			"key2", fromSecondFileValue)
	})

	// using in inside your templates:
	view := iris.HTML("./templates", ".html")
	app.RegisterView(view)

	app.Get("/templates", func(ctx iris.Context) {
		ctx.View("index.html", iris.Map{
			"tr":     ctx.Translate,     // word, arguments...
			"trLang": ctx.TranslateLang, // locale, word, arguments...
		})
		// it will return "hello, iris"
		// when {{call .tr "hi" "iris"}}
	})
	//

	return app
}

func main() {
	app := newApp()

	// go to http://localhost:8080/el-GR/some-path
	// or http://localhost:8080/zh-cn/templates
	// or http://localhost:8080/some-path?lang=el-GR
	// or http://localhost:8080 (default is en-US)
	// or http://localhost:8080/?lang=zh-CN
	//
	// go to http://localhost:8080/multi?lang=el-GR
	// or http://localhost:8080/multi (default is en-US)
	// or http://localhost:8080/multi?lang=en-US
	//
	// or use cookies to set the language.
	//
	app.Run(iris.Addr(":8080"))
}
