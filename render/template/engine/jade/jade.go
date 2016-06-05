// Package jade the JadeEngine's functionality lives inside ../html now
package jade

import (
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/render/template/engine/html"
)

// New creates and returns the HTMLTemplate template engine
func New(c config.Template) *html.Engine {
	// copy the Jade to the HTMLTemplate
	c.HTMLTemplate = config.HTMLTemplate(c.Jade)
	s := &html.Engine{Config: &c}
	return s
}
