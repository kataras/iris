package mvc

import (
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/mvc/activator/methodfunc"
)

// View completes the `methodfunc.Result` interface.
// It's being used as an alternative return value which
// wraps the template file name, layout, (any) view data, status code and error.
// It's smart enough to complete the request and send the correct response to the client.
//
// Example at: https://github.com/kataras/iris/blob/master/_examples/mvc/using-method-result/controllers/hello_controller.go.
type View struct {
	Name   string
	Layout string
	Data   interface{} // map or a custom struct.
	Code   int
	Err    error
}

var _ methodfunc.Result = View{}

const dotB = byte('.')

// DefaultViewExt is the default extension if `view.Name `is missing,
// but note that it doesn't care about
// the app.RegisterView(iris.$VIEW_ENGINE("./$dir", "$ext"))'s $ext.
// so if you don't use the ".html" as extension for your files
// you have to append the extension manually into the `view.Name`
// or change this global variable.
var DefaultViewExt = ".html"

func ensureExt(s string) string {
	if strings.IndexByte(s, dotB) < 1 {
		s += DefaultViewExt
	}
	return s
}

// Dispatch writes the template filename, template layout and (any) data to the  client.
// Completes the `Result` interface.
func (r View) Dispatch(ctx context.Context) { // r as Response view.
	if r.Err != nil {
		if r.Code < 400 {
			r.Code = methodfunc.DefaultErrStatusCode
		}
		ctx.StatusCode(r.Code)
		ctx.WriteString(r.Err.Error())
		ctx.StopExecution()
		return
	}

	if r.Code > 0 {
		ctx.StatusCode(r.Code)
	}

	if r.Name != "" {
		r.Name = ensureExt(r.Name)

		if r.Layout != "" {
			r.Layout = ensureExt(r.Layout)
			ctx.ViewLayout(r.Layout)
		}

		if r.Data != nil {
			ctx.Values().Set(
				ctx.Application().ConfigurationReadOnly().GetViewDataContextKey(),
				r.Data,
			)
		}

		ctx.View(r.Name)
	}
}
