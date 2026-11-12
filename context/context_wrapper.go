package context

import (
	"sync"
	"time"
)

type (
	// ContextSetter is an interface which can be implemented by a struct
	// to set the iris.Context to the struct.
	// The receiver must be a pointer of the struct.
	ContextSetter interface {
		// SetContext sets the iris.Context to the struct.
		SetContext(*Context)
	}

	// ContextSetterPtr is a pointer of T which implements the `ContextSetter` interface.
	// The T must be a struct.
	ContextSetterPtr[T any] interface {
		*T
		ContextSetter
	}

	// emptyContextSetter is an empty struct which implements the `ContextSetter` interface.
	emptyContextSetter struct{}
)

// SetContext method implements `ContextSetter` interface.
func (*emptyContextSetter) SetContext(*Context) {}

// ContextPool is a pool of T. It's used to acquire and release custom context.
// Use of custom implementation or `NewContextPool`.
//
// See `NewContextWrapper` and `NewContextPool` for more.
type (
	ContextPool[T any] interface {
		// Acquire must return a new T from a pool.
		Acquire(ctx *Context) T
		// Release must put the T back to the pool.
		Release(T)
	}

	// syncContextPool is a sync pool implementation of T.
	// It's used to acquire and release T.
	// The contextPtr is acquired from the sync pool and released back to the sync pool after the handler's execution.
	// The contextPtr is passed to the handler as an argument.
	// ThecontextPtr is not shared between requests.
	// The contextPtr must implement the `ContextSetter` interface.
	// The T must be a struct.
	// The contextPtr must be a pointer of T.
	syncContextPool[T any, contextPtr ContextSetterPtr[T]] struct {
		pool *sync.Pool
	}
)

// Ensure that syncContextPool implements ContextPool.
var _ ContextPool[*emptyContextSetter] = (*syncContextPool[emptyContextSetter, *emptyContextSetter])(nil)

// NewContextPool returns a new ContextPool default implementation which
// uses sync.Pool to implement its Acquire and Release methods.
// The contextPtr is acquired from the sync pool and released back to the sync pool after the handler's execution.
// The contextPtr is passed to the handler as an argument.
// ThecontextPtr is not shared between requests.
// The contextPtr must implement the `ContextSetter` interface.
// The T must be a struct.
// The contextPtr must be a pointer of T.
//
// Example:
// w := context.NewContextWrapper(context.NewContextPool[myCustomContext]())
func NewContextPool[T any, contextPtr ContextSetterPtr[T]]() ContextPool[contextPtr] {
	return &syncContextPool[T, contextPtr]{
		pool: &sync.Pool{
			New: func() any {
				var t contextPtr = new(T)
				return t
			},
		},
	}
}

// Acquire returns a new T from the sync pool.
func (p *syncContextPool[T, contextPtr]) Acquire(ctx *Context) contextPtr {
	// var t contextPtr
	// if v := p.pool.Get(); v == nil {
	// 	t = new(T)
	// } else {
	// 	t = v.(contextPtr)
	// }

	t := p.pool.Get().(contextPtr)
	t.SetContext(ctx)
	return t
}

// Release zeroes the T and puts it back to the sync pool.
//
// The zeroing is not optional: a custom context typically carries per-request
// fields next to the embedded Context (the authenticated user, a tenant id, a
// decoded token). Without it those fields survive into whichever request picks
// the value up next, and a handler that reads a field it did not set in this
// request reads another client's data.
func (p *syncContextPool[T, contextPtr]) Release(t contextPtr) {
	var zero T
	*t = zero
	p.pool.Put(t)
}

// ContextWrapper is a wrapper for handlers which expect a T instead of Iris Context.
//
// See the `NewContextWrapper` function for more.
type ContextWrapper[T any] struct {
	pool ContextPool[T]
}

