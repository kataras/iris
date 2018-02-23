package router

import (
	"net/http"

	"github.com/kataras/iris/context"
)

// FallbackStack is a stack (with LIFO calling order) for fallback handlers
// A fallback handler(s) is(are) called from Fallback stack
//   when no route found and before sending NotFound status.
// Therefore Handler(s) in Fallback stack could send another thing than NotFound status,
//   if `Context.Next()` method is not called.
// Done & DoneGlobal Handlers are not called.
type FallbackStack struct {
	parent   *FallbackStack
	handlers context.Handlers
}

// _size is a terminal recursive method for computing size the stack
func (stk *FallbackStack) _size(i int) int {
	res := i + len(stk.handlers)

	if stk.parent == nil {
		return res
	}

	return stk.parent._size(res)
}

// populate is a recursive method for concatenating handlers to `list` parameter
func (stk *FallbackStack) populate(list context.Handlers) {
	n := copy(list, stk.handlers)

	if stk.parent != nil {
		stk.parent.populate(list[n:])
	}
}

// Size gives the size of the full stack hierarchy
func (stk *FallbackStack) Size() int {
	return stk._size(0)
}

// Add appends handlers to the beginning of the stack to have a LIFO calling order
func (stk *FallbackStack) Add(h context.Handlers) {
	stk.handlers = append(stk.handlers, h...)

	copy(stk.handlers[len(h):], stk.handlers)
	copy(stk.handlers, h)
}

// Fork make a new stack from this stack, and so create a stack child (leaf from a tree of stacks)
func (stk *FallbackStack) Fork() *FallbackStack {
	return &FallbackStack{
		parent: stk,
	}
}

// List concatenate all handlers in stack hierarchy
func (stk *FallbackStack) List() context.Handlers {
	res := make(context.Handlers, stk.Size())
	stk.populate(res)

	return res
}

// NewFallbackStack create a new Fallback stack with as first entry
//   a handler which send NotFound status (the default)
func NewFallbackStack() *FallbackStack {
	return &FallbackStack{
		handlers: context.Handlers{
			func(ctx context.Context) {
				ctx.StatusCode(http.StatusNotFound)
			},
		},
	}
}
