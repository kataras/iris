package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	// 1. Create the iris app instance.
	app := iris.New()

	/*
		w := iris.NewContextWrapper(&iris.DefaultContextPool[*myCustomContext]{
			AcquireFunc: func(ctx iris.Context) *myCustomContext {
				return &myCustomContext{
					Context: ctx,
					// custom fields here...
				}
			},
			ReleaseFunc: func(t *myCustomContext) {
				// do nothing
			},
		})
	OR: */
	// 2. Create the Context Wrapper which will be used to wrap the handlers
	// that expect a *myCustomContext instead of iris.Context.
	w := iris.NewContextWrapper(&myCustomContextPool{})

	// 3. Register the handler(s) which expects a *myCustomContext instead of iris.Context.
	// The `w.Handler` will wrap the handler and will call the `Acquire` and `Release`
	// methods of the `myCustomContextPool` to get and release the *myCustomContext.
	app.Get("/", w.Handler(index))

	// 4. Start the server.
	app.Listen(":8080")
}

func index(ctx *myCustomContext) {
	ctx.HTML("<h1>Hello, World!</h1>")
}

// Create a custom context.
type myCustomContext struct {
	// It's just an embedded field which is set on AcquireFunc,
	// so you can use myCustomContext with the same methods as iris.Context,
	// override existing iris.Context's methods or add custom methods.
	// You can use the `Context` field to access the original context.
	iris.Context
}

func (c *myCustomContext) HTML(format string, args ...interface{}) (int, error) {
	c.Application().Logger().Info("HTML was called from custom Context")

	return c.Context.HTML(format, args...)
}

// Create the context memory pool for your custom context,
// the pool must contain Acquire() T and Release(T) methods.
type myCustomContextPool struct{}

// Acquire returns a new custom context from the pool.
func (p *myCustomContextPool) Acquire(ctx iris.Context) *myCustomContext {
	return &myCustomContext{
		Context: ctx,
		// custom fields here...
	}
}

// Release puts a custom context back to the pool.
func (p *myCustomContextPool) Release(t *myCustomContext) {
	// You can take advantage of this method to clear the context
	// and re-use it on the Acquire method, use the sync.Pool.
	//
	// We do nothing for the shake of the exampel.
}
