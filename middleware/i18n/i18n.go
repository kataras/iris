// Package i18n provides internalization and localization via middleware.
// See _examples/miscellaneous/i18n
package i18n

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12/context"

	"gopkg.in/ini.v1"
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
	// with a valid language from the language list or a valid map to a language.
	Subdomain bool

	// Indentifier is a function which the language can be indentified if the above URLParameter and Cookie failed to.
	Indentifier func(context.Context) string

	// Languages is a map[string]string which the key is the language i81n and the value is the file location.
	//
	// Example of key is: 'en-US'.
	// Example of value is: './locales/en-US.ini'.
	Languages map[string]string
	// LanguagesMap is a language map which if it's filled,
	// it tries to associate an incoming possible language code to a key of "Languages" field
	// when the value of "Language" was not present as it is at serve-time.
	//
	// Defaults to a non-nil LanguagesMap which accepts all lowercase and [en] as [en-US] and e.t.c.
	LanguagesMap LanguagesMap
}

// LanguagesMap the type for mapping an incoming word to a locale.
type LanguagesMap interface {
	Map(lang string) (locale string, found bool)
}

// Map is a Go map[string]string type which is a LanguagesMap that
// matches literal key with value as the found locale.
type Map map[string]string

// Map loops through its registered alternative language codes
// and reports if it is valid registered locale one.
func (m Map) Map(lang string) (string, bool) {
	locale, ok := m[lang]
	return locale, ok
}

// MapFunc is a function shortcut for the LanguagesMap.
type MapFunc func(lang string) (locale string, found bool)

// Map should report if a given "lang" is valid registered locale.
func (m MapFunc) Map(lang string) (string, bool) {
	return m(lang)
}

func makeDefaultLanguagesMap(languages map[string]string) MapFunc {
	return func(lang string) (string, bool) {
		lang = strings.ToLower(lang)
		for locale := range languages {
			if lang == strings.ToLower(locale) {
				return locale, true
			}

			// this matches "en-anything" too, which can be accepted too on some cases, but not here.
			// if sep := strings.IndexRune(lang, '-'); sep > 0 {
			// 	lang = lang[0:sep]
			// }

			if len(lang) == 2 {
				if strings.Contains(locale, lang) {
					return locale, true
				}
			}
		}

		return "", false
	}
}

// I18n is the structure which keeps the i18n configuration and implement all Iris i18n features.
type I18n struct {
	config Config

	locales map[string][]*ini.File
}

// If `Config.Default` is missing and `Config.Languages` or `Config.LanguagesMap` contains this key then it will set as the default locale,
// no need to be exported(see `Config.Default`).
const defLangCode = "en-US"

// NewI18n returns a new i18n middleware which contains
// the middleware itself and a router wrapper.
func NewI18n(c Config) *I18n {
	if len(c.Languages) == 0 {
		panic("field Languages is empty")
	}

	// check and validate (if possible) languages map.
	if c.LanguagesMap == nil {
		c.LanguagesMap = makeDefaultLanguagesMap(c.Languages)
	}

	if mTyp, ok := c.LanguagesMap.(Map); ok {
		for k, v := range mTyp {
			if _, ok := c.Languages[v]; !ok {
				panic(fmt.Sprintf("language alternative '%s' does not map to a valid language '%s'", k, v))
			}
		}
	}

	i := new(I18n)

	// load messages.
	i.locales = make(map[string][]*ini.File)
	for locale, src := range c.Languages {
		if err := i.AddSource(locale, src); err != nil {
			panic(err)
		}
	}

	// validate and set default lang code.
	if c.Default == "" {
		c.Default = defLangCode
	}

	if locale, _, ok := i.Exists(c.Default); !ok {
		panic(fmt.Sprintf("default language '%s' does not match any of the registered language", c.Default))
	} else {
		c.Default = locale
	}

	i.config = c

	return i
}

// AddSource adds a source file to the lang locale.
// It is called on NewI18n, New and NewWrapper.
//
// If you wish to use this at serve-time please protect the process with a mutex.
func (i *I18n) AddSource(locale, src string) error {
	// remove all spaces.
	src = strings.Replace(src, " ", "", -1)
	// note: if only one, then the first element is the "v".
	languageFiles := strings.Split(src, ",")

	for _, fileName := range languageFiles {
		if !strings.HasSuffix(fileName, ".ini") {
			fileName += ".ini"
		}

		f, err := ini.Load(fileName)
		if err != nil {
			return err
		}

		i.locales[locale] = append(i.locales[locale], f)
	}

	return nil
}

// GetMessage returns a message from a locale, locale is case-sensitivity and languages map does not playing its part here.
func (i *I18n) GetMessage(locale, section, format string, args ...interface{}) (string, bool) {
	files, ok := i.locales[locale]
	if !ok {
		return "", false
	}

	return i.getMessage(files, section, format, args)
}

func (i *I18n) getMessage(files []*ini.File, section, format string, args []interface{}) (string, bool) {
	for _, f := range files {
		// returns the first available.
		// section is the same for both files if key(format) exists.
		s, err := f.GetSection(section)
		if err != nil {
			return "", false
		}

		k, err := s.GetKey(format)
		if err != nil {
			continue
		}

		format = k.Value()
		if len(args) > 0 {
			return fmt.Sprintf(format, args...), true
		}

		return format, true
	}

	return "", false
}

