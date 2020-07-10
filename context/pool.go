package context

import (
	"net/http"
	"sync"
)

// Pool is the context pool, it's used inside router and the framework by itself.
type Pool struct {
	pool *sync.Pool
}

// New creates and returns a new context pool.
func New(newFunc func() interface{}) *Pool {
	return &Pool{pool: &sync.Pool{New: newFunc}}
}

// Acquire returns a Context from pool.
// See Release.
func (c *Pool) Acquire(w http.ResponseWriter, r *http.Request) *Context {
	ctx := c.pool.Get().(*Context)
	ctx.BeginRequest(w, r)
	return ctx
}

// Release puts a Context back to its pull, this function releases its resources.
// See Acquire.
func (c *Pool) Release(ctx *Context) {
	ctx.EndRequest()
	c.pool.Put(ctx)
}

// ReleaseLight will just release the object back to the pool, but the
// clean method is caller's responsibility now, currently this is only used
// on `SPABuilder`.
func (c *Pool) ReleaseLight(ctx *Context) {
	c.pool.Put(ctx)
}
