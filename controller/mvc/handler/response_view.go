package handler

import (
	"errors"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/view"
)

// View returns a new View Builder.
// It can be used as a Response value.
func View() *ViewBuilder {
	return &ViewBuilder{}
}

// ViewBuilder is the view builder.
// It's used to build a view.
// It's used by the `View` package-level function.
type ViewBuilder struct {
	statusCode int
	layout     string
	name       string
	data       any
}

// StatusCode sets the status code for the response.
func (v *ViewBuilder) StatusCode(code int) *ViewBuilder {
	v.statusCode = code
	return v
}

// Layout sets the layout for the view.
func (v *ViewBuilder) Layout(layout string) *ViewBuilder {
	v.layout = layout
	return v
}

// Name sets the name for the view.
func (v *ViewBuilder) Name(name string) *ViewBuilder {
	v.name = name
	return v
}

// Data sets the model data for the view.
func (v *ViewBuilder) Data(data any) *ViewBuilder {
	v.data = data
	return v
}

var _ Renderer = (*ViewBuilder)(nil)

// Render renders the view.
// It writes the response headers and body.
// It completes the `Renderer` interface.
func (v *ViewBuilder) Render(ctx *context.Context) error {
	if v.data != nil {
		if dataAsRenderer, ok := v.data.(Renderer); ok {
			return dataAsRenderer.Render(ctx)
		}
	}

	if v.name == "" {
		// Note(@kataras): an error, not a panic. Render runs per request, so a
		// ViewBuilder that reaches here without a name used to crash the request
		// goroutine rather than report a misconfiguration.
		return errors.New("mvc: view name is missing: use View(name) or Name(name) on the ViewBuilder")
	}

	if value := v.statusCode; value > 0 {
		ctx.StatusCode(value)
	} else if value = ctx.GetStatusCode(); value == 0 {
		ctx.StatusCode(200)
	}

	// content-type is handled by view.Render.
	err := view.Render(ctx, v.name, v.layout, v.data)
	return err
}
