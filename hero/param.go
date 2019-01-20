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
	v, ok := context.ParamResolverByTypeAndIndex(typ, currentParamIndex)

	p.next = p.next + 1
	return v, ok
}
