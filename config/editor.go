package config

import "github.com/imdario/mergo"

// Editor the configs for the Editor plugin
type Editor struct {
	// Host if empty used the iris server's host
	Host string
	// Port if 0 4444
	Port int
	// WorkingDir if empty "./"
	WorkingDir string
	// Username if empty iris
	Username string
	// Password if empty admin!123
	Password string
}

// DefaultEditor returns the default configs for the Editor plugin
func DefaultEditor() Editor {
	return Editor{"", 4444, "." + pathSeparator, DefaultUsername, DefaultPassword}
}

// Merge merges the default with the given config and returns the result
func (c Editor) Merge(cfg []Editor) (config Editor) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}

// MergeSingle merges the default with the given config and returns the result
func (c Editor) MergeSingle(cfg Editor) (config Editor) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
