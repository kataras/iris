// Package i18n provides internalization and localization features for Iris.
// To use with net/http see https://github.com/kataras/i18n instead.
package i18n

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"

	"golang.org/x/text/language"
)

type (
	// Loader accepts a `Matcher` and should return a `Localizer`.
	// Functions that implement this type should load locale files.
	Loader func(m *Matcher) (Localizer, error)

	// Localizer is the interface which returned from a `Loader`.
	// Types that implement this interface should be able to retrieve a `Locale`
	// based on the language index.
	Localizer interface {
		// GetLocale should return a valid `Locale` based on the language index.
		// It will always match the Loader.Matcher.Languages[index].
		// It may return the default language if nothing else matches based on custom localizer's criteria.
		GetLocale(index int) context.Locale
	}
)

// I18n is the structure which keeps the i18n configuration and implements localization and internationalization features.
type I18n struct {
	localizer Localizer
	matcher   *Matcher

	loader Loader
	mu     sync.Mutex

	// ExtractFunc is the type signature for declaring custom logic
	// to extract the language tag name.
	// Defaults to nil.
	ExtractFunc func(ctx context.Context) string
	// If not empty, it is language identifier by url query.
	//
	// Defaults to "lang".
	URLParameter string
	// If not empty, it is language identifier by cookie of this name.
	//
	// Defaults to empty.
	Cookie string
	// If true then a subdomain can be a language identifier.
	//
	// Defaults to true.
	Subdomain bool
	// If true then it will return empty string when translation for a a specific language's key was not found.
	// Defaults to false, fallback defaultLang:key will be used.
	Strict bool

	// If true then Iris will wrap its router with the i18n router wrapper on its Build state.
	// It will (local) redirect requests like:
	// 1. /$lang_prefix/$path to /$path with the language set to $lang_prefix part.
	// 2. $lang_subdomain.$domain/$path to $domain/$path with the language set to $lang_subdomain part.
	//
	// Defaults to true.
	PathRedirect bool
}

var _ context.I18nReadOnly = (*I18n)(nil)

// makeTags converts language codes to language Tags.
func makeTags(languages ...string) (tags []language.Tag) {
	for _, lang := range languages {
		tag, err := language.Parse(lang)
		if err == nil && tag != language.Und {
			tags = append(tags, tag)
		}
	}

	return
}

// New returns a new `I18n` instance. Use its `Load` or `LoadAssets` to load languages.
func New() *I18n {
	return &I18n{
		URLParameter: "lang",
		Subdomain:    true,
		PathRedirect: true,
	}
}

// Load is a method shortcut to load files using a filepath.Glob pattern.
// It returns a non-nil error on failure.
//
// See `New` and `Glob` package-level functions for more.
func (i *I18n) Load(globPattern string, languages ...string) error {
	return i.Reset(Glob(globPattern), languages...)
}

// LoadAssets is a method shortcut to load files using go-bindata.
// It returns a non-nil error on failure.
//
// See `New` and `Asset` package-level functions for more.
func (i *I18n) LoadAssets(assetNames func() []string, asset func(string) ([]byte, error), languages ...string) error {
	return i.Reset(Assets(assetNames, asset), languages...)
}

// Reset sets the locales loader and languages.
// It is not meant to be used by users unless
// a custom `Loader` must be used instead of the default one.
func (i *I18n) Reset(loader Loader, languages ...string) error {
	tags := makeTags(languages...)

	i.loader = loader
	i.matcher = &Matcher{
		strict:    len(tags) > 0,
		Languages: tags,
		matcher:   language.NewMatcher(tags),
	}

	return i.reload()
}

// reload loads the language files from the provided Loader,
// the `New` package-level function preloads those files already.
func (i *I18n) reload() error { // May be an exported function, if requested.
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.loader == nil {
		return fmt.Errorf("nil loader")
	}

	localizer, err := i.loader(i.matcher)
	if err != nil {
		return err
	}

	i.localizer = localizer
	return nil
}

