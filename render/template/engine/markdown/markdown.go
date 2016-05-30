package markdown

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fmt"

	"github.com/kataras/iris/config"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// Supports RAW markdown only, no context binding or layout, to use dynamic markdown with other template engine use the context.Markdown/MarkdownString
type (
	Engine struct {
		Config        *config.Template
		templateCache map[string][]byte
		mu            sync.Mutex
	}
)

// New creates and returns a Pongo template engine
func New(c config.Template) *Engine {
	return &Engine{Config: &c, templateCache: make(map[string][]byte)}
}

func (e *Engine) BuildTemplates() error {
	if e.Config.Asset == nil || e.Config.AssetNames == nil {
		return e.buildFromDir()
	}
	return e.buildFromAsset()

}

func (e *Engine) buildFromDir() (templateErr error) {
	if e.Config.Directory == "" {
		return nil //we don't return fill error here(yet)
	}
	dir := e.Config.Directory

	// Walk the supplied directory and compile any files that match our extension list.
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {

		if info == nil || info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		ext := ""
		if strings.Index(rel, ".") != -1 {
			ext = filepath.Ext(rel)
		}

		for _, extension := range e.Config.Extensions {
			if ext == extension {
				buf, err := ioutil.ReadFile(path)
				if err != nil {
					templateErr = err
					break
				}

				buf = blackfriday.MarkdownCommon(buf)
				if e.Config.Markdown.Sanitize {
					buf = bluemonday.UGCPolicy().SanitizeBytes(buf)
				}

				if err != nil {
					templateErr = err
					break
				}
				name := filepath.ToSlash(rel)
				e.templateCache[name] = buf
				break
			}
		}
		return nil
	})

	return nil
}

func (e *Engine) buildFromAsset() error {
	var templateErr error
	dir := e.Config.Directory
	for _, path := range e.Config.AssetNames() {
		if !strings.HasPrefix(path, dir) {
			continue
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			panic(err)
		}

		ext := ""
		if strings.Index(rel, ".") != -1 {
			ext = "." + strings.Join(strings.Split(rel, ".")[1:], ".")
		}

		for _, extension := range e.Config.Extensions {
			if ext == extension {

				buf, err := e.Config.Asset(path)
				if err != nil {
					templateErr = err
					break
				}
				b := blackfriday.MarkdownCommon(buf)
				if e.Config.Markdown.Sanitize {
					b = bluemonday.UGCPolicy().SanitizeBytes(b)
				}
				name := filepath.ToSlash(rel)
				e.templateCache[name] = b
				break
			}
		}
	}
	return templateErr
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

// layout here is unnesecery
func (e *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, layout string) error {
	if tmpl := e.fromCache(name); tmpl != nil {
		_, err := out.Write(tmpl)
		return err
	}

	return fmt.Errorf("[IRIS TEMPLATES] Template with name %s doesn't exists in the dir %s", name, e.Config.Directory)
}
