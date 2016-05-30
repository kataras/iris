package pongo

/* TODO:
1. Find if pongo2 supports layout, it should have extends or something like django but I don't know yet, if exists then do something with the layour parameter in Exeucte/Gzip.

*/
import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fmt"

	"github.com/flosch/pongo2"
	"github.com/kataras/iris/config"
)

type (
	Engine struct {
		Config        *config.Template
		templateCache map[string]*pongo2.Template
		mu            sync.Mutex
	}
)

// New creates and returns a Pongo template engine
func New(c config.Template) *Engine {
	return &Engine{Config: &c, templateCache: make(map[string]*pongo2.Template)}
}

func (p *Engine) BuildTemplates() error {
	// Add our filters. first
	for k, v := range p.Config.Pongo.Filters {
		pongo2.RegisterFilter(k, v)
	}
	if p.Config.Asset == nil || p.Config.AssetNames == nil {
		return p.buildFromDir()

	}
	return p.buildFromAsset()

}

func (p *Engine) buildFromDir() (templateErr error) {
	if p.Config.Directory == "" {
		return nil //we don't return fill error here(yet)
	}
	dir := p.Config.Directory

	fsLoader, err := pongo2.NewLocalFileSystemLoader(dir) // I see that this doesn't read the content if already parsed, so do it manually via filepath.Walk
	if err != nil {
		return err
	}

	set := pongo2.NewSet("", fsLoader)
	set.Globals = getPongoContext(p.Config.Pongo.Globals)
	// Walk the supplied directory and compile any files that match our extension list.
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// Fix same-extension-dirs bug: some dir might be named to: "users.tmpl", "local.html".
		// These dirs should be excluded as they are not valid golang templates, but files under
		// them should be treat as normal.
		// If is a dir, return immediately (dir is not a valid golang template).
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

		for _, extension := range p.Config.Extensions {
			if ext == extension {
				buf, err := ioutil.ReadFile(path)
				if err != nil {
					templateErr = err
					break
				}
				if err != nil {
					templateErr = err
					break
				}
				name := filepath.ToSlash(rel)
				p.templateCache[name], templateErr = set.FromString(string(buf))

				//_, templateErr = p.Templates.FromCache(rel) // use Relative, no from path because it calculates the basedir of the fsLoader
				if templateErr != nil {
					return templateErr // break the file walk(;)
				}
				break
			}
		}
		return nil
	})

	return
}

func (p *Engine) buildFromAsset() error {
	var templateErr error
	dir := p.Config.Directory
	fsLoader, err := pongo2.NewLocalFileSystemLoader(dir)
	if err != nil {
		return err
	}
	set := pongo2.NewSet("", fsLoader)
	set.Globals = getPongoContext(p.Config.Pongo.Globals)
	for _, path := range p.Config.AssetNames() {
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

		for _, extension := range p.Config.Extensions {
			if ext == extension {

				buf, err := p.Config.Asset(path)
				if err != nil {
					templateErr = err
					break
				}
				name := filepath.ToSlash(rel)
				p.templateCache[name], err = set.FromString(string(buf)) // I don't konw if that will work, yet
				if err != nil {
					templateErr = err
					break
				}
				break
			}
		}
	}
	return templateErr
}

// getPongoContext returns the pongo2.Context from map[string]interface{} or from pongo2.Context, used internaly
func getPongoContext(templateData interface{}) pongo2.Context {
	if templateData == nil {
		return nil
	}

	if v, isMap := templateData.(map[string]interface{}); isMap {
		return v
	}

	if contextData, isPongoContext := templateData.(pongo2.Context); isPongoContext {
		return contextData
	}

	return nil
}

func (p *Engine) fromCache(relativeName string) *pongo2.Template {
	p.mu.Lock()

	tmpl, ok := p.templateCache[relativeName]

	if ok {
		p.mu.Unlock() // defer is slow
		return tmpl
	}
	p.mu.Unlock() // defer is slow
	return nil
}

// layout here is unnesecery
func (p *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, layout string) error {
	if tmpl := p.fromCache(name); tmpl != nil {
		return tmpl.ExecuteWriter(getPongoContext(binding), out)
	}

	return fmt.Errorf("[IRIS TEMPLATES] Template with name %s doesn't exists in the dir %s", name, p.Config.Directory)
}
