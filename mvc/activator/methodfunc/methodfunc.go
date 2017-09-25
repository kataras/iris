package methodfunc

import (
	"reflect"

	"github.com/kataras/golog"
	"github.com/kataras/iris/context"
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
func Resolve(typ reflect.Type) (methodFuncs []MethodFunc) {
	infos := fetchInfos(typ)
	for _, info := range infos {
		parser := newFuncParser(info)
		a, err := parser.parse()
		if err != nil {
			golog.Errorf("MVC: %s\n", err)
			continue
		}
		methodFunc := MethodFunc{
			RelPath:    a.relPath,
			FuncInfo:   info,
			MethodCall: buildMethodCall(a),
		}

		methodFuncs = append(methodFuncs, methodFunc)
	}

	return
}
