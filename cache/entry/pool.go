package entry

import (
	"sync"
	"time"

	"github.com/kataras/iris/v12/cache/cfg"
	"github.com/kataras/iris/v12/core/memstore"
)

// Pool is the context pool, it's used inside router and the framework by itself.
type Pool struct {
	pool *sync.Pool
}

// NewPool creates and returns a new context pool.
func NewPool() *Pool {
	return &Pool{pool: &sync.Pool{New: func() any { return &Entry{} }}}
}

// func NewPool(newFunc func() any) *Pool {
// 	return &Pool{pool: &sync.Pool{New: newFunc}}
// }

// Acquire returns an Entry from pool.
// See Release.
func (c *Pool) Acquire(lifeDuration time.Duration, r *Response, onExpire func()) *Entry {
	// If the given duration is not <=0 (which means finds from the headers)
	// then we should check for the MinimumCacheDuration here
	if lifeDuration >= 0 && lifeDuration < cfg.MinimumCacheDuration {
		lifeDuration = cfg.MinimumCacheDuration
	}

	e := c.pool.Get().(*Entry)

	lt := memstore.NewLifeTime()
	lt.Begin(lifeDuration, func() {
		onExpire()
		c.release(e)
	})

	e.reset(lt, r)
	return e
}

// Release puts an Entry back to its pull, this function releases its resources.
// See Acquire.
func (c *Pool) release(e *Entry) {
	e.response.body = nil
	e.response.headers = nil
	e.response.statusCode = 0
	e.response = nil

	// do not call it, it contains a lock too, release is controlled only inside the Acquire itself when the entry is expired.
	// if e.lifeTime != nil {
	// 	e.lifeTime.ExpireNow() // stop any opening timers if force released.
	// }

	c.pool.Put(e)
}

// Release can be called by custom stores to release an entry.
func (c *Pool) Release(e *Entry) {
	if e.lifeTime != nil {
		e.lifeTime.ExpireNow() // stop any opening timers if force released.
	}

	c.release(e)
}