// Loaded reports whether `New` or `Load/LoadAssets` called.
func (i *I18n) Loaded() bool {
	return i != nil && i.loader != nil && i.localizer != nil && i.matcher != nil
}

// Tags returns the registered languages or dynamically resolved by files.
// Use `Load` or `LoadAssets` first.
func (i *I18n) Tags() []language.Tag {
	if !i.Loaded() {
		return nil
	}

	return i.matcher.Languages
}

// SetDefault changes the default language.
// Please avoid using this method; the default behavior will accept
// the first language of the registered tags as the default one.
func (i *I18n) SetDefault(langCode string) bool {
	t, err := language.Parse(langCode)
	if err != nil {
		return false
	}

	if tag, index, conf := i.matcher.Match(t); conf > language.Low {
		if l, ok := i.localizer.(interface {
			SetDefault(int) bool
		}); ok {
			if l.SetDefault(index) {
				tags := i.matcher.Languages
				// set the order
				tags[index] = tags[0]
				tags[0] = tag

				i.matcher.Languages = tags
				i.matcher.matcher = language.NewMatcher(tags)
				return true
			}
		}
	}

	return false
}

// Matcher implements the languae.Matcher.
// It contains the original language Matcher and keeps an ordered
// list of the registered languages for further use (see `Loader` implementation).
type Matcher struct {
	strict    bool
	Languages []language.Tag
	matcher   language.Matcher
}

var _ language.Matcher = (*Matcher)(nil)

// Match returns the best match for any of the given tags, along with
// a unique index associated with the returned tag and a confidence
// score.
func (m *Matcher) Match(t ...language.Tag) (language.Tag, int, language.Confidence) {
	return m.matcher.Match(t...)
}

// MatchOrAdd acts like Match but it checks and adds a language tag, if not found,
// when the `Matcher.strict` field is true (when no tags are provided by the caller)
// and they should be dynamically added to the list.
func (m *Matcher) MatchOrAdd(t language.Tag) (tag language.Tag, index int, conf language.Confidence) {
	tag, index, conf = m.Match(t)
	if conf <= language.Low && !m.strict {
		// not found, add it now.
		m.Languages = append(m.Languages, t)
		tag = t
		index = len(m.Languages) - 1
		conf = language.Exact
		m.matcher = language.NewMatcher(m.Languages) // reset matcher to include the new language.
	}

	return
}

// ParseLanguageFiles returns a map of language indexes and
// their associated files based on the "fileNames".
func (m *Matcher) ParseLanguageFiles(fileNames []string) (map[int][]string, error) {
	languageFiles := make(map[int][]string)

	for _, fileName := range fileNames {
		index := parsePath(m, fileName)
		if index == -1 {
			continue
		}

		languageFiles[index] = append(languageFiles[index], fileName)
	}

	return languageFiles, nil
}

func parsePath(m *Matcher, path string) int {
	if t, ok := parseLanguage(path); ok {
		if _, index, conf := m.MatchOrAdd(t); conf > language.Low {
			return index
		}
	}

	return -1
}

func parseLanguage(path string) (language.Tag, bool) {
	if idx := strings.LastIndexByte(path, '.'); idx > 0 {
		path = path[0:idx]
	}

	// path = strings.ReplaceAll(path, "..", "")

	names := strings.FieldsFunc(path, func(r rune) bool {
		return r == '_' || r == os.PathSeparator || r == '/' || r == '.'
	})

	for _, s := range names {
		t, err := language.Parse(s)
		if err != nil {
			continue
		}

		return t, true
	}

	return language.Und, false
}

// TryMatchString will try to match the "s" with a registered language tag.
// It returns -1 as the language index and false if not found.
func (i *I18n) TryMatchString(s string) (language.Tag, int, bool) {
	if tag, err := language.Parse(s); err == nil {
		if tag, index, conf := i.matcher.Match(tag); conf > language.Low {
			return tag, index, true
		}
	}

	return language.Und, -1, false
}

