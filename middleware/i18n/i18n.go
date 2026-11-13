// Package i18n provides internalization and localization features for Iris.
//
// It has no single Handler entry point: New() returns an *I18n that loads and
// matches the locales and is registered through Application.I18n, because the
// same value also resolves translations outside a request. `NewHandler` and
// `NewRouter` are the two lower-level Handlers for an existing LocaleMatcher.
// To use with net/http see https://github.com/kataras/i18n instead.
package i18n

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kataras/iris/v14/middleware/i18n/locale"

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
		GetLocale(index int) locale.Locale
	}
)

// I18n is the structure which keeps the i18n configuration and implements localization and internationalization features.
type I18n struct {
	localizer Localizer
	matcher   *Matcher

	Loader LoaderConfig
	loader Loader
	mu     sync.Mutex
}

var _ LocaleMatcher = (*I18n)(nil)

// makeTags converts language codes to language Tags.
func makeTags(languages ...string) (tags []language.Tag) {
	languages = removeDuplicates(languages)

	for _, lang := range languages {
		tag, err := language.Parse(lang)
		if err == nil && tag != language.Und {
			tags = append(tags, tag)
		}
	}

	return
}

// New returns a new `I18n` instance. Use its `Load` or `LoadAssets` to load languages.
// Example at: https://github.com/kataras/iris/tree/main/_examples/06-middleware/i18n.
func New() *I18n {
	i := &I18n{
		Loader: DefaultLoaderConfig,
	}

	return i
}

// SetDefaultMessageFunc sets the default message function for unknown keys in the matched language.
func (i *I18n) SetDefaultMessageFunc(fn func(langMatched, key string, args ...any) string) {
	i.Loader.DefaultMessageFunc = fn
}

// Load is a method shortcut to load files using a filepath.Glob pattern.
// It returns a non-nil error on failure.
//
// See `New` and `Glob` package-level functions for more.
func (i *I18n) Load(globPattern string, languages ...string) error {
	return i.Reset(Glob(globPattern, i.Loader), languages...)
}

// LoadAssets is a method shortcut to load files using go-bindata.
// It returns a non-nil error on failure.
//
// See `New` and `Assets` package-level functions for more.
func (i *I18n) LoadAssets(assetNames func() []string, asset func(string) ([]byte, error), languages ...string) error {
	return i.Reset(Assets(assetNames, asset, i.Loader), languages...)
}

// LoadFS is a method shortcut to load files using
// an `embed.FS` or `fs.FS` or `http.FileSystem` value.
// The "pattern" is a classic glob pattern.
//
// See `New` and `FS` package-level functions for more.
// Example: https://github.com/kataras/iris/blob/main/_examples/06-middleware/i18n/main.go.
func (i *I18n) LoadFS(fileSystem fs.FS, pattern string, languages ...string) error {
	loader, err := FS(fileSystem, pattern, i.Loader)
	if err != nil {
		return err
	}

	return i.Reset(loader, languages...)
}

// LoadKV is a method shortcut to load locales from a map of specified languages.
// See `KV` package-level function for more.
func (i *I18n) LoadKV(langMap LangMap, languages ...string) error {
	loader := KV(langMap, i.Loader)
	return i.Reset(loader, languages...)
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
// when the `Matcher.strict` field is false (i.e. when no language tags were provided
// by the caller) and they should be dynamically added to the list.
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

		// Not a registered language. A file name such as "app", "api" or "tag"
		// is a valid language code by itself, so give the directory of the
		// file a try: locales/en-US/app.ini belongs to en-US.
		if t, err := language.Parse(filepath.Base(filepath.Dir(path))); err == nil {
			if _, index, conf := m.Match(t); conf > language.Low {
				return index
			}
		}
	}

	return -1
}

func parseLanguageName(m *Matcher, name string) int {
	if t, err := language.Parse(name); err == nil {
		if _, index, conf := m.MatchOrAdd(t); conf > language.Low {
			return index
		}
	}

	return -1
}

func reverseStrings(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func parseLanguage(path string) (language.Tag, bool) {
	if idx := strings.LastIndexByte(path, '.'); idx > 0 {
		path = path[0:idx]
	}

	// path = strings.ReplaceAll(path, "..", "")

	names := strings.FieldsFunc(path, func(r rune) bool {
		return r == '_' || r == os.PathSeparator || r == '/' || r == '.'
	})

	names = reverseStrings(names) // see https://github.com/kataras/i18n/issues/1

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

// GetLocaleByIndex returns the locale based on the given index.
func (i *I18n) GetLocaleByIndex(index int) (locale.Locale, bool) {
	if locale := i.localizer.GetLocale(index); locale != nil {
		return locale, true
	}

	return nil, false
}

// GetLocale returns the matched locale based on the "s" accept-language header.
func (i *I18n) GetLocale(s string) (locale.Locale, bool) {
	index := -1
	switch s {
	case "":
		return nil, false
	case "default":
		index = 0
	default:
		// _, index, _ = i.TryMatchString(s)
		if desired, _, err := language.ParseAcceptLanguage(s); err == nil {
			if _, idx, conf := i.matcher.Match(desired...); conf > language.Low {
				index = idx
			}
		}
	}

	return i.GetLocaleByIndex(index)
}

// GetDefaultLocale returns the default locale.
func (i *I18n) GetDefaultLocale() locale.Locale {
	if defaultLocale, ok := i.GetLocaleByIndex(0); ok {
		return defaultLocale
	}

	return nil
}

func removeDuplicates(elements []string) (result []string) {
	seen := make(map[string]struct{})

	for v := range elements {
		val := elements[v]
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	return result
}
