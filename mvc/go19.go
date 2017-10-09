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

	// ActivatePayload contains the necessary information and the ability
	// to alt a controller's registration options, i.e the binder.
	//
	// With `ActivatePayload` the `Controller` can register custom routes
	// or modify the provided values that will be binded to the
	// controller later on.
	//
	// Look the `mvc/activator#ActivatePayload` for its implementation.
	//
	// A shortcut for the `mvc/activator#ActivatePayload`, useful when `OnActivate` is being used.
	ActivatePayload = activator.ActivatePayload
)
