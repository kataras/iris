package handlers

import "github.com/kataras/iris/context"

// Stack is a stack (with LIFO calling order) for handlers
type Stack struct {
	parent   *Stack
	handlers context.Handlers
}

// _size is a terminal recursive method for computing size the stack
func (stk *Stack) _size(i int) int {
	res := i + len(stk.handlers)

	if stk.parent == nil {
		return res
	}

	return stk.parent._size(res)
}

// populate is a recursive method for concatenating handlers to `list` parameter
func (stk *Stack) populate(list context.Handlers) {
	n := copy(list, stk.handlers)

	if stk.parent != nil {
		stk.parent.populate(list[n:])
	}
}

// Size gives the size of the full stack hierarchy
func (stk *Stack) Size() int {
	return stk._size(0)
}

// IsEmpty equals to `stk.Size() == 0`
func (stk *Stack) IsEmpty() bool {
	return (len(stk.handlers) == 0) && ((stk.parent == nil) || stk.parent.IsEmpty())
}

// Add appends handlers to the beginning of the stack to have a LIFO calling order
func (stk *Stack) Add(h context.Handlers) {
	stk.handlers = append(stk.handlers, h...)

	copy(stk.handlers[len(h):], stk.handlers)
	copy(stk.handlers, h)
}

// Fork make a new stack from this stack, and so create a stack child (leaf from a tree of stacks)
func (stk *Stack) Fork() Stack {
	return Stack{
		parent: stk,
	}
}

// List concatenate all handlers in stack hierarchy
func (stk *Stack) List() context.Handlers {
	res := make(context.Handlers, stk.Size())
	stk.populate(res)

	return res
}
