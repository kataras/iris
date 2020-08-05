package context

import "io"

// ViewEngine is the interface which all view engines should be implemented in order to be registered inside iris.
type ViewEngine interface {
	// Load should load the templates from a physical system directory or by an embedded one (assets/go-bindata).
	Load() error
	// ExecuteWriter should execute a template by its filename with an optional layout and bindingData.
	ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error
	// Ext should return the final file extension which this view engine is responsible to render.
	Ext() string
}

// ViewEngineFuncer is an addition of a view engine,
// if a view engine implements that interface
// then iris can add some closed-relative iris functions
// like {{ url }}, {{ urlpath }} and {{ tr }}.
type ViewEngineFuncer interface {
	// AddFunc should adds a function to the template's function map.
	AddFunc(funcName string, funcBody interface{})
}
