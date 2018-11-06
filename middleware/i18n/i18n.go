// Package i18n provides internalization and localization via middleware.
// See _examples/miscellaneous/i18n
package i18n

import (
	"reflect"
	"strings"

	"github.com/iris-contrib/i18n"
	"github.com/kataras/iris/context"
)

// test file: ../../_examples/miscellaneous/i18n/main_test.go
type i18nMiddleware struct {
	config Config
}

// ServeHTTP serves the request, the actual middleware's job is here
func (i *i18nMiddleware) ServeHTTP(ctx context.Context) {
	wasByCookie := false

	langKey := ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey()
	language := ctx.Values().GetString(langKey)
	if language == "" {
		// try to get by url parameter
		language = ctx.URLParam(i.config.URLParameter)
		if language == "" {
			// then try to take the lang field from the cookie
			language = ctx.GetCookie(langKey)

			if len(language) > 0 {
				wasByCookie = true
			} else {
				// try to get by the request headers.
				langHeader := ctx.GetHeader("Accept-Language")
				if len(langHeader) > 0 {
					for _, langEntry := range strings.Split(langHeader, ",") {
						lc := strings.Split(langEntry, ";")[0]
						if lc, ok := i18n.IsExistSimilar(lc); ok {
							language = lc
							break
						}
					}
				}
			}
		}
		// if it was not taken by the cookie, then set the cookie in order to have it
		if !wasByCookie {
			ctx.SetCookieKV(langKey, language)
		}

		if language == "" {
			language = i.config.Default
		}

		ctx.Values().Set(langKey, language)
	}
	locale := i18n.Locale{Lang: language}

	// if unexpected language given, the middleware will  transtlate to the default language, the language key should be
	// also this language instead of the user-given
	if indexLang := locale.Index(); indexLang == -1 {
		locale.Lang = i.config.Default
	}

	translateFuncKey := ctx.Application().ConfigurationReadOnly().GetTranslateFunctionContextKey()
	ctx.Values().Set(translateFuncKey, locale.Tr)
	ctx.Next()
}

// Translate returns the translated word from a context
// the second parameter is the key of the world or line inside the .ini file
// the third parameter is the '%s' of the world or line inside the .ini file
func Translate(ctx context.Context, format string, args ...interface{}) string {
	return ctx.Translate(format, args...)
}

// New returns a new i18n middleware
func New(c Config) context.Handler {
	if len(c.Languages) == 0 {
		panic("You cannot use this middleware without set the Languages option, please try again and read the _example.")
	}
	i := &i18nMiddleware{config: c}
	firstlanguage := ""
	//load the files
	for k, langFileOrFiles := range c.Languages {
		// remove all spaces.
		langFileOrFiles = strings.Replace(langFileOrFiles, " ", "", -1)
		// note: if only one, then the first element is the "v".
		languages := strings.Split(langFileOrFiles, ",")

		for _, v := range languages { // loop each of the files separated by comma, if any.
			if !strings.HasSuffix(v, ".ini") {
				v += ".ini"
			}
			err := i18n.SetMessage(k, v)
			if err != nil && err != i18n.ErrLangAlreadyExist {
				panic("Failed to set locale file'" + k + "' Error:" + err.Error())
			}
			if firstlanguage == "" {
				firstlanguage = k
			}
		}
	}
	// if not default language setted then set to the first of the i.config.Languages
	if c.Default == "" {
		c.Default = firstlanguage
	}

	i18n.SetDefaultLang(i.config.Default)
	return i.ServeHTTP
}

// TranslatedMap returns translated map[string]interface{} from i18n structure.
func TranslatedMap(ctx context.Context, sourceInterface interface{}) map[string]interface{} {
	iType := reflect.TypeOf(sourceInterface).Elem()
	result := make(map[string]interface{})

	for i := 0; i < iType.NumField(); i++ {
		fieldName := reflect.TypeOf(sourceInterface).Elem().Field(i).Name
		fieldValue := reflect.ValueOf(sourceInterface).Elem().Field(i).String()

		result[fieldName] = Translate(ctx, fieldValue)
	}

	return result
}
