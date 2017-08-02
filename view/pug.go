package view

import (
	"github.com/Joker/jade"
)

// Pug (or Jade) returns a new pug view engine.
// It shares the same exactly logic with the
// html view engine, it uses the same exactly configuration.
// It has got some features and a lot of functions
// which will make your life easier.
// Read more about the Jade Go Template: https://github.com/Joker/jade
func Pug(directory, extension string) *HTMLEngine {
	s := HTML(directory, extension)
	s.middleware = jade.Parse
	return s
}
