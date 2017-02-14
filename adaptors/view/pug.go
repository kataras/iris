package view

import (
	"github.com/Joker/jade"
)

// Pug (or Jade) returns a new kataras/go-template/pug engine.
// It shares the same exactly logic with the
// HTMLAdaptor, it uses the same exactly configuration.
// It has got some features and a lot of functions
// which will make your life easier.
// Read more about the Jade Go Template: https://github.com/Joker/jade
func Pug(directory string, extension string) *HTMLAdaptor {
	h := HTML(directory, extension)
	// because I has designed the kataras/go-template
	// so powerful, we can just pass a parser middleware
	// into the standard html template engine
	// and we're ready to go.
	h.engine.Middleware = jade.Parse
	return h
}
