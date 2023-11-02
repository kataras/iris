package iris

// ContextPool is a pool of T.
//
// See `NewContextWrapper` and `ContextPool` for more.
type ContextPool[T any] interface {
	Acquire(ctx Context) T
	Release(T)
}

// DefaultContextPool is a pool of T.
// It's used to acquire and release T.
// The T is acquired from the pool and released back to the pool after the handler's execution.
// The T is passed to the handler as an argument.
// The T is not shared between requests.
type DefaultContextPool[T any] struct {
	AcquireFunc func(Context) T
	ReleaseFunc func(T)
}

// Ensure that DefaultContextPool[T] implements ContextPool[T].
var _ ContextPool[any] = (*DefaultContextPool[any])(nil)

// Acquire returns a new T from the pool's AcquireFunc.
func (p *DefaultContextPool[T]) Acquire(ctx Context) T {
	acquire := p.AcquireFunc
	if p.AcquireFunc == nil {
		acquire = func(ctx Context) T {
			var t T
			return t
		}
	}

	return acquire(ctx)
}

// Release does nothing if the pool's ReleaseFunc is nil.
func (p *DefaultContextPool[T]) Release(t T) {
	release := p.ReleaseFunc
	if p.ReleaseFunc == nil {
		release = func(t T) {}
	}

	release(t)
}

// ContextWrapper is a wrapper for handlers which expect a T instead of iris.Context.
//
// See the `NewContextWrapper` function for more.
type ContextWrapper[T any] struct {
	pool ContextPool[T]
}

// NewContextWrapper returns a new ContextWrapper.
// If pool is nil, a default pool is used.
// The default pool's AcquireFunc returns a zero value of T.
// The default pool's ReleaseFunc does nothing.
// The default pool is used when the pool is nil.
// Use the `&iris.DefaultContextPool{...}` to pass a simple context pool.
//
// See the `Handler` method for more.
// Example: https://github.com/kataras/iris/tree/main/_examples/routing/custom-context
func NewContextWrapper[T any](pool ContextPool[T]) *ContextWrapper[T] {
	if pool == nil {
		pool = &DefaultContextPool[T]{
			AcquireFunc: func(ctx Context) T {
				var t T
				return t
			},
			ReleaseFunc: func(t T) {},
		}
	}

	return &ContextWrapper[T]{
		pool: pool,
	}
}

// Handler wraps the handler with the pool's Acquire and Release methods.
// It returns a new handler which expects a T instead of iris.Context.
// The T is the type of the pool.
// The T is acquired from the pool and released back to the pool after the handler's execution.
// The T is passed to the handler as an argument.
// The T is not shared between requests.
func (w *ContextWrapper[T]) Handler(handler func(T)) Handler {
	return func(ctx Context) {
		newT := w.pool.Acquire(ctx)
		handler(newT)
		w.pool.Release(newT)
	}
}
