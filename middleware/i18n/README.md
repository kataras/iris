## Middleware information

This folder contains a middleware for internationalization uses a third-party package named i81n.

More can be found here: 
[https://github.com/Unknwon/i18n](https://github.com/Unknwon/i18n)


## Description

Package i18n is for app Internationalization and Localization.


## How to use

Create folder named 'locales'
```
///Files: 

./locales/locale_en-US.ini 
./locales/locale_el-US.ini 
```
Contents on locale_en-US:
``` 
hi = hello, %s
``` 
Contents on locale_el-GR:
``` 
hi = Гейб, %s
``` 

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/i18n"
)

func main() {

	iris.UseFunc(i18n.I18n(i18n.Options{Default: "en-US",
		Languages: map[string]string{
			"en-US": "./locales/locale_en-US.ini",
			"el-GR": "./locales/locale_el-GR.ini",
			"zh-CN": "./locales/locale_zh-CN.ini"}}))	
	// or iris.Use(i18n.I18nHandler(....))
	// or iris.Get("/",i18n.I18n(....), func (ctx *iris.Context){}) 
		
	iris.Get("/", func(ctx *iris.Context) {
		hi := ctx.GetFmt("translate")("hi", "maki") // hi is the key, 'maki' is the %s, the second parameter is optional
		language := ctx.Get("language") // language is the language key, example 'en-US'

		ctx.Write("From the language %s translated output: %s", language, hi)
	})
	
	
	println("Server is running at :8080")
	iris.Listen(":8080")

}

```

### [For a working example, click here](https://github.com/kataras/iris/tree/examples/middleware_internationalization_i18n)