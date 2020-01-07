package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kataras/iris/v12/context"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/language"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

// LoaderConfig is an optional configuration structure which contains
// some options about how the template loader should act.
//
// See `Glob` and `Assets` package-level functions.
type LoaderConfig struct {
	// Template delimeters, defaults to {{ }}.
	Left, Right string
	// Template functions map, defaults to nil.
	FuncMap template.FuncMap
	// If true then it will return error on invalid templates instead of moving them to simple string-line keys.
	// Also it will report whether the registered languages matched the loaded ones.
	// Defaults to false.
	Strict bool
}

// LoaderOption is a type which accepts a pointer to `LoaderConfig`
// and can be optionally passed to the second variadic input argument of the `Glob` and `Assets` functions.
type LoaderOption func(*LoaderConfig)

// Glob accepts a glob pattern (see: https://golang.org/pkg/path/filepath/#Glob)
// and loads the locale files based on any "options".
//
// The "globPattern" input parameter is a glob pattern which the default loader should
// search and load for locale files.
//
// See `New` and `LoaderConfig` too.
func Glob(globPattern string, options ...LoaderOption) Loader {
	assetNames, err := filepath.Glob(globPattern)
	if err != nil {
		panic(err)
	}

	return load(assetNames, ioutil.ReadFile, options...)
}

// Assets accepts a function that returns a list of filenames (physical or virtual),
// another a function that should return the contents of a specific file
// and any Loader options. Go-bindata usage.
// It returns a valid `Loader` which loads and maps the locale files.
//
// See `Glob`, `Assets`, `New` and `LoaderConfig` too.
func Assets(assetNames func() []string, asset func(string) ([]byte, error), options ...LoaderOption) Loader {
	return load(assetNames(), asset, options...)
}

// load accepts a list of filenames (physical or virtual),
// a function that should return the contents of a specific file
// and any Loader options.
// It returns a valid `Loader` which loads and maps the locale files.
//
// See `Glob`, `Assets` and `LoaderConfig` too.
func load(assetNames []string, asset func(string) ([]byte, error), options ...LoaderOption) Loader {
	var c = LoaderConfig{
		Left:   "{{",
		Right:  "}}",
		Strict: false,
	}

	for _, opt := range options {
		opt(&c)
	}

	return func(m *Matcher) (Localizer, error) {
		languageFiles, err := m.ParseLanguageFiles(assetNames)
		if err != nil {
			return nil, err
		}

		locales := make(MemoryLocalizer)

		for langIndex, langFiles := range languageFiles {
			keyValues := make(map[string]interface{})

			for _, fileName := range langFiles {
				unmarshal := yaml.Unmarshal
				if idx := strings.LastIndexByte(fileName, '.'); idx > 1 {
					switch fileName[idx:] {
					case ".toml", ".tml":
						unmarshal = toml.Unmarshal
					case ".json":
						unmarshal = json.Unmarshal
					case ".ini":
						unmarshal = unmarshalINI
					}
				}

				b, err := asset(fileName)
				if err != nil {
					return nil, err
				}

				if err = unmarshal(b, &keyValues); err != nil {
					return nil, err
				}
			}

			var (
				templateKeys = make(map[string]*template.Template)
				lineKeys     = make(map[string]string)
				other        = make(map[string]interface{})
			)

			for k, v := range keyValues {
				// fmt.Printf("[%d] %s = %v of type: [%T]\n", langIndex, k, v, v)

				switch value := v.(type) {
				case string:
					if leftIdx, rightIdx := strings.Index(value, c.Left), strings.Index(value, c.Right); leftIdx != -1 && rightIdx > leftIdx {
						// we assume it's template?
						if t, err := template.New(k).Delims(c.Left, c.Right).Funcs(c.FuncMap).Parse(value); err == nil {
							templateKeys[k] = t
							continue
						} else if c.Strict {
							return nil, err
						}
					}

					lineKeys[k] = value
				default:
					other[k] = v
				}

			}

			t := m.Languages[langIndex]
			locales[langIndex] = &defaultLocale{
				index:        langIndex,
				id:           t.String(),
				tag:          &t,
				templateKeys: templateKeys,
				lineKeys:     lineKeys,
				other:        other,
			}
		}

		if n := len(locales); n == 0 {
			return nil, fmt.Errorf("locales not found in %s", strings.Join(assetNames, ", "))
		} else if c.Strict && n < len(m.Languages) {
			return nil, fmt.Errorf("locales expected to be %d but %d parsed", len(m.Languages), n)
		}

		return locales, nil
	}
}

// MemoryLocalizer is a map which implements the `Localizer`.
type MemoryLocalizer map[int]context.Locale

// GetLocale returns a valid `Locale` based on the "index".
func (l MemoryLocalizer) GetLocale(index int) context.Locale {
	// loc, ok := l[index]
	// if !ok {
	// 	panic(fmt.Sprintf("locale of index [%d] not found", index))
	// }
	// return loc

	return l[index]
}

// SetDefault changes the default language based on the "index".
// See `I18n#SetDefault` method for more.
func (l MemoryLocalizer) SetDefault(index int) bool {
	// callers should protect with mutex if called at serve-time.
	if loc, ok := l[index]; ok {
		f := l[0]
		l[0] = loc
		l[index] = f
		return true
	}

	return false
}

type defaultLocale struct {
	index int
	id    string
	tag   *language.Tag
	// templates *template.Template // we could use the ExecuteTemplate too.
	templateKeys map[string]*template.Template
	lineKeys     map[string]string
	other        map[string]interface{}
}

func (l *defaultLocale) Index() int {
	return l.index
}

func (l *defaultLocale) Tag() *language.Tag {
	return l.tag
}

func (l *defaultLocale) Language() string {
	return l.id
}

func (l *defaultLocale) GetMessage(key string, args ...interface{}) string {
	n := len(args)
	if n > 0 {
		// search on templates.
		if tmpl, ok := l.templateKeys[key]; ok {
			buf := new(bytes.Buffer)
			if err := tmpl.Execute(buf, args[0]); err == nil {
				return buf.String()
			}
		}
	}

	if text, ok := l.lineKeys[key]; ok {
		return fmt.Sprintf(text, args...)
	}

	if v, ok := l.other[key]; ok {
		if n > 0 {
			return fmt.Sprintf("%v [%v]", v, args)
		}
		return fmt.Sprintf("%v", v)
	}

	return ""
}

func unmarshalINI(data []byte, v interface{}) error {
	f, err := ini.Load(data)
	if err != nil {
		return err
	}

	m := *v.(*map[string]interface{})

	// Includes the ini.DefaultSection which has the root keys too.
	// We don't have to iterate to each section to find the subsection,
	// the Sections() returns all sections, sub-sections are separated by dot '.'
	// and we match the dot with a section on the translate function, so we just save the values as they are,
	// so we don't have to do section lookup on every translate call.
	for _, section := range f.Sections() {
		keyPrefix := ""
		if name := section.Name(); name != ini.DefaultSection {
			keyPrefix = name + "."
		}

		for _, key := range section.Keys() {
			m[keyPrefix+key.Name()] = key.Value()
		}
	}

	return nil
}
