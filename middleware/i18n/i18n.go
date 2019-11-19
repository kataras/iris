// Package i18n provides internalization and localization via middleware.
// See _examples/miscellaneous/i18n
package i18n

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/iris-contrib/i18n"
	"github.com/kataras/iris/v12/context"
)

// Config the i18n options.
type Config struct {
	// Default set it if you want a default language.
	//
	// Checked: Configuration state, not at runtime.
	Default string
	// URLParameter is the name of the url parameter which the language can be indentified,
	// e.g. "lang" for ?lang=.
	//
	// Checked: Serving state, runtime.
	URLParameter string
	// PathParameter is the name of the path parameter which the language can be indentified,
	// e.g. "lang" for "{lang:string}".
	//
	// Checked: Serving state, runtime.
	//
	// You can set custom handler to set the language too.
	// Example:
	// setLangMiddleware := func(ctx iris.Context){
	// 	langKey := ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey()
	// 	languageByPath := ctx.Params().Get("lang") // see {lang}
	// 	ctx.Values().Set(langKey, languageByPath)
	// 	ctx.Next()
	// }
	// app.Use(setLangMiddleware)
	// app.Use(theI18nMiddlewareInstance)
	PathParameter string

	// Languages is a map[string]string which the key is the language i81n and the value is the file location.
	//
	// Example of key is: 'en-US'.
	// Example of value is: './locales/en-US.ini'.
	Languages map[string]string
	// Alternatives is a language map which if it's filled,
	// it tries to associate its keys with a value of "Languages" field when a possible value of "Language" was not present.
	// Example of
	// Languages: map[string]string{"en-US": "./locales/en-US.ini"} set
	// Alternatives: map[string]string{ "en":"en-US", "english": "en-US"}.
	Alternatives map[string]string

	// If SetCookie is true then it will set the cookie to the language found by URLParameter, PathParameter or by Context's Value's "lang" key.
	// Defaults to false.
	SetCookie bool
}

// Exists returns true if the language, or something similar
// exists (e.g. en-US maps to en or Alternatives[key] == lang).
// it returns the found name and whether it was able to match something.
func (c *Config) Exists(lang string) (string, bool) {
	for k, v := range c.Alternatives {
		if k == lang {
			lang = v
			break
		}
	}

	return i18n.IsExistSimilar(lang)
}

func (c *Config) loadLanguages() {
	if len(c.Languages) == 0 {
		panic("field Languages is empty")
	}

	for k, v := range c.Alternatives {
		if _, ok := c.Languages[v]; !ok {
			panic(fmt.Sprintf("language alternative '%s' does not map to a valid language '%s'", k, v))
		}
	}

	firstlanguage := ""
	// load the files
	for k, langFileOrFiles := range c.Languages {
		if i18n.IsExist(k) {
			// if it is already stored through middleware (`New`) then skip it.
			continue
		}

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
				panic(fmt.Sprintf("Failed to set locale file' %s' with error: %v", k, err))
			}
			if firstlanguage == "" {
				firstlanguage = k
			}
		}
	}
	// if not default language set then set to the first of the "Languages".
	if c.Default == "" {
		c.Default = firstlanguage
	}

	i18n.SetDefaultLang(c.Default)
}

// test file: ../../_examples/miscellaneous/i18n/main_test.go
type i18nMiddleware struct {
	config Config
}

// New returns a new i18n middleware.
func New(c Config) context.Handler {
	c.loadLanguages()
	i := &i18nMiddleware{config: c}
	return i.ServeHTTP
}

// ServeHTTP serves the request, the actual middleware's job is located here.
func (i *i18nMiddleware) ServeHTTP(ctx context.Context) {
	wasByCookie := false

	langKey := ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey()
	language := ctx.Values().GetString(langKey)
	if language == "" {
		// try to get by path parameter
		if i.config.PathParameter != "" {
			language = ctx.Params().Get(i.config.PathParameter)
		}

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
							if lc, ok := i.config.Exists(lc); ok {
								language = lc
								break
							}
						}
					}
				}
			}
		}
	}

	// returns the original key of the language and true
	// when the language, or something similar exists (e.g. en-US maps to en).
	if lc, ok := i.config.Exists(language); ok {
		language = lc
	} else {
		// if unexpected language given, the middleware will translate to the default language,
		// the language key should be also this language instead of the user-given.
		language = i.config.Default
	}

	// if it was not taken by the cookie, then set the cookie in order to have it.
	if !wasByCookie && i.config.SetCookie {
		ctx.SetCookieKV(langKey, language)
	}

	ctx.Values().Set(langKey, language)

	// Set iris.translate and iris.translateLang functions (they can be passed to templates as they are later on).
	ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetTranslateFunctionContextKey(), getTranslateFunction(language))
	// Note: translate (global) language function input argument should match exactly, case-sensitive and "Alternatives" field is not part of the fetch progress.
	ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetTranslateLangFunctionContextKey(), i18n.Tr)

	ctx.Next()
}

func getTranslateFunction(lang string) func(string, ...interface{}) string {
	return func(format string, args ...interface{}) string {
		return i18n.Tr(lang, format, args...)
	}
}

// NewWrapper accepts a Config and returns a new router wrapper.
// The result function can be passed on `Application.WrapRouter`.
// It compares the path prefix for translated language and
// local redirects the requested path with the selected (from the path) language to the router.
//
// In order this to work as expected, it should be combined with `Application.Use(New)`
// which registers the i18n middleware itself.
func NewWrapper(c Config) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	c.loadLanguages()

	return func(w http.ResponseWriter, r *http.Request, routerHandler http.HandlerFunc) {
		path := r.URL.Path[1:]

		if idx := strings.IndexRune(path, '/'); idx > 0 {
			path = path[:idx]
		}

		if lang, ok := c.Exists(path); ok {
			path = r.URL.Path[len(path)+1:]
			if path == "" {
				path = "/"
			}
			r.RequestURI = path
			r.URL.Path = path
			r.Header.Set("Accept-Language", lang)
		}

		routerHandler(w, r)
	}
}

// Translate returns the translated word from a context based on the current selected locale.
// The second parameter is the key of the world or line inside the .ini file and
// the third parameter is the '%s' of the world or line inside the .ini file
func Translate(ctx context.Context, format string, args ...interface{}) string {
	return ctx.Translate(format, args...)
}

// TranslateLang returns the translated word from a context based on the given "lang".
// The second parameter is the language key which the word "format" is translated to and
// the third parameter is the key of the world or line inside the .ini file and
// the forth parameter is the '%s' of the world or line inside the .ini file
func TranslateLang(ctx context.Context, lang, format string, args ...interface{}) string {
	return ctx.TranslateLang(lang, format, args...)
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
