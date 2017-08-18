package activator

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
)

var availableMethods = [...]string{
	"ANY",  // will be registered using the `core/router#APIBuilder#Any`
	"ALL",  // same as ANY
	"NONE", // offline route
	// valid http methods
	"GET",
	"POST",
	"PUT",
	"DELETE",
	"CONNECT",
	"HEAD",
	"PATCH",
	"OPTIONS",
	"TRACE",
}

type methodControl struct{}

// ErrMissingHTTPMethodFunc fired when the controller doesn't handle any valid HTTP method.
var ErrMissingHTTPMethodFunc = errors.New(`controller can not be activated,
 missing a compatible HTTP method function, i.e Get()`)

func (mc *methodControl) Load(t *TController) error {
	// search the entire controller
	// for any compatible method function
	// and register that.
	for _, method := range availableMethods {
		if m, ok := t.Type.MethodByName(getMethodName(method)); ok {

			t.Methods = append(t.Methods, MethodFunc{
				HTTPMethod: method,
				Index:      m.Index,
			})

			// check if method was Any() or All()
			// if yes, then break to skip any conflict with the rest of the method functions.
			// (this will be registered to all valid http methods by the APIBuilder)
			if method == "ANY" || method == "ALL" {
				break
			}
		}
	}

	if len(t.Methods) == 0 {
		// no compatible method found, fire an error and stop everything.
		return ErrMissingHTTPMethodFunc
	}

	return nil
}

func getMethodName(httpMethod string) string {
	httpMethodFuncName := strings.Title(strings.ToLower(httpMethod))
	return httpMethodFuncName
}

func (mc *methodControl) Handle(ctx context.Context, c reflect.Value, methodFunc func()) {
	// execute the responsible method for that handler.
	// Remember:
	// To improve the performance
	// we don't compare the ctx.Method()[HTTP Method]
	// to the instance's Method, each handler is registered
	// to a specific http method.
	methodFunc()
}

// MethodControl loads and serve the main functionality of the controllers,
// which is to run a function based on the http method (pre-computed).
func MethodControl() TControl {
	return &methodControl{}
}
