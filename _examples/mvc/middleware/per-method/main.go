/*
If you want to use it as middleware for the entire controller
you can use its router which is just a sub router to add it as you normally do with standard API:

I'll show you 4 different methods for adding a middleware into an mvc application,
all of those 4 do exactly the same thing, select what you prefer,
I prefer the last code-snippet when I need the middleware to be registered somewhere
else as well, otherwise I am going with the first one:

```go
// 1
mvc.Configure(app.Party("/user"), func(m *mvc.Application) {
     m.Router.Use(cache.Handler(10*time.Second))
})
```

```go
// 2
// same:
userRouter := app.Party("/user")
userRouter.Use(cache.Handler(10*time.Second))
mvc.Configure(userRouter, ...)
```

```go
// 3
// same:
userRouter := app.Party("/user", cache.Handler(10*time.Second))
mvc.Configure(userRouter, ...)
```

```go
// 4
// same:
app.PartyFunc("/user", func(r iris.Party){
    r.Use(cache.Handler(10*time.Second))
    mvc.Configure(r, ...)
})
```

If you want to use a middleware for a single route,
for a single controller's method that is already registered by the engine
and not by custom `Handle` (which you can add
the middleware there on the last parameter) and it's not depend on the `Next Handler` to do its job
then you just call it on the method:

```go
var myMiddleware := myMiddleware.New(...) // this should return an iris/context.Handler

type UserController struct{}
func (c *UserController) GetSomething(ctx iris.Context) {
    // ctx.Proceed checks if myMiddleware called `ctx.Next()`
    // inside it and returns true if so, otherwise false.
    nextCalled := ctx.Proceed(myMiddleware)
    if !nextCalled {
        return
    }

    // else do the job here, it's allowed
}
```

And last, if you want to add a middleware on a specific method
and it depends on the next and the whole chain then you have to do it
using the `AfterActivation` like the example below:
*/
package main

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/cache"
	"github.com/kataras/iris/mvc"
)

var cacheHandler = cache.Handler(10 * time.Second)

func main() {
	app := iris.New()
	// You don't have to use .Configure if you do it all in the main func
	// mvc.Configure and mvc.New(...).Configure() are just helpers to split
	// your code better, here we use the simplest form:
	m := mvc.New(app)
	m.Handle(&exampleController{})

	app.Run(iris.Addr(":8080"))
}

type exampleController struct{}

func (c *exampleController) AfterActivation(a mvc.AfterActivation) {
	// select the route based on the method name you want to
	// modify.
	index := a.GetRoute("Get")
	// just prepend the handler(s) as middleware(s) you want to use.
	// or append for "done" handlers.
	index.Handlers = append([]iris.Handler{cacheHandler}, index.Handlers...)
}

func (c *exampleController) Get() string {
	// refresh every 10 seconds and you will see different time output.
	now := time.Now().Format("Mon, Jan 02 2006 15:04:05")
	return "last time executed without cache: " + now
}