// Tr returns a translated message based on the "lang" language code
// and its key(format) with any optional arguments attached to it.
//
// It returns an empty string if "format" not matched.
func (i *I18n) Tr(lang, format string, args ...interface{}) string {
	_, index, ok := i.TryMatchString(lang)
	if !ok {
		index = 0
	}

	loc := i.localizer.GetLocale(index)
	if loc != nil {
		msg := loc.GetMessage(format, args...)
		if msg == "" && !i.Strict && index > 0 {
			// it's not the default/fallback language and not message found for that lang:key.
			return i.localizer.GetLocale(0).GetMessage(format, args...)
		}
		return msg
	}

	return fmt.Sprintf(format, args...)
}

const acceptLanguageHeaderKey = "Accept-Language"

// GetLocale returns the found locale of a request.
// It will return the first registered language if nothing else matched.
func (i *I18n) GetLocale(ctx context.Context) context.Locale {
	// if v := ctx.Values().Get(ctx.Application().ConfigurationReadOnly().GetLocaleContextKey()); v != nil {
	// 	if locale, ok := v.(context.Locale); ok {
	// 		return locale
	// 	}
	// }

	var (
		index int
		ok    bool
	)

	if !ok && i.ExtractFunc != nil {
		if v := i.ExtractFunc(ctx); v != "" {
			_, index, ok = i.TryMatchString(v)

		}
	}

	if !ok && i.URLParameter != "" {
		if v := ctx.URLParam(i.URLParameter); v != "" {
			_, index, ok = i.TryMatchString(v)
		}
	}

	if !ok && i.Cookie != "" {
		if v := ctx.GetCookie(i.Cookie); v != "" {
			_, index, ok = i.TryMatchString(v) // url.QueryUnescape(cookie.Value)
		}
	}

	if !ok && i.Subdomain {
		if v := ctx.Subdomain(); v != "" {
			_, index, ok = i.TryMatchString(v)
		}
	}

	if !ok {
		if v := ctx.GetHeader(acceptLanguageHeaderKey); v != "" {
			desired, _, err := language.ParseAcceptLanguage(v)
			if err == nil {
				if _, idx, conf := i.matcher.Match(desired...); conf > language.Low {
					index = idx
				}
			}
		}
	}

	// locale := i.localizer.GetLocale(index)
	// ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetLocaleContextKey(), locale)

	// // if 0 then it defaults to the first language.
	// return locale
	locale := i.localizer.GetLocale(index)
	if locale == nil {
		return nil
	}

	return locale
}

// GetMessage returns the localized text message for this "r" request based on the key "format".
func (i *I18n) GetMessage(ctx context.Context, format string, args ...interface{}) string {
	loc := i.GetLocale(ctx)
	if loc != nil {
		// it's not the default/fallback language and not message found for that lang:key.
		msg := loc.GetMessage(format, args...)
		if msg == "" && !i.Strict && loc.Index() > 0 {
			return i.localizer.GetLocale(0).GetMessage(format, args...)
		}
	}

	return fmt.Sprintf(format, args...)
}

// Wrapper returns a new router wrapper.
// The result function can be passed on `Application.WrapRouter`.
// It compares the path prefix for translated language and
// local redirects the requested path with the selected (from the path) language to the router.
//
// You do NOT have to call it manually, just set the `I18n.PathRedirect` field to true.
func (i *I18n) Wrapper() router.WrapperFunc {
	if !i.PathRedirect {
		return nil
	}
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		found := false
		path := r.URL.Path[1:]

		if idx := strings.IndexByte(path, '/'); idx > 0 {
			path = path[:idx]
		}

		if path != "" {
			if tag, _, ok := i.TryMatchString(path); ok {
				lang := tag.String()

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

		if !found && i.Subdomain {
			host := context.GetHost(r)
			if dotIdx := strings.IndexByte(host, '.'); dotIdx > 0 {
				if subdomain := host[0:dotIdx]; subdomain != "" {
					if tag, _, ok := i.TryMatchString(subdomain); ok {
						host = host[dotIdx+1:]
						r.URL.Host = host
						r.Host = host
						r.Header.Set(acceptLanguageHeaderKey, tag.String())
					}
				}
			}

		}

		next(w, r)
	}
}
