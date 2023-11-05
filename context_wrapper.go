package iris

import (
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
)

type (
	// ContextSetter is an interface which can be implemented by a struct
	// to set the iris.Context to the struct.
	// The receiver must be a pointer of the struct.
	ContextSetter interface {
		// SetContext sets the iris.Context to the struct.
		SetContext(Context)
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
func (*emptyContextSetter) SetContext(Context) {}

// ContextPool is a pool of T. It's used to acquire and release custom context.
// Use of custom implementation or `NewContextPool`.
//
// See `NewContextWrapper` and `NewContextPool` for more.
type (
	ContextPool[T any] interface {
		// Acquire must return a new T from a pool.
		Acquire(ctx Context) T
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
// w := iris.NewContextWrapper(iris.NewContextPool[myCustomContext, *myCustomContext]())
func NewContextPool[T any, contextPtr ContextSetterPtr[T]]() ContextPool[contextPtr] {
	return &syncContextPool[T, contextPtr]{
		pool: &sync.Pool{
			New: func() interface{} {
				var t contextPtr = new(T)
				return t
			},
		},
	}
}

// Acquire returns a new T from the sync pool.
func (p *syncContextPool[T, contextPtr]) Acquire(ctx Context) contextPtr {
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

// Release puts the T back to the sync pool.
func (p *syncContextPool[T, contextPtr]) Release(t contextPtr) {
	p.pool.Put(t)
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
// Use the `iris.NewContextPool[T, *T]()` to pass a simple context pool.
// Then, use the `Handler` method to wrap custom handlers to iris ones.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/routing/custom-context
func NewContextWrapper[T any](pool ContextPool[T]) *ContextWrapper[T] {
	if pool == nil {
		panic("pool cannot be nil")
	}

	return &ContextWrapper[T]{
		pool: pool,
	}
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

	return func(ctx Context) {
		newT := w.pool.Acquire(ctx)
		handler(newT)
		w.pool.Release(newT)
	}
}

// Handlers wraps the handlers with the pool's Acquire and Release methods.
func (w *ContextWrapper[T]) Handlers(handlers ...func(T)) context.Handlers {
	newHandlers := make(context.Handlers, len(handlers))
	for i, handler := range handlers {
		newHandlers[i] = w.Handler(handler)
	}

	return newHandlers
}

// HandlerReturnError same as `Handler` but it converts a handler which returns an error.
func (w *ContextWrapper[T]) HandlerReturnError(handler func(T) error) func(Context) error {
	if handler == nil {
		return nil
	}

	return func(ctx Context) error {
		newT := w.pool.Acquire(ctx)
		err := handler(newT)
		w.pool.Release(newT)
		return err
	}
}

// HandlerReturnDuration same as `Handler` but it converts a handler which returns a time.Duration.
func (w *ContextWrapper[T]) HandlerReturnDuration(handler func(T) time.Duration) func(Context) time.Duration {
	if handler == nil {
		return nil
	}

	return func(ctx Context) time.Duration {
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

	return func(ctx Context) bool {
		newT := w.pool.Acquire(ctx)
		shouldContinue := handler(newT)
		w.pool.Release(newT)
		return shouldContinue
	}
}

// FallbackViewFunc same as `Handler` but it converts a handler to FallbackViewFunc.
func (w *ContextWrapper[T]) FallbackViewFunc(handler func(ctx T, err ErrViewNotExist) error) FallbackViewFunc {
	if handler == nil {
		return nil
	}

	return func(ctx Context, err ErrViewNotExist) error {
		newT := w.pool.Acquire(ctx)
		returningErr := handler(newT, err)
		w.pool.Release(newT)
		return returningErr
	}
}
