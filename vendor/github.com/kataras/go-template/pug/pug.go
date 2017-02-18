//Package pug the JadeEngine's functionality lives inside ../html now
package pug

import (
	"github.com/Joker/jade"
	"github.com/kataras/go-template/html"
)

// New creates and returns the Pug template engine
func New(cfg ...Config) *html.Engine {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	// pass the Pug/Jade configs to the html's configuration

	s := html.New(html.Config(c))
	s.Middleware = jade.Parse
	return s
}
