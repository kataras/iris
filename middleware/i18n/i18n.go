// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package i18n

import (
	"github.com/Unknwon/i18n"
	"github.com/kataras/iris"
	"strings"
)

var AcceptLanguage = "Accept-Language"

type Options struct {
	// Default set it if you want a default language
	//
	// Checked: Configuration state, not at runtime
	Default string
	// URLParameter is the name of the url parameter which the language can be indentified
	//
	// Checked: Serving state, runtime
	URLParameter string
	// Languages is a map[string]string which the key is the language i81n and the value is the file location
	//
	// Example of key is: 'en-US'
	// Example of value is: './locales/en-US.ini'
	Languages map[string]string
}

type i18nMiddleware struct {
	options Options
}

func (i *i18nMiddleware) Serve(ctx *iris.Context) {
	wasByCookie := false
	// try to get by url parameter
	language := ctx.URLParam(i.options.URLParameter)

	if language == "" {
		// then try to take the lang field from the cookie
		language = ctx.GetCookie("lang")

		if len(language) > 0 {
			wasByCookie = true
		} else {
			// try to get by the request headers(?)
			if langHeader := ctx.Request.Header.Get(AcceptLanguage); i18n.IsExist(langHeader) {
				language = langHeader
			}
		}
	}
	// if it was not taken by the cookie, then set the cookie in order to have it
	if !wasByCookie {
		ctx.SetCookie("language", language)
	}
	if language == "" {
		language = i.options.Default
	}
	locale := i18n.Locale{language}
	ctx.Set("language", language)
	ctx.Set("translate", locale.Tr)
	ctx.Next()
}

func I18nHandler(_options ...Options) *i18nMiddleware {
	i := &i18nMiddleware{}
	if len(_options) == 0 || (len(_options) > 0 && len(_options[0].Languages) == 0) {
		panic("You cannot use this middleware without set the Languages option, please try again and read the docs.")
	}

	i.options = _options[0]
	firstlanguage := ""
	//load the files
	for k, v := range i.options.Languages {
		if !strings.HasSuffix(v, ".ini") {
			v += ".ini"
		}
		err := i18n.SetMessage(k, v)
		if err != nil && err != i18n.ErrLangAlreadyExist {
			panic("Iris i18n Middleware: Failed to set locale file" + k + " Error:" + err.Error())
		}
		if firstlanguage == "" {
			firstlanguage = k
		}
	}
	// if not default language setted then set to the first of the i.options.Languages
	if i.options.Default == "" {
		i.options.Default = firstlanguage
	}

	i18n.SetDefaultLang(i.options.Default)
	return i
}

func I18n(_options ...Options) iris.HandlerFunc {
	return I18nHandler(_options...).Serve
}
