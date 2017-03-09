package template

import (
	"os"
	"path/filepath"

	"github.com/kataras/go-errors"
)

var (
	// DefaultExtension the default file extension if empty setted
	DefaultExtension = ".html"
	// DefaultDirectory the default directory if empty setted
	DefaultDirectory = "." + string(os.PathSeparator) + "templates"
)

type (
	// Loader contains the funcs to set the location for the templates by directory or by binary
	Loader struct {
		Dir       string
		Extension string
		// AssetFn and NamesFn used when files are distributed inside the app executable
		AssetFn func(name string) ([]byte, error)
		NamesFn func() []string
	}
	// BinaryLoader optionally, called after EngineLocation's Directory, used when files are distributed inside the app executable
	// sets the AssetFn and NamesFn
	BinaryLoader struct {
		*Loader
	}
)

// NewLoader returns a default Loader which is used to load template engine(s)
func NewLoader() *Loader {
	return &Loader{Dir: DefaultDirectory, Extension: DefaultExtension}
}

// Directory sets the directory to load from
// returns the Binary location which is optional
func (t *Loader) Directory(dir string, fileExtension string) *BinaryLoader {
	if dir == "" {
		dir = DefaultDirectory // the default templates dir
	}
	if fileExtension == "" {
		fileExtension = DefaultExtension
	} else if fileExtension[0] != '.' { // if missing the start dot
		fileExtension = "." + fileExtension
	}

	t.Dir = dir

	t.Extension = fileExtension

	return &BinaryLoader{Loader: t}
}

// Binary optionally, called after Loader.Directory, used when files are distributed inside the app executable
// sets the AssetFn and NamesFn
func (t *BinaryLoader) Binary(assetFn func(name string) ([]byte, error), namesFn func() []string) {
	if assetFn == nil || namesFn == nil {
		return
	}

	t.AssetFn = assetFn
	t.NamesFn = namesFn
	// if extension is not static(setted by .Directory)
	if t.Extension == "" {
		if names := namesFn(); len(names) > 0 {
			t.Extension = filepath.Ext(names[0]) // we need the extension to get the correct template engine on the Render method
		}
	}
}

// IsBinary returns true if .Binary is called and AssetFn and NamesFn are setted
func (t *Loader) IsBinary() bool {
	return t.AssetFn != nil && t.NamesFn != nil
}

var errMissingDirectoryOrAssets = errors.New("missing Directory or Assets by binary for the template engine")

// LoadEngine receives a template Engine and calls its LoadAssets or the LoadDirectory with the loader's locations
func (t *Loader) LoadEngine(e Engine) error {
	if t.Dir == "" {
		return errMissingDirectoryOrAssets
	}

	if t.IsBinary() {
		// don't try to put abs path here
		// fixes: http://support.iris-go.com/d/22-template-binary-problem-in-v6
		return e.LoadAssets(t.Dir, t.Extension, t.AssetFn, t.NamesFn)
	}

	// fixes when user tries to execute the binary from a temp location while the templates are relatively located
	absDir, err := filepath.Abs(t.Dir)
	// panic here of course.
	if err != nil {
		panic("couldn't find the dir in the relative dir: '" + t.Dir +
			"' neither as absolute: '" + absDir + "'\n" + err.Error())
	}
	return e.LoadDirectory(absDir, t.Extension)
}
