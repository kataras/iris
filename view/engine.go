package view

import (
	"io"
)

// NoLayout disables the configuration's layout for a specific execution.
const NoLayout = "iris.nolayout"

// returns empty if it's no layout or empty layout and empty configuration's layout.
func getLayout(layout string, globalLayout string) string {
	if layout == NoLayout {
		return ""
	}

	if layout == "" && globalLayout != "" {
		return globalLayout
	}

	return layout
}

// Engine is the interface which all view engines should be implemented in order to be registered inside iris.
type Engine interface {
	// Load should load the templates from a directory of by binary(assets/go-bindata).
	Load() error
	// ExecuteWriter should execute a template by its filename with an optional layout and bindingData.
	ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error
	// Ext should return the final file extension which this view engine is responsible to render.
	Ext() string
}
