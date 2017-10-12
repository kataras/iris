package mvc

import (
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/mvc/activator/methodfunc"

	"github.com/fatih/structs"
)

// View completes the `methodfunc.Result` interface.
// It's being used as an alternative return value which
// wraps the template file name, layout, (any) view data, status code and error.
// It's smart enough to complete the request and send the correct response to the client.
//
// Example at: https://github.com/kataras/iris/blob/master/_examples/mvc/overview/web/controllers/hello_controller.go.
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
	if len(s) == 0 {
		return "index" + DefaultViewExt
	}

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
			// In order to respect any c.Ctx.ViewData that may called manually before;
			dataKey := ctx.Application().ConfigurationReadOnly().GetViewDataContextKey()
			if ctx.Values().Get(dataKey) == nil {
				// if no c.Ctx.ViewData then it's empty do a
				// pure set, it's faster.
				ctx.Values().Set(dataKey, r.Data)
			} else {
				// else check if r.Data is map or struct, if struct convert it to map,
				// do a range loop and set the data one by one.
				// context.Map is actually a map[string]interface{} but we have to make that check;
				if m, ok := r.Data.(map[string]interface{}); ok {
					setViewData(ctx, m)
				} else if m, ok := r.Data.(context.Map); ok {
					setViewData(ctx, m)
				} else if structs.IsStruct(r.Data) {
					setViewData(ctx, structs.Map(r))
				}
			}
		}

		ctx.View(r.Name)
	}
}

func setViewData(ctx context.Context, data map[string]interface{}) {
	for k, v := range data {
		ctx.ViewData(k, v)
	}
}
