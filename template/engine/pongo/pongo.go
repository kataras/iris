package pongo

/* TODO:
1. Find if pongo2 supports layout, it should have extends or something like django but I don't know yet, if exists then do something with the layour parameter in Exeucte/Gzip.

*/
import (
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/template/engine"
	"github.com/kataras/iris/utils"
)

var (
	buffer *utils.BufferPool
)

type (
	PongoConfig struct {
		// Filters for pongo2, map[name of the filter] the filter function . The filters are auto register
		Filters map[string]pongo2.FilterFunction
	}

	Config struct {
		engine.Config
		PongoConfig
	}

	Engine struct {
		Config    *Config
		Templates *pongo2.TemplateSet
	}
)

func WrapConfig(common engine.Config, pongo PongoConfig) *Config {
	return &Config{Config: common, PongoConfig: pongo}
}

// DefaultPongoConfig returns the default pongo specific options, no the whole Config
func DefaultPongoConfig() PongoConfig {
	return PongoConfig{Filters: make(map[string]pongo2.FilterFunction, 0)}
}

func New(config *Config) *Engine {
	if buffer == nil {
		buffer = utils.NewBufferPool(64)
	}
	if config == nil {
		config = WrapConfig(engine.Common(), DefaultPongoConfig())
	}
	return &Engine{Config: config}
}

func (p *Engine) GetConfig() *engine.Config {
	return &p.Config.Config
}

func (p *Engine) BuildTemplates() error {
	// Add our filters. first
	for k, v := range p.Config.Filters {
		pongo2.RegisterFilter(k, v)
	}
	if p.Config.Asset == nil || p.Config.AssetNames == nil {
		return p.buildFromDir()

	}
	return p.buildFromAsset()

}

func (p *Engine) buildFromDir() error {
	if p.Config.Directory == "" {
		return nil //we don't return fill error here(yet)
	}

	var templateErr error
	dir := p.Config.Directory
	fsLoader, err := pongo2.NewLocalFileSystemLoader(dir) // I see that this doesn't read the content if already parsed, so do it manually via filepath.Walk
	if err != nil {
		return err
	}

	p.Templates = pongo2.NewSet("", fsLoader)

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

				_, err := p.Templates.FromFile(rel) // use Relative, no from path because it calculates the basedir of the fsLoader: /templates/templates/index.html
				//if that doesn't works then do tmpl, err..; p.Templates = tmpl
				if err != nil {
					templateErr = err
					break
				}
				break
			}
		}
		return nil
	})

	return templateErr
}

func (p *Engine) buildFromAsset() error {
	var templateErr error
	dir := p.Config.Directory
	fsLoader, err := pongo2.NewLocalFileSystemLoader(dir)
	if err != nil {
		return err
	}
	p.Templates = pongo2.NewSet("", fsLoader)
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
				_, err = p.Templates.FromString(string(buf)) // I don't konw if that will work, yet
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

func (p *Engine) Execute(ctx context.IContext, name string, binding interface{}, layout string) error {
	// get the template from cache, I never used pongo2 but I think reading its code helps me to understand that this is the best way to do it with the best performance.
	tmpl, err := p.Templates.FromCache(name)
	if err != nil {
		return err
	}
	// Retrieve a buffer from the pool to write to.
	out := buffer.Get()

	err = tmpl.ExecuteWriter(getPongoContext(binding), out)

	if err != nil {
		buffer.Put(out)
		return err
	}
	w := ctx.GetRequestCtx().Response.BodyWriter()
	out.WriteTo(w)

	// Return the buffer to the pool.
	buffer.Put(out)
	return nil
}

func (p *Engine) ExecuteGzip(ctx context.IContext, name string, binding interface{}, layout string) error {
	tmpl, err := p.Templates.FromCache(name)
	if err != nil {
		return err
	}
	// Retrieve a buffer from the pool to write to.
	out := gzip.NewWriter(ctx.GetRequestCtx().Response.BodyWriter())
	err = tmpl.ExecuteWriter(getPongoContext(binding), out)

	if err != nil {
		return err
	}
	//out.Flush()
	out.Close()
	ctx.GetRequestCtx().Response.Header.Add("Content-Encoding", "gzip")

	return nil
}
