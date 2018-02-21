package router

import (
	"net/http"
	"sync"

	"github.com/kataras/iris/context"
)

type FallbackStack struct {
	handlers context.Handlers
	m        sync.Mutex
}

func (stk *FallbackStack) add(h context.Handlers) {
	stk.m.Lock()
	defer stk.m.Unlock()

	stk.handlers = append(stk.handlers, h...)

	copy(stk.handlers[len(h):], stk.handlers)
	copy(stk.handlers, h)
}

func (stk *FallbackStack) list() context.Handlers {
	res := make(context.Handlers, len(stk.handlers))

	stk.m.Lock()
	defer stk.m.Unlock()

	copy(res, stk.handlers)

	return res
}

func NewFallbackStack() *FallbackStack {
	return &FallbackStack{
		handlers: context.Handlers{
			func(ctx context.Context) {
				ctx.StatusCode(http.StatusNotFound)
			},
		},
	}
}