// Translate translates and returns a message based on any language code
// and its key(format) with any optional arguments attached to it.
func (i *I18n) Translate(lang, format string, args ...interface{}) string {
	if _, files, ok := i.Exists(lang); ok {
		return i.translate(files, format, args)
	}

	return ""
}

func (i *I18n) translate(files []*ini.File, format string, args []interface{}) string {
	section := ""

	if idx := strings.IndexRune(format, '.'); idx > 0 {
		section = format[:idx]
		format = format[idx+1:]
	}

	msg, ok := i.getMessage(files, section, format, args)
	if !ok {
		return fmt.Sprintf(format, args...)
	}

	return msg
}

// Exists reports whether a language code is a valid registered locale through its Languages list and Languages mapping.
func (i *I18n) Exists(lang string) (string, []*ini.File, bool) {
	if lang == "" {
		return "", nil, false
	}

	files, ok := i.locales[lang]
	if ok {
		return lang, files, true
	}

	for locale, files := range i.locales {
		if locale == lang {
			return locale, files, true
		}
	}

	if i.config.LanguagesMap != nil {
		if locale, ok := i.config.LanguagesMap.Map(lang); ok {
			if files, ok := i.locales[locale]; ok {
				return locale, files, true
			}
		}
	}

	return "", nil, false
}

func (i *I18n) newTranslateLanguageFunc(files []*ini.File) func(format string, args ...interface{}) string {
	return func(format string, args ...interface{}) string {
		return i.translate(files, format, args)
	}
}

const acceptLanguageHeaderKey = "Accept-Language"

// Handler returns the middleware handler.
func (i *I18n) Handler() context.Handler {
	return func(ctx context.Context) {
		wasByCookie := false

		langKey := ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey()

		language, files, ok := i.Exists(ctx.Values().GetString(langKey))

		if !ok {
			if i.config.URLParameter != "" {
				language, files, ok = i.Exists(ctx.URLParam(i.config.URLParameter))
			}

			if !ok {
				// then try to take the lang field from the cookie
				if i.config.Cookie != "" {
					if language, files, ok = i.Exists(ctx.GetCookie(i.config.Cookie)); ok {
						wasByCookie = true
					}
				}

				if !ok && i.config.Subdomain {
					language, files, ok = i.Exists(ctx.Subdomain())
				}

				if !ok {
					// try to get by the request headers.
					if langHeader := ctx.GetHeader(acceptLanguageHeaderKey); langHeader != "" {
						idx := strings.IndexRune(langHeader, ';')
						if idx > 0 {
							langHeader = langHeader[:idx]
						}

						language, files, ok = i.Exists(langHeader)
					}
				}

				if !ok && i.config.Indentifier != nil {
					language, files, ok = i.Exists(i.config.Indentifier(ctx))
				}
			}
		}

		if !ok {
			language, files, ok = i.Exists(i.config.Default)
		}

		// if it was not taken by the cookie, then set the cookie in order to have it.
		if !wasByCookie && i.config.SetCookie && i.config.Cookie != "" {
			ctx.SetCookieKV(i.config.Cookie, language)
		}

		ctx.Values().Set(langKey, language)

		// Set iris.translate and iris.translateLang functions (they can be passed to templates as they are later on).
		ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetTranslateFunctionContextKey(), i.newTranslateLanguageFunc(files))
		ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetTranslateLangFunctionContextKey(), i.Translate)

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
		reqPath := r.URL.Path[1:]
		path := reqPath

		if idx := strings.IndexByte(path, '/'); idx > 0 {
			path = path[:idx]
		}

		if path != "" {
			if lang, _, ok := i.Exists(path); ok {
				path = r.URL.Path[len(path)+1:]
				if path == "" {
					path = "/"
				}

				r.RequestURI = path
				r.URL.Path = path
				r.Header.Set(acceptLanguageHeaderKey, lang)
				found = true
			}
		}

		if !found && i.config.Subdomain {
			host := context.GetHost(r)
			if dotIdx := strings.IndexByte(host, '.'); dotIdx > 0 {
				subdomain := host[0:dotIdx]
				if subdomain != "" {
					if lang, _, ok := i.Exists(subdomain); ok {
						host = host[dotIdx+1:]
						r.URL.Host = host
						r.Host = host
						r.Header.Set(acceptLanguageHeaderKey, lang)
					}
				}
			}
		}

		routerHandler(w, r)
	}
}

// New returns a new i18n middleware.
func New(c Config) context.Handler {
	return NewI18n(c).Handler()
}

// NewWrapper accepts a Config and returns a new router wrapper.
// The result function can be passed on `Application.WrapRouter`.
// It compares the path prefix for translated language and
// local redirects the requested path with the selected (from the path) language to the router.
//
// In order this to work as expected, it should be combined with `Application.Use(New)`
// which registers the i18n middleware itself.
func NewWrapper(c Config) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	return NewI18n(c).Wrapper()
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
