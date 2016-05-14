package i18n

import (
	"strings"

	"github.com/Unknwon/i18n"
	"github.com/kataras/iris"
)

// AcceptLanguage is the Header key "Accept-Language"
const AcceptLanguage = "Accept-Language"

// Options the i18n options
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
			if langHeader := ctx.RequestHeader(AcceptLanguage); i18n.IsExist(langHeader) {
				language = langHeader
			}
		}
	}
	// if it was not taken by the cookie, then set the cookie in order to have it
	if !wasByCookie {
		ctx.SetCookieKV("language", language)
	}
	if language == "" {
		language = i.options.Default
	}
	locale := i18n.Locale{language}
	ctx.Set("language", language)
	ctx.Set("translate", locale.Tr)
	ctx.Next()
}

// I18nHandler returns the middleware which is just an iris.handler
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

// I18n returns the middleware as iris.HandlerFunc with the passed options
func I18n(_options ...Options) iris.HandlerFunc {
	return I18nHandler(_options...).Serve
}
