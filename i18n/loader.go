package i18n

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
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

	return load(assetNames, os.ReadFile, options)
}

// Assets accepts a function that returns a list of filenames (physical or virtual),
// another a function that should return the contents of a specific file
// and any Loader options. Go-bindata usage.
// It returns a valid `Loader` which loads and maps the locale files.
//
// See `Glob`, `FS`, `New` and `LoaderConfig` too.
func Assets(assetNames func() []string, asset func(string) ([]byte, error), options LoaderConfig) Loader {
	return load(assetNames(), asset, options)
}

// LoadFS loads the files using embed.FS or fs.FS or
// http.FileSystem or string (local directory).
// The "pattern" is a classic glob pattern.
//
// See `Glob`, `Assets`, `New` and `LoaderConfig` too.
func FS(fileSystem fs.FS, pattern string, options LoaderConfig) (Loader, error) {
	pattern = strings.TrimPrefix(pattern, "./")

	assetNames, err := fs.Glob(fileSystem, pattern)
	if err != nil {
		return nil, err
	}

	assetFunc := func(name string) ([]byte, error) {
		f, err := fileSystem.Open(name)
		if err != nil {
			return nil, err
		}

		return io.ReadAll(f)
	}

	return load(assetNames, assetFunc, options), nil
}

// LangMap key as language (e.g. "el-GR") and value as a map of key-value pairs (e.g. "hello": "Γειά").
type LangMap = map[string]map[string]interface{}

// KV is a loader which accepts a map of language(key) and the available key-value pairs.
// Example Code:
//
//	m := i18n.LangMap{
//		"en-US": map[string]interface{}{
//			"hello": "Hello",
//		},
//		"el-GR": map[string]interface{}{
//			"hello": "Γειά",
//		},
//	}
//
// app := iris.New()
// [...]
// app.I18N.LoadKV(m)
// app.I18N.SetDefault("en-US")
func KV(langMap LangMap, opts ...LoaderConfig) Loader {
	return func(m *Matcher) (Localizer, error) {
		options := DefaultLoaderConfig
		if len(opts) > 0 {
			options = opts[0]
		}

		languageIndexes := make([]int, 0, len(langMap))
		keyValuesMulti := make([]map[string]interface{}, 0, len(langMap))

		for languageName, pairs := range langMap {
			langIndex := parseLanguageName(m, languageName) // matches and adds the language tag to m.Languages.
			languageIndexes = append(languageIndexes, langIndex)
			keyValuesMulti = append(keyValuesMulti, pairs)
		}

		cat, err := internal.NewCatalog(m.Languages, options)
		if err != nil {
			return nil, err
		}

		for _, langIndex := range languageIndexes {
			if langIndex == -1 {
				// If loader has more languages than defined for use in New function,
				// e.g. when New(KV(m), "en-US") contains el-GR and en-US but only "en-US" passed.
				continue
			}

			kv := keyValuesMulti[langIndex]
			err := cat.Store(langIndex, kv)
			if err != nil {
				return nil, err
			}
		}

		if n := len(cat.Locales); n == 0 {
			return nil, fmt.Errorf("locales not found in map")
		} else if options.Strict && n < len(m.Languages) {
			return nil, fmt.Errorf("locales expected to be %d but %d parsed", len(m.Languages), n)
		}

		return cat, nil
	}
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
// See `FS`, `Glob`, `Assets` and `LoaderConfig` too.
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
