package i18n_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/kataras/iris/v14/middleware/i18n"
)

func expectMessage(t *testing.T, locales *i18n.I18n, lang, expected string) {
	t.Helper()

	loc, ok := locales.GetLocale(lang)
	if !ok {
		t.Fatalf("%s: locale not found", lang)
	}

	if got := loc.GetMessage("hi", "iris"); got != expected {
		t.Errorf("%s: expected message %q but got %q", lang, expected, got)
	}
}

// A directory per language and a file name that is a language code for
// golang.org/x/text/language by itself, "app" among them. The directory names
// the language. See https://github.com/kataras/iris/issues/1772.
func TestLoadFileNamedAsLanguageCode(t *testing.T) {
	for _, name := range []string{"app", "api", "nav", "tag", "log", "msg"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()

			for lang, content := range map[string]string{
				"en-US": "hi = hello, %s\n",
				"el-GR": "hi = γεια, %s\n",
			} {
				langDir := filepath.Join(dir, "locales", lang)
				if err := os.MkdirAll(langDir, 0o755); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(filepath.Join(langDir, name+".ini"), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			t.Chdir(dir)

			locales := i18n.New()
			if err := locales.Load("./locales/*/*.ini", "en-US", "el-GR"); err != nil {
				t.Fatal(err)
			}

			expectMessage(t, locales, "en-US", "hello, iris")
			expectMessage(t, locales, "el-GR", "γεια, iris")
		})
	}
}

// The same layout through LoadFS, where the paths are slash-separated on every
// system.
func TestLoadFSFileNamedAsLanguageCode(t *testing.T) {
	fileSystem := fstest.MapFS{
		"locales/en-US/app.ini": {Data: []byte("hi = hello, %s\n")},
		"locales/el-GR/app.ini": {Data: []byte("hi = γεια, %s\n")},
	}

	locales := i18n.New()
	if err := locales.LoadFS(fileSystem, "./locales/*/*.ini", "en-US", "el-GR"); err != nil {
		t.Fatal(err)
	}

	expectMessage(t, locales, "en-US", "hello, iris")
	expectMessage(t, locales, "el-GR", "γεια, iris")
}

// Only the directory the file is in may name its language. The zh-CN file is
// not registered and must not end up in el-GR because of the "el" directory
// further up.
func TestLoadFileOfUnregisteredLanguageStaysOut(t *testing.T) {
	fileSystem := fstest.MapFS{
		"el/locales/en-US/app.ini": {Data: []byte("hi = hello, %s\n")},
		"el/locales/el-GR/app.ini": {Data: []byte("hi = γεια, %s\n")},
		"el/locales/zh-CN/app.ini": {Data: []byte("hi = 您好，%s\n")},
	}

	locales := i18n.New()
	if err := locales.LoadFS(fileSystem, "./el/locales/*/*.ini", "en-US", "el-GR"); err != nil {
		t.Fatal(err)
	}

	expectMessage(t, locales, "en-US", "hello, iris")
	expectMessage(t, locales, "el-GR", "γεια, iris")
}
