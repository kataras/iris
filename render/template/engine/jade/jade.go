package jade

import (
	"github.com/Joker/jade"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/render/template/engine/html"
)

// Engine the JadeEngine
type Engine struct {
	*html.Engine
}

// new creates and returns a new JadeEngine with its configs
func New(cfg config.Template) *Engine {

	underline := &Engine{Engine: html.New(cfg)}
	underline.Middleware = func(relativeName string, fileContents string) (string, error) {
		return jade.Parse(relativeName, fileContents)
	}
	return underline
}
