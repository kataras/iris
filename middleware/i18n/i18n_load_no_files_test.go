package i18n_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kataras/iris/v14/middleware/i18n"
)

// The locale file of https://github.com/kataras/iris/issues/2212.
const pluralsFile = `Vars:
  - Houses:
      one: "house"
      other: "houses"
  - Gender:
      "=1": "She"
      "=2": "He"

VarTemplatePlural:
  one: "${Gender} is awesome!"
  other: "other (${Gender}) has %[3]d ${Houses}."
  "=5": "{{call .InlineJoin .Names}} are awesome."
`

// With the file in the directory of its language the three messages render.
func TestLoadPluralsFile(t *testing.T) {
	fileSystem := fstest.MapFS{
		"locales/en-US/welcome.yml": {Data: []byte(pluralsFile)},
	}

	locales := i18n.New()
	if err := locales.LoadFS(fileSystem, "./locales/*/*", "en-US"); err != nil {
		t.Fatal(err)
	}

	const female = 1

	loc := locales.GetDefaultLocale()
	got := loc.GetMessage("VarTemplatePlural", map[string]any{
		"PluralCount": 5,
		"Names":       []string{"John", "Peter"},
		"InlineJoin": func(arr []string) string {
			return strings.Join(arr, ", ")
		},
	})
	got += loc.GetMessage("VarTemplatePlural", 1, female)
	got += loc.GetMessage("VarTemplatePlural", 2, female, 5)

	const expected = "John, Peter are awesome.She is awesome!other (She) has 5 houses."
	if got != expected {
		t.Fatalf("expected %q but got %q", expected, got)
	}
}

// The languages are given and not one locale file is loaded for them. Load has
// to report it. It returned nil, so the application started with no messages
// and every key rendered as an empty string.
func TestLoadWithoutLocaleFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    fstest.MapFS
		pattern  string
		expected string // the error message.
	}{
		{
			name:     "the pattern matches no file",
			files:    fstest.MapFS{},
			pattern:  "./locales/*/*",
			expected: "locales not found: no files to load",
		},
		{
			name:     "the file is one directory up",
			files:    fstest.MapFS{"locales/welcome.yml": {Data: []byte(pluralsFile)}},
			pattern:  "./locales/*/*",
			expected: "locales not found: no files to load",
		},
		{
			name:     "no language in the path of the file",
			files:    fstest.MapFS{"locales/welcome.yml": {Data: []byte(pluralsFile)}},
			pattern:  "./locales/*",
			expected: "locales not found in locales/welcome.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := i18n.New().LoadFS(tt.files, tt.pattern, "en-US")
			if err == nil {
				t.Fatalf("LoadFS(%s, en-US): expected an error but got nil", tt.pattern)
			}

			if got := err.Error(); got != tt.expected {
				t.Fatalf("LoadFS(%s, en-US): expected error %q but got %q", tt.pattern, tt.expected, got)
			}
		})
	}

	t.Run("Load", func(t *testing.T) {
		t.Chdir(t.TempDir())

		if err := i18n.New().Load("./locales/*/*", "en-US"); err == nil {
			t.Fatal("Load(./locales/*/*, en-US): expected an error but got nil")
		}
	})

	t.Run("LoadAssets", func(t *testing.T) {
		names := func() []string { return nil }
		asset := func(string) ([]byte, error) { return nil, nil }

		if err := i18n.New().LoadAssets(names, asset, "en-US"); err == nil {
			t.Fatal("LoadAssets(en-US): expected an error but got nil")
		}
	})
}

// Loader.Strict wants files for every given language.
func TestLoadStrict(t *testing.T) {
	fileSystem := fstest.MapFS{
		"locales/en-US/welcome.yml": {Data: []byte(pluralsFile)},
	}

	// Not strict, the default: a language without files is fine.
	if err := i18n.New().LoadFS(fileSystem, "./locales/*/*", "en-US", "el-GR"); err != nil {
		t.Fatalf("not strict: %v", err)
	}

	locales := i18n.New()
	locales.Loader.Strict = true

	err := locales.LoadFS(fileSystem, "./locales/*/*", "en-US", "el-GR")
	if err == nil {
		t.Fatal("strict: expected an error but got nil, el-GR has no files")
	}

	if expected, got := "locales expected to be 2 but 1 parsed", err.Error(); got != expected {
		t.Fatalf("strict: expected error %q but got %q", expected, got)
	}

	fileSystem["locales/el-GR/welcome.yml"] = &fstest.MapFile{Data: []byte(pluralsFile)}
	if err = locales.LoadFS(fileSystem, "./locales/*/*", "en-US", "el-GR"); err != nil {
		t.Fatalf("strict, every language has a file: %v", err)
	}
}
