package pug

import "github.com/kataras/go-template/html"

// Pug is the 'jade', same configs as the html engine

// Config for pug template engine
type Config html.Config

// DefaultConfig returns the default configuration for the pug(jade) template engine
func DefaultConfig() Config {
	return Config(html.DefaultConfig())
}
