package iris

/* This is not idiomatic code but I did it to help users configure the Templates without need to import other packages for template configuration */
import (
	"github.com/kataras/iris/template"
	"github.com/kataras/iris/template/engine"
	"github.com/kataras/iris/template/engine/pongo"
	"github.com/kataras/iris/template/engine/standar"
)

//[ENGINE-3]
// conversions
const (
	StandarEngine EngineType = 0
	PongoEngine   EngineType = 1
)

type (
	EngineType engine.EngineType
	// TemplateConfig template.TemplateOptions
	StandarConfig standar.StandarConfig
	PongoConfig   pongo.PongoConfig

	TemplateConfig struct {
		// contains common configs for both standar & pongo
		Engine        EngineType
		Gzip          bool
		IsDevelopment bool
		Directory     string
		Extensions    []string
		ContentType   string
		Charset       string
		Asset         func(name string) ([]byte, error)
		AssetNames    func() []string
		Layout        string
		Standar       StandarConfig // contains specific configs for standar html/template
		Pongo         PongoConfig   // contains specific configs for pongo2
	}
)

func (tc *TemplateConfig) Convert() template.TemplateOptions {
	opt := template.TemplateOptions{}
	opt.Engine = engine.EngineType(tc.Engine)
	opt.Gzip = tc.Gzip
	opt.IsDevelopment = tc.IsDevelopment
	opt.Directory = tc.Directory
	opt.Extensions = tc.Extensions
	opt.ContentType = tc.ContentType
	opt.Charset = tc.Charset
	opt.Asset = tc.Asset
	opt.AssetNames = tc.AssetNames
	opt.Layout = tc.Layout
	opt.Standar = standar.StandarConfig(tc.Standar)
	opt.Pongo = pongo.PongoConfig(tc.Pongo)
	return opt
}

/* same as
&TemplateConfig{
			Engine:  engine.Standar,
			Config:  engine.Common(),
			Standar: standar.DefaultStandarConfig(),
			Pongo:   pongo.DefaultPongoConfig(),
*/
func DefaultTemplateConfig() *TemplateConfig {
	common := engine.Common()
	defaultStandar := standar.DefaultStandarConfig()
	defaultPongo := pongo.DefaultPongoConfig()

	tc := &TemplateConfig{}
	tc.Engine = StandarEngine
	tc.Gzip = common.Gzip
	tc.IsDevelopment = common.IsDevelopment
	tc.Directory = common.Directory
	tc.Extensions = common.Extensions
	tc.ContentType = common.ContentType
	tc.Charset = common.Charset
	tc.Asset = common.Asset
	tc.AssetNames = common.AssetNames
	tc.Layout = common.Layout
	tc.Standar = StandarConfig(defaultStandar)
	tc.Pongo = PongoConfig(defaultPongo)
	return tc
}

// end
