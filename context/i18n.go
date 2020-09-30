package context

import "golang.org/x/text/language"

// I18nReadOnly is the interface which ontains the read-only i18n features.
// Read the "i18n" package fo details.
type I18nReadOnly interface {
	Tags() []language.Tag
	GetLocale(ctx *Context) Locale
	Tr(lang string, key string, args ...interface{}) string
	TrContext(ctx *Context, key string, args ...interface{}) string
}

// Locale is the interface which returns from a `Localizer.GetLocale` method.
// It serves the translations based on "key" or format. See `GetMessage`.
type Locale interface {
	// Index returns the current locale index from the languages list.
	Index() int
	// Tag returns the full language Tag attached tothis Locale,
	// it should be uniue across different Locales.
	Tag() *language.Tag
	// Language should return the exact languagecode of this `Locale`
	//that the user provided on `New` function.
	//
	// Same as `Tag().String()` but it's static.
	Language() string
	// GetMessage should return translated text based on the given "key".
	GetMessage(key string, args ...interface{}) string
}
