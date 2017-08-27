package methodfunc

import (
	"reflect"
)

// MethodFunc the handler function.
type MethodFunc struct {
	FuncInfo
	FuncCaller
	RelPath string
}

// Resolve returns all the method funcs
// necessary information and actions to
// perform the request.
func Resolve(typ reflect.Type) (methodFuncs []MethodFunc) {
	infos := fetchInfos(typ)
	for _, info := range infos {
		p, ok := resolveRelativePath(info)
		if !ok {
			continue
		}
		caller := resolveCaller(p)
		methodFunc := MethodFunc{
			RelPath:    p.RelPath,
			FuncInfo:   info,
			FuncCaller: caller,
		}

		methodFuncs = append(methodFuncs, methodFunc)
	}

	return
}
