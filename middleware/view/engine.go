package view

import (
	"fmt"
	"io"
)

type (
	// Engine is the interface which all view engines should be implemented in order to be registered inside iris.
	Engine interface {
		// ExecuteTemplate should execute a template by its filename with an optional layout and data.
		ExecuteTemplate(w io.Writer, templateName string, layoutName string, data any) error
	}

	// EngineFuncer is the interface which supports the registration of custom template functions.
	EngineFuncer interface {
		// AddFunc should adds a function to the template's function map.
		AddFunc(funcName string, funcBody any)
	}

	// EngineExtenser is the interface which supports the retrieval of the template files extension.
	EngineExtenser interface {
		// Ext should return the final file extension (including the dot)
		// which this view engine is responsible to render.
		Ext() string
	}
)

// ErrTemplateNotFound it's an error.
// It reports whether a template was not found in the parsed templates tree.
type ErrTemplateNotFound struct {
	Name     string
	IsLayout bool
	Data     any
}

// Error completes the `error` interface.
func (e ErrTemplateNotFound) Error() string {
	title := "template"
	if e.IsLayout {
		title = "layout"
	}
	return fmt.Sprintf("%s '%s' does not exist", title, e.Name)
}
