// Package i18n provides internalization and localization via middleware.
// See _examples/miscellaneous/i18n
package i18n

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/iris-contrib/i18n"
	"github.com/kataras/iris/v12/context"
)

// If `Config.Default` is missing and `Config.Languages` or `Config.Alternatives` contains this key then it will set as the default locale,
// no need to be exported(see `Config.Default`).
const defLang = "en-US"

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
	// Cookie is the key of the request cookie which the language can be indentified,
	// e.g. "lang".
	//
	// Checked: Serving state, runtime.
	Cookie string
	// If SetCookie is true and Cookie field is not empty
	// then it will set the cookie to the language found by Context's Value's "lang" key or URLParameter or Cookie or Indentifier.
	// Defaults to false.
	SetCookie bool

	// If Subdomain is true then it will try to map a subdomain
	// with a valid language from the language list or alternatives.
	Subdomain bool

	// Indentifier is a function which the language can be indentified if the above URLParameter and Cookie failed to.
	Indentifier func(context.Context) string

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

// all locale files passed, we keep them in order
// to check if a file is already passed by `New` or `NewWrapper`,
// because we don't have a way to check before the appending of
// a locale file and the same locale code can be used more than one to register different file names (at runtime too).
var (
	localeFilesSet = make(map[string]struct{})
	localesMutex   sync.RWMutex
	once           sync.Once
)

func (c *Config) loadLanguages() {
	if len(c.Languages) == 0 {
		panic("field Languages is empty")
	}

	for k, v := range c.Alternatives {
		if _, ok := c.Languages[v]; !ok {
			panic(fmt.Sprintf("language alternative '%s' does not map to a valid language '%s'", k, v))
		}
	}

	// load the files
	for k, langFileOrFiles := range c.Languages {
		// remove all spaces.
		langFileOrFiles = strings.Replace(langFileOrFiles, " ", "", -1)
		// note: if only one, then the first element is the "v".
		languages := strings.Split(langFileOrFiles, ",")

		for _, v := range languages { // loop each of the files separated by comma, if any.
			if !strings.HasSuffix(v, ".ini") {
				v += ".ini"
			}

			localesMutex.RLock()
			_, exists := localeFilesSet[v]
			localesMutex.RUnlock()
			if !exists {
				localesMutex.Lock()
				err := i18n.SetMessage(k, v)
				// fmt.Printf("add %s = %s\n", k, v)
				if err != nil && err != i18n.ErrLangAlreadyExist {
					panic(fmt.Sprintf("Failed to set locale file' %s' with error: %v", k, err))
				}

				localeFilesSet[v] = struct{}{}
				localesMutex.Unlock()
			}

		}
	}

	if c.Default == "" {
		if lang, ok := c.Exists(defLang); ok {
			c.Default = lang
		}
	}

	once.Do(func() { // set global default lang once.
		// fmt.Printf("set default language: %s\n", c.Default)
		i18n.SetDefaultLang(c.Default)
	})
}

// I18n is the structure which keeps the i18n configuration and implement all Iris i18n features.
type I18n struct {
	config Config
}

// NewI18n returns a new i18n middleware which contains
// the middleware itself and a router wrapper.
func NewI18n(config Config) *I18n {
	config.loadLanguages()
	return &I18n{config}
}

// Handler returns the middleware handler.
func (i *I18n) Handler() context.Handler {
	return func(ctx context.Context) {
		wasByCookie := false

		langKey := ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey()
		language := ctx.Values().GetString(langKey)

		if language == "" {
			if i.config.URLParameter != "" {
				// try to get by url parameter
				language = ctx.URLParam(i.config.URLParameter)
			}

			if language == "" {
				if i.config.Cookie != "" {
					// then try to take the lang field from the cookie
					language = ctx.GetCookie(i.config.Cookie)
					wasByCookie = language != ""
				}

				if language == "" && i.config.Subdomain {
					if subdomain := ctx.Subdomain(); subdomain != "" {
						if lang, ok := i.config.Exists(subdomain); ok {
							language = lang
						}
					}
				}

				if language == "" {
					// try to get by the request headers.
					langHeader := ctx.GetHeader("Accept-Language")
					if len(langHeader) > 0 {
						for _, langEntry := range strings.Split(langHeader, ",") {
							lc := strings.Split(langEntry, ";")[0]
							if lang, ok := i.config.Exists(lc); ok {
								language = lang
								break
							}
						}
					}
				}

				if language == "" && i.config.Indentifier != nil {
					language = i.config.Indentifier(ctx)
				}
			}
		}

		if language == "" {
			language = i.config.Default
		}

		// returns the original key of the language and true
		// when the language, or something similar exists (e.g. en-US maps to en).
		if lc, ok := i.config.Exists(language); ok {
			language = lc
		}

		// if it was not taken by the cookie, then set the cookie in order to have it.
		if !wasByCookie && i.config.SetCookie && i.config.Cookie != "" {
			ctx.SetCookieKV(i.config.Cookie, language)
		}

		ctx.Values().Set(langKey, language)

		// Set iris.translate and iris.translateLang functions (they can be passed to templates as they are later on).
		ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetTranslateFunctionContextKey(), getTranslateFunction(language))
		// Note: translate (global) language function input argument should match exactly, case-sensitive and "Alternatives" field is not part of the fetch progress.
		ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetTranslateLangFunctionContextKey(), i18n.Tr)

		ctx.Next()
	}
}

// Wrapper returns a new router wrapper.
// The result function can be passed on `Application.WrapRouter`.
// It compares the path prefix for translated language and
// local redirects the requested path with the selected (from the path) language to the router.
//
// In order this to work as expected, it should be combined with `Application.Use(i.Handler())`
// which registers the i18n middleware itself.
func (i *I18n) Wrapper() func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	return func(w http.ResponseWriter, r *http.Request, routerHandler http.HandlerFunc) {
		found := false
		path := r.URL.Path[1:]

		if idx := strings.IndexByte(path, '/'); idx > 0 {
			path = path[:idx]
		}

		if path != "" {
			if lang, ok := i.config.Exists(path); ok {
				path = r.URL.Path[len(path)+1:]
				if path == "" {
					path = "/"
				}
				r.RequestURI = path
				r.URL.Path = path
				r.Header.Set("Accept-Language", lang)
				found = true
			}
		}

		if !found && i.config.Subdomain {
			host := context.GetHost(r)
			if dotIdx := strings.IndexByte(host, '.'); dotIdx > 0 {
				subdomain := host[0:dotIdx]
				if subdomain != "" {
					if lang, ok := i.config.Exists(subdomain); ok {
						host = host[dotIdx+1:]
						r.URL.Host = host
						r.Host = host
						r.Header.Set("Accept-Language", lang)
					}
				}
			}
		}

		routerHandler(w, r)
	}
}

func getTranslateFunction(lang string) func(string, ...interface{}) string {
	return func(format string, args ...interface{}) string {
		return i18n.Tr(lang, format, args...)
	}
}

// New returns a new i18n middleware.
func New(config Config) context.Handler {
	return NewI18n(config).Handler()
}

// NewWrapper accepts a Config and returns a new router wrapper.
// The result function can be passed on `Application.WrapRouter`.
// It compares the path prefix for translated language and
// local redirects the requested path with the selected (from the path) language to the router.
//
// In order this to work as expected, it should be combined with `Application.Use(New)`
// which registers the i18n middleware itself.
func NewWrapper(config Config) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	return NewI18n(config).Wrapper()
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
