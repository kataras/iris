package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/kataras/iris/v12/i18n/internal"

	"github.com/BurntSushi/toml"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

// LoaderConfig the configuration structure which contains
// some options about how the template loader should act.
//
// See `Glob` and `Assets` package-level functions.
type LoaderConfig = internal.Options

// Glob accepts a glob pattern (see: https://golang.org/pkg/path/filepath/#Glob)
// and loads the locale files based on any "options".
//
// The "globPattern" input parameter is a glob pattern which the default loader should
// search and load for locale files.
//
// See `New` and `LoaderConfig` too.
func Glob(globPattern string, options LoaderConfig) Loader {
	assetNames, err := filepath.Glob(globPattern)
	if err != nil {
		panic(err)
	}

	return load(assetNames, ioutil.ReadFile, options)
}

// Assets accepts a function that returns a list of filenames (physical or virtual),
// another a function that should return the contents of a specific file
// and any Loader options. Go-bindata usage.
// It returns a valid `Loader` which loads and maps the locale files.
//
// See `Glob`, `Assets`, `New` and `LoaderConfig` too.
func Assets(assetNames func() []string, asset func(string) ([]byte, error), options LoaderConfig) Loader {
	return load(assetNames(), asset, options)
}

// DefaultLoaderConfig represents the default loader configuration.
var DefaultLoaderConfig = LoaderConfig{
	Left:               "{{",
	Right:              "}}",
	Strict:             false,
	DefaultMessageFunc: nil,
	PluralFormDecoder:  internal.DefaultPluralFormDecoder,
	Funcs:              nil,
}

// load accepts a list of filenames (physical or virtual),
// a function that should return the contents of a specific file
// and any Loader options.
// It returns a valid `Loader` which loads and maps the locale files.
//
// See `Glob`, `Assets` and `LoaderConfig` too.
func load(assetNames []string, asset func(string) ([]byte, error), options LoaderConfig) Loader {
	return func(m *Matcher) (Localizer, error) {
		languageFiles, err := m.ParseLanguageFiles(assetNames)
		if err != nil {
			return nil, err
		}

		if options.DefaultMessageFunc == nil {
			options.DefaultMessageFunc = m.defaultMessageFunc
		}

		cat, err := internal.NewCatalog(m.Languages, options)
		if err != nil {
			return nil, err
		}

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

			err = cat.Store(langIndex, keyValues)
			if err != nil {
				return nil, err
			}
		}

		if n := len(cat.Locales); n == 0 {
			return nil, fmt.Errorf("locales not found in %s", strings.Join(assetNames, ", "))
		} else if options.Strict && n < len(m.Languages) {
			return nil, fmt.Errorf("locales expected to be %d but %d parsed", len(m.Languages), n)
		}

		return cat, nil
	}
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
