package engine

import (
	"github.com/kataras/iris/context"
)

type (
	EngineType uint8

	Engine interface {
		BuildTemplates() error
		Execute(ctx context.IContext, name string, binding interface{}, layout string) error
		ExecuteGzip(ctx context.IContext, name string, binding interface{}, layout string) error
	}

	// I tried a lot of code styles and patterns for more than 9 hours, this is the only way that will be easier for users. Do not try to pr here I will kill you <3
	// Config the common configs for all parsers/engines
	Config struct {
		Gzip          bool
		IsDevelopment bool
		Directory     string
		Extensions    []string
		ContentType   string
		Charset       string
		Asset         func(name string) ([]byte, error)
		AssetNames    func() []string
		Layout        string
	}
)

const (
	Standar EngineType = 0
	Pongo   EngineType = 1
)

func Common() Config {
	return Config{
		Gzip:          false,
		IsDevelopment: false,
		Directory:     "templates",
		Extensions:    []string{".html"},
		ContentType:   "text/html",
		Charset:       "UTF-8",
		Layout:        "", // currently this is the only config which not working for pongo2 yet but I will find a way
	}

	// although I could add the StandarConfig  & PongoConfig here and make it more easier but I dont want, keep the things in their packages
}
