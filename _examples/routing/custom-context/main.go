package main

import (
	"sync"

	"github.com/kataras/iris/v12"
)

func main() {
	// 1. Create the iris app instance.
	app := iris.New()

	// 2. Create the Context Wrapper which will be used to wrap the handlers
	// that expect a *myCustomContext instead of iris.Context.
	w := iris.NewContextWrapper(&myCustomContextPool{})
	// OR:
	// w := iris.NewContextWrapper(iris.NewContextPool[myCustomContext, *myCustomContext]())
	// The example custom context pool operates exactly the same as the result of iris.NewContextPool.

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

/*
	Custom Context Pool
*/
// Create the context sync pool for our custom context,
// the pool must implement Acquire() T and Release(T) methods to satisfy the iris.ContextPool interface.
type myCustomContextPool struct {
	pool sync.Pool
}

// Acquire returns a new custom context from the pool.
func (p *myCustomContextPool) Acquire(ctx iris.Context) *myCustomContext {
	v := p.pool.Get()
	if v == nil {
		v = &myCustomContext{
			Context: ctx,
			// custom fields here...
		}
	}

	return v.(*myCustomContext)
}

// Release puts a custom context back to the pool.
func (p *myCustomContextPool) Release(t *myCustomContext) {
	// You can take advantage of this method to clear the context
	// and re-use it on the Acquire method, use the sync.Pool.
	p.pool.Put(t)
}

/*
	Custom Context
*/
// Create a custom context.
type myCustomContext struct {
	// It's just an embedded field which is set on AcquireFunc,
	// so you can use myCustomContext with the same methods as iris.Context,
	// override existing iris.Context's methods or add custom methods.
	// You can use the `Context` field to access the original context.
	iris.Context
}

// SetContext sets the original iris.Context,
// should be implemented by custom context type(s) when
// the ContextWrapper uses a context Pool through the iris.NewContextPool function.
// Comment line 15, uncomment line 17 and the method below.
func (c *myCustomContext) SetContext(ctx iris.Context) {
	c.Context = ctx
}

func (c *myCustomContext) HTML(format string, args ...interface{}) (int, error) {
	c.Application().Logger().Info("HTML was called from custom Context")

	return c.Context.HTML(format, args...)
}