// NewContextWrapperWithPool returns a new ContextWrapper.
// The "pool" must not be nil, this function panics if it is.
// Use the `NewContextPool[T, contextPtr]()` to pass the default sync.Pool-backed
// implementation, or the `NewContextWrapper[T]()` which builds it for you.
// Then, use the `Handler` method to wrap custom handlers to iris ones.
//
// Create the context sync pool for our custom context,
// the pool must implement Acquire() T and Release(T) methods to satisfy the ContextPool interface.
//
// Usage:
//
//	type myCustomContextPool struct {
//		pool sync.Pool
//	}
//
// Acquire returns a new custom context from the pool.
//
//	func (p *myCustomContextPool) Acquire(ctx iris.Context) *myCustomContext {
//		v := p.pool.Get()
//		if v == nil {
//			v = &myCustomContext{
//				Context: ctx,
//				// custom fields here...
//			}
//		}
//
//		return v.(*myCustomContext)
//	}
//
// Release puts a custom context back to the pool.
//
//	func (p *myCustomContextPool) Release(t *myCustomContext) {
//		// You can take advantage of this method to clear the context
//		// and re-use it on the Acquire method, use the sync.Pool.
//		p.pool.Put(t)
//	}
//
// Declare a custom context.
//
//	type myCustomContext struct {
//		// It's just an embedded field which is set on AcquireFunc,
//		// so you can use myCustomContext with the same methods as iris.Context,
//		// override existing iris.Context's methods or add custom methods.
//		// You can use the `Context` field to access the original context.
//		iris.Context
//	}
//
//	func (c *myCustomContext) MyCustomMethod() {
//		// custom method here...
//	}
//
// Initialize a new context wrapper.
//
//	w := context.NewContextWrapperWithPool(&myCustomContextPool{})
//
// OR initialize a new context wrapper using the default pool,
// where the myCustomContext SHOULD implement the `SetContext(ctx iris.Context)` method.
//
//	w := context.NewContextWrapper[myCustomContext]()
//
// Use it to wrap the handlers.
//
//	app.Get("/", w.Handler(index))
func NewContextWrapperWithPool[T any](pool ContextPool[T]) *ContextWrapper[T] {
	if pool == nil {
		panic("pool cannot be nil")
	}

	return &ContextWrapper[T]{
		pool: pool,
	}
}

// NewContextWrapper returns a new ContextWrapper. It uses the default pool.
// The default pool's Acquire returns a T from a sync.Pool and sets the iris Context to it.
// The default pool's Release zeroes the T before it puts it back to the sync.Pool,
// so per-request fields never leak into the next request.
//
// Usage:
//
// Initialize a new context wrapper using the default pool,
// where the myCustomContext SHOULD implement the `SetContext(ctx iris.Context)` method.
//
//	w := context.NewContextWrapper[myCustomContext]()
//
// Use it to wrap the handlers.
//
//	app.Get("/", w.Handler(index))
//
// See the `NewContextWrapperWithPool` for more.
func NewContextWrapper[T any, contextPtr ContextSetterPtr[T]]() *ContextWrapper[contextPtr] {
	return NewContextWrapperWithPool(NewContextPool[T, contextPtr]())
}

// Pool returns the pool, useful when manually Acquire and Release of custom context is required.
func (w *ContextWrapper[T]) Pool() ContextPool[T] {
	return w.pool
}

// Handler wraps the handler with the pool's Acquire and Release methods.
// It returns a new handler which expects a T instead of iris.Context.
// The T is the type of the pool.
// The T is acquired from the pool and released back to the pool after the handler's execution.
// The T is passed to the handler as an argument.
// The T is not shared between requests.
func (w *ContextWrapper[T]) Handler(handler func(T)) Handler {
	if handler == nil {
		return nil
	}

	return func(ctx *Context) {
		newT := w.pool.Acquire(ctx)
		handler(newT)
		w.pool.Release(newT)
	}
}

// Handlers wraps the handlers with the pool's Acquire and Release methods.
func (w *ContextWrapper[T]) Handlers(handlers ...func(T)) Handlers {
	newHandlers := make(Handlers, len(handlers))
	for i, handler := range handlers {
		newHandlers[i] = w.Handler(handler)
	}

	return newHandlers
}

// HandlerReturnError same as `Handler` but it converts a handler which returns an error.
func (w *ContextWrapper[T]) HandlerReturnError(handler func(T) error) func(*Context) error {
	if handler == nil {
		return nil
	}

	return func(ctx *Context) error {
		newT := w.pool.Acquire(ctx)
		err := handler(newT)
		w.pool.Release(newT)
		return err
	}
}

// HandlerReturnDuration same as `Handler` but it converts a handler which returns a time.Duration.
func (w *ContextWrapper[T]) HandlerReturnDuration(handler func(T) time.Duration) func(*Context) time.Duration {
	if handler == nil {
		return nil
	}

	return func(ctx *Context) time.Duration {
		newT := w.pool.Acquire(ctx)
		duration := handler(newT)
		w.pool.Release(newT)
		return duration
	}
}

// Filter same as `Handler` but it converts a handler to Filter.
func (w *ContextWrapper[T]) Filter(handler func(T) bool) Filter {
	if handler == nil {
		return nil
	}

	return func(ctx *Context) bool {
		newT := w.pool.Acquire(ctx)
		shouldContinue := handler(newT)
		w.pool.Release(newT)
		return shouldContinue
	}
}
