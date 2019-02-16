package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/i18n"
)

func newApp() *iris.Application {
	app := iris.New()

	globalLocale := i18n.New(i18n.Config{
		Default:      "en-US",
		URLParameter: "lang",
		Languages: map[string]string{
			"en-US": "./locales/locale_en-US.ini",
			"el-GR": "./locales/locale_el-GR.ini",
			"zh-CN": "./locales/locale_zh-CN.ini"}})
	app.Use(globalLocale)

	app.Get("/", func(ctx iris.Context) {

		// it tries to find the language by:
		// ctx.Values().GetString("language")
		// if that was empty then
		// it tries to find from the URLParameter setted on the configuration
		// if not found then
		// it tries to find the language by the "language" cookie
		// if didn't found then it it set to the Default setted on the configuration

		// hi is the key, 'iris' is the %s on the .ini file
		// the second parameter is optional

		// hi := ctx.Translate("hi", "iris")
		// or:
		hi := i18n.Translate(ctx, "hi", "iris")

		language := ctx.Values().GetString(ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey())
		// return is form of 'en-US'

		// The first succeed language found saved at the cookie with name ("language"),
		//  you can change that by changing the value of the:  iris.TranslateLanguageContextKey
		ctx.Writef("From the language %s translated output: %s", language, hi)
	})

	multiLocale := i18n.New(i18n.Config{
		Default:      "en-US",
		URLParameter: "lang",
		Languages: map[string]string{
			"en-US": "./locales/locale_multi_first_en-US.ini, ./locales/locale_multi_second_en-US.ini",
			"el-GR": "./locales/locale_multi_first_el-GR.ini, ./locales/locale_multi_second_el-GR.ini"}})

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
			"tr": ctx.Translate,
		})
		// it will return "hello, iris"
		// when {{call .tr "hi" "iris"}}
	})
	//

	return app
}

func main() {
	app := newApp()

	// go to http://localhost:8080/?lang=el-GR
	// or http://localhost:8080 (default is en-US)
	// or http://localhost:8080/?lang=zh-CN
	//
	// go to http://localhost:8080/multi?lang=el-GR
	// or http://localhost:8080/multi (default is en-US)
	// or http://localhost:8080/multi?lang=en-US
	//
	// or use cookies to set the language.
	app.Run(iris.Addr(":8080"))
}
