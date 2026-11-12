Sure, here is a detailed markdown documentation for the Context Wrapper feature:

# Context Wrapper Documentation

## Overview

A context wrapper lets you define your own context type on top of the standard Iris one, with the methods and fields your application needs. Handlers then receive your type instead of the plain `iris.Context`.

## Usage

### Creating a Custom Context

To create a custom context, define a struct that embeds the Iris context (optional) and implements the `SetContext` method (required).

```go
type myCustomContext struct {
    iris.Context
}

func (c *myCustomContext) SetContext(ctx iris.Context) {
    c.Context = ctx
}

func (c *myCustomContext) HTML(format string, args ...any) (int, error) {
    c.Application().Logger().Info("HTML was called from custom Context")
    return c.Context.HTML(format, args...)
}
```

### Initializing the Context Wrapper

To initialize the context wrapper, use the `NewContextWrapper` function. This function returns a new `ContextWrapper` that uses the default pool.

```go
w := iris.NewContextWrapper[myCustomContext]()
```

### Registering Handlers

Use the `Handler` method of the `ContextWrapper` to wrap handlers that expect the custom context.

```go
app.Get("/", w.Handler(index))
```

### Example

Here is a complete example that demonstrates how to use the Context Wrapper feature:

```go
package main

import (
    "github.com/kataras/iris/v14"
)

func main() {
    // 1. Create the iris app instance.
    app := iris.New()

    // 2. Create the Context Wrapper which will be used to wrap the handlers.
    w := iris.NewContextWrapper[myCustomContext]()
    // You can move the "w" variable on its own package so you can use it everywhere in your application.

    // 3. Register the handler(s) which expects a *myCustomContext instead of iris.Context.
    // The `w.Handler` will wrap the handler.
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

// SetContext sets the original iris.Context,
// should be implemented by custom context type(s) when
// the ContextWrapper uses a the default context Pool.
func (c *myCustomContext) SetContext(ctx iris.Context) {
    c.Context = ctx
}

func (c *myCustomContext) HTML(format string, args ...any) (int, error) {
    c.Application().Logger().Info("HTML was called from custom Context")
    return c.Context.HTML(format, args...)
}
```

## Advanced Usage

### Custom Context Pool

If you need more control over the context pool, you can create a custom context pool by implementing the `ContextPool` interface.

The `ContextPool` interface is a pool of type `T`. It is used to acquire and release custom contexts.

```go
type ContextPool[T any] interface {
    Acquire(ctx *Context) T
    Release(T)
}
```

### Usage

```go
type myCustomContextPool struct {
    pool sync.Pool
}

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

func (p *myCustomContextPool) Release(t *myCustomContext) {
    // You can take advantage of this method to clear the context
    // and re-use it on the Acquire method, use the sync.Pool.
    p.pool.Put(t)
}
```

### Initializing with Custom Context Pool

To initialize the context wrapper with a custom context pool, use the `context.NewContextWrapperWithPool` function.

```go
import "github.com/kataras/iris/v14/context"
```

```go
w := context.NewContextWrapperWithPool(&myCustomContextPool{})
```

