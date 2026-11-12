package context

import (
	"sync"
)

// Pool is the context pool, it's used inside router and the framework by itself.
type Pool struct {
	pool *sync.Pool
}

// New creates and returns a new context pool.
func New(newFunc func() any) *Pool {
	return &Pool{pool: &sync.Pool{New: newFunc}}
}

// Acquire returns a Context from pool.
// See Release.
func (c *Pool) Acquire() *Context {
	ctx := c.pool.Get().(*Context)
	return ctx
}

// Release puts a Context back to its pool, this function releases its resources.
// See Acquire.
func (c *Pool) Release(ctx *Context) {
	c.ForceRelease(ctx)
}

// ForceRelease puts a Context back to its pool, this function releases its resources.
// It does the same as Release.
func (c *Pool) ForceRelease(ctx *Context) {
	if options := ctx.options; options != nil {
		releaseOptions(options)
		ctx.options = nil
	}

	ctx.currentRoute = nil
	ctx.handlers = nil // will be filled by router.Serve/HTTP

	// Zero the entries before re-slicing. Values and params hold "any", so
	// re-slicing alone would keep this request's data - the decoded body, the
	// User and the plaintext password an auth middleware may have put on it -
	// reachable from the pooled Context until the slot is overwritten.
	clear(ctx.values)
	clear(ctx.params)
	ctx.values = ctx.values[0:0] // >> will be filled by context.Values().Set
	ctx.params = ctx.params[0:0] // >>      >>     by context.Params().Set

	ctx.currentHandlerIndex = 0
	ctx.proceeded = 0
	ctx.writer = nil
	ctx.request = nil

	// The router runs the release hooks before it gets here and leaves the
	// slice empty. A caller that releases a Context itself skips that step,
	// and a later RunReleaseHooks would then run on a Context that may already
	// belong to the next request; dropping the hooks here is what makes that
	// call a no-op.
	clear(ctx.releaseHooks)
	ctx.releaseHooks = ctx.releaseHooks[:0]

	c.pool.Put(ctx)
}
