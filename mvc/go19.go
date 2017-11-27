// +build go1.9

package mvc

import (
	"html/template"

	"github.com/kataras/iris/mvc/activator"
)

type (
	// HTML wraps the "s" with the template.HTML
	// in order to be marked as safe content, to be rendered as html and not escaped.
	HTML = template.HTML

	// TController contains the necessary controller's pre-serve information.
	//
	// With `TController` the `Controller` can register custom routes
	// or modify the provided values that will be binded to the
	// controller later on.
	//
	// Look the `mvc/activator#TController` for its implementation.
	//
	// A shortcut for the `mvc/activator#TController`, useful when `OnActivate` is being used.
	TController = activator.TController
)
