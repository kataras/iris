package hero

import (
	"reflect"

	"github.com/kataras/iris/context"
)

// weak because we don't have access to the path, neither
// the macros, so this is just a guess based on the index of the path parameter,
// the function's path parameters should be like a chain, in the same order as
// the caller registers a route's path.
// A context or any value(s) can be in front or back or even between them.
type params struct {
	// the next function input index of where the next path parameter
	// should be inside the CONTEXT.
	next int
}

func (p *params) resolve(index int, typ reflect.Type) (reflect.Value, bool) {
	currentParamIndex := p.next
	v, ok := resolveParam(currentParamIndex, typ)

	p.next = p.next + 1
	return v, ok
}

func resolveParam(currentParamIndex int, typ reflect.Type) (reflect.Value, bool) {
	var fn interface{}

	switch typ.Kind() {
	case reflect.Int:
		fn = func(ctx context.Context) int {
			// the second "ok/found" check is not necessary,
			// because even if the entry didn't found on that "index"
			// it will return an empty entry which will return the
			// default value passed from the xDefault(def) because its `ValueRaw` is nil.
			entry, _ := ctx.Params().GetEntryAt(currentParamIndex)
			v, _ := entry.IntDefault(0)
			return v
		}
	case reflect.Int64:
		fn = func(ctx context.Context) int64 {
			entry, _ := ctx.Params().GetEntryAt(currentParamIndex)
			v, _ := entry.Int64Default(0)

			return v
		}
	case reflect.Bool:
		fn = func(ctx context.Context) bool {
			entry, _ := ctx.Params().GetEntryAt(currentParamIndex)
			v, _ := entry.BoolDefault(false)
			return v
		}
	case reflect.String:
		fn = func(ctx context.Context) string {
			entry, _ := ctx.Params().GetEntryAt(currentParamIndex)
			// print(entry.Key + " with index of: ")
			// print(currentParamIndex)
			// println(" and value: " + entry.String())
			return entry.String()
		}
	default:
		return reflect.Value{}, false
	}

	return reflect.ValueOf(fn), true
}
