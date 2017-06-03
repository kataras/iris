// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

// Options should contains the dynamic options for the engine's ExecuteWriter.
type Options interface {
	// the per-execute layout,
	// most view engines will have a static configuration field for that too.
	GetLayout() string
	// should returns the dynamic binding data, which will be used inside the template file
	GetData() interface{}
} // this Options interface is implemented inside context, in order to use one import path for all context's methods.

// Engine is the interface which all viwe engines should be implemented in order to be adapted inside Iris.
type Engine interface {
	// Load should load the templates from a directory of by binary(assets/go-bindata).
	Load() error
	// ExecuteWriter should execute a template by its filename with an optional layout and bindingData.
	ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error
	// Ext should return the final file extension which this view engine is responsible to render.
	Ext() string
}
