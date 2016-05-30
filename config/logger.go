package config

import "github.com/imdario/mergo"

import (
	"io"
	"os"
)

var (
	// TimeFormat default time format for any kind of datetime parsing
	TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
)

type (
	Logger struct {
		Out    io.Writer
		Prefix string
		Flag   int
	}
)

func DefaultLogger() Logger {
	return Logger{Out: os.Stdout, Prefix: "", Flag: 0}
}

// Merge merges the default with the given config and returns the result
func (c Logger) Merge(cfg []Logger) (config Logger) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}

// Merge MergeSingle the default with the given config and returns the result
func (c Logger) MergeSingle(cfg Logger) (config Logger) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
