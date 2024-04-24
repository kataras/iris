package internal

import (
	"fmt"
	"text/template"

	"github.com/kataras/iris/v12/context"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

// MessageFunc is the function type to modify the behavior when a key or language was not found.
// All language inputs fallback to the default locale if not matched.
// This is why this signature accepts both input and matched languages, so caller
// can provide better messages.
//
// The first parameter is set to the client real input of the language,
// the second one is set to the matched language (default one if input wasn't matched)
// and the third and forth are the translation format/key and its optional arguments.
//
// Note: we don't accept the Context here because Tr method and template func {{ tr }}
// have no direct access to it.
type MessageFunc func(langInput, langMatched, key string, args ...interface{}) string

// Catalog holds the locales and the variables message storage.
type Catalog struct {
	builder *catalog.Builder
	Locales []*Locale
}

// The Options of the Catalog and its Locales.
type Options struct {
	// Left delimiter for template messages.
	Left string
	// Right delimeter for template messages.
	Right string
	// Enable strict mode.
	Strict bool
	// Optional functions for template messages per locale.
	Funcs func(context.Locale) template.FuncMap
	// Optional function to be called when no message was found.
	DefaultMessageFunc MessageFunc
	// Customize the overall behavior of the plurazation feature.
	PluralFormDecoder PluralFormDecoder
}

// NewCatalog returns a new Catalog based on the registered languages and the loader options.
func NewCatalog(languages []language.Tag, opts Options) (*Catalog, error) { // ordered languages, the first should be the default one.
	if len(languages) == 0 {
		return nil, fmt.Errorf("catalog: empty languages")
	}

	if opts.Left == "" {
		opts.Left = "{{"
	}

	if opts.Right == "" {
		opts.Right = "}}"
	}

	if opts.PluralFormDecoder == nil {
		opts.PluralFormDecoder = DefaultPluralFormDecoder
	}

	builder := catalog.NewBuilder(catalog.Fallback(languages[0]))

	locales := make([]*Locale, 0, len(languages))
	for idx, tag := range languages {
		locale := &Locale{
			tag:      tag,
			index:    idx,
			ID:       tag.String(),
			Options:  opts,
			Printer:  message.NewPrinter(tag, message.Catalog(builder)),
			Messages: make(map[string]Renderer),
		}
		locale.FuncMap = getFuncs(locale)

		locales = append(locales, locale)
	}

	c := &Catalog{
		builder: builder,
		Locales: locales,
	}

	return c, nil
}

// Set sets a simple translation message.
func (c *Catalog) Set(tag language.Tag, key string, msgs ...catalog.Message) error {
	// fmt.Printf("Catalog.Set[%s] %s:\n", tag.String(), key)
	// for _, msg := range msgs {
	// 	fmt.Printf("%#+v\n", msg)
	// }
	return c.builder.Set(tag, key, msgs...)
}

// Store stores the a map of values to the locale derives from the given "langIndex".
func (c *Catalog) Store(langIndex int, kv Map) error {
	loc := c.getLocale(langIndex)
	if loc == nil {
		return fmt.Errorf("expected language index to be lower or equal than %d but got %d", len(c.Locales), langIndex)
	}
	return loc.Load(c, kv)
}

/* Localizer interface. */

// SetDefault changes the default language based on the "index".
// See `I18n#SetDefault` method for more.
func (c *Catalog) SetDefault(index int) bool {
	if index < 0 {
		index = 0
	}

	if maxIdx := len(c.Locales) - 1; index > maxIdx {
		return false
	}

	// callers should protect with mutex if called at serve-time.
	loc := c.Locales[index]
	loc.index = 0
	f := c.Locales[0]
	c.Locales[0] = loc
	f.index = index
	c.Locales[index] = f
	return true
}

// GetLocale returns a valid `Locale` based on the "index".
func (c *Catalog) GetLocale(index int) context.Locale {
	return c.getLocale(index)
}

func (c *Catalog) getLocale(index int) *Locale {
	if index < 0 {
		index = 0
	}

	if maxIdx := len(c.Locales) - 1; index > maxIdx {
		// panic("expected language index to be lower or equal than %d but got %d", maxIdx, langIndex)
		return nil
	}

	loc := c.Locales[index]
	return loc
}
