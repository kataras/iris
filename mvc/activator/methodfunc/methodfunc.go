package methodfunc

import (
	"reflect"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
)

// MethodFunc the handler function.
type MethodFunc struct {
	FuncInfo
	// MethodCall fires the actual handler.
	// The "ctx" is the current context, helps us to get any path parameter's values.
	//
	// The "f" is the controller's function which is responsible
	// for that request for this http method.
	// That function can accept one parameter.
	//
	// The default callers (and the only one for now)
	// are pre-calculated by the framework.
	MethodCall func(ctx context.Context, f reflect.Value)
	RelPath    string
}

// Resolve returns all the method funcs
// necessary information and actions to
// perform the request.
func Resolve(typ reflect.Type) ([]MethodFunc, error) {
	r := errors.NewReporter()
	var methodFuncs []MethodFunc
	infos := fetchInfos(typ)
	for _, info := range infos {
		methodFunc, err := ResolveMethodFunc(info)
		if r.AddErr(err) {
			continue
		}
		methodFuncs = append(methodFuncs, methodFunc)
	}

	return methodFuncs, r.Return()
}

// ResolveMethodFunc resolves a single `MethodFunc` from a single `FuncInfo`.
func ResolveMethodFunc(info FuncInfo, paramKeys ...string) (MethodFunc, error) {
	parser := newFuncParser(info)
	a, err := parser.parse()
	if err != nil {
		return MethodFunc{}, err
	}

	if len(paramKeys) > 0 {
		a.paramKeys = paramKeys
	}

	methodFunc := MethodFunc{
		RelPath:    a.relPath,
		FuncInfo:   info,
		MethodCall: buildMethodCall(a),
	}

	/* TODO: split the method path and ast param keys, and all that
	because now we want to use custom param keys but 'paramfirst' is set-ed.

	*/

	return methodFunc, nil
}
