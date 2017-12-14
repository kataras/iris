package markdown

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// Supports RAW markdown only, no context binding or layout, to use dynamic markdown with other template engine use the context.Markdown/MarkdownString

type (
	// Engine the jade engine
	Engine struct {
		Config        Config
		templateCache map[string][]byte
		mu            sync.Mutex
	}
)

// New creates and returns a Pongo template engine
func New(cfg ...Config) *Engine {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	return &Engine{Config: c, templateCache: make(map[string][]byte)}
}

// LoadDirectory builds the markdown templates
func (e *Engine) LoadDirectory(dir string, extension string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	var templateErr error
	// Walk the supplied directory and compile any files that match our extension list.
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {

		if info == nil || info.IsDir() {

		} else {

			rel, err := filepath.Rel(dir, path)
			if err != nil {
				templateErr = err
				return err
			}

			ext := filepath.Ext(rel)

			if ext == extension {
				buf, err := ioutil.ReadFile(path)
				if err != nil {
					templateErr = err
					return err
				}

				buf = blackfriday.MarkdownCommon(buf)
				if e.Config.Sanitize {
					buf = bluemonday.UGCPolicy().SanitizeBytes(buf)
				}

				if err != nil {
					templateErr = err
					return err
				}
				name := filepath.ToSlash(rel)
				e.templateCache[name] = buf

			}
		}
		return nil
	})

	return templateErr

}

// LoadAssets loads the templates by binary
func (e *Engine) LoadAssets(virtualDirectory string, virtualExtension string, assetFn func(name string) ([]byte, error), namesFn func() []string) error {
	if len(virtualDirectory) > 0 {
		if virtualDirectory[0] == '.' { // first check for .wrong
			virtualDirectory = virtualDirectory[1:]
		}
		if virtualDirectory[0] == '/' || virtualDirectory[0] == os.PathSeparator { // second check for /something, (or ./something if we had dot on 0 it will be removed
			virtualDirectory = virtualDirectory[1:]
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	names := namesFn()
	for _, path := range names {
		if !strings.HasPrefix(path, virtualDirectory) {
			continue
		}

		rel, err := filepath.Rel(virtualDirectory, path)
		if err != nil {
			return err
		}

		ext := filepath.Ext(rel)
		if ext == virtualExtension {

			buf, err := assetFn(path)
			if err != nil {
				return err
			}
			b := blackfriday.MarkdownCommon(buf)
			if e.Config.Sanitize {
				b = bluemonday.UGCPolicy().SanitizeBytes(b)
			}
			name := filepath.ToSlash(rel)
			e.templateCache[name] = b

		}

	}
	return nil
}

func (e *Engine) fromCache(relativeName string) []byte {
	e.mu.Lock()

	tmpl, ok := e.templateCache[relativeName]

	if ok {
		e.mu.Unlock() // defer is slow
		return tmpl
	}
	e.mu.Unlock() // defer is slow
	return nil
}

// ExecuteWriter executes a templates and write its results to the out writer
// layout here is useless
func (e *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) error {
	if tmpl := e.fromCache(name); tmpl != nil {
		_, err := out.Write(tmpl)
		return err
	}

	return fmt.Errorf("[IRIS TEMPLATES] Template with name %s doesn't exists in the dir", name)
}

// ExecuteRaw receives, parse and executes raw source template contents
// it's super-simple function without options and funcs, it's not used widely
// implements the EngineRawExecutor interface
func (e *Engine) ExecuteRaw(src string, wr io.Writer, binding interface{}) (err error) {
	parsed := blackfriday.MarkdownCommon([]byte(src))
	if e.Config.Sanitize {
		parsed = bluemonday.UGCPolicy().SanitizeBytes(parsed)
	}
	_, err = wr.Write(parsed)
	return
}
