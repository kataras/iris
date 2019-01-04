package context

import (
	"net/http"
	"sync"
)

// Pool is the context pool, it's used inside router and the framework by itself.
//
// It's the only one real implementation inside this package because it used widely.
type Pool struct {
	pool    *sync.Pool
	newFunc func() Context // we need a field otherwise is not working if we change the return value
}

// New creates and returns a new context pool.
func New(newFunc func() Context) *Pool {
	c := &Pool{pool: &sync.Pool{}, newFunc: newFunc}
	c.pool.New = func() interface{} { return c.newFunc() }
	return c
}

// Attach changes the pool's return value Context.
//
// The new Context should explicitly define the `Next()`
// and `Do(context.Handlers)` functions.
//
// Example: https://github.com/kataras/iris/blob/master/_examples/routing/custom-context/method-overriding/main.go
func (c *Pool) Attach(newFunc func() Context) {
	c.newFunc = newFunc
}

// Acquire returns a Context from pool.
// See Release.
func (c *Pool) Acquire(w http.ResponseWriter, r *http.Request) Context {
	ctx := c.pool.Get().(Context)
	ctx.BeginRequest(w, r)
	return ctx
}

// Release puts a Context back to its pull, this function releases its resources.
// See Acquire.
func (c *Pool) Release(ctx Context) {
	ctx.EndRequest()
	c.pool.Put(ctx)
}

// ReleaseLight will just release the object back to the pool, but the
// clean method is caller's responsibility now, currently this is only used
// on `SPABuilder`.
func (c *Pool) ReleaseLight(ctx Context) {
	c.pool.Put(ctx)
}
