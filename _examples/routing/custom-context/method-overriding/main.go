package main

// In this package I'll show you how to override the existing Context's functions and methods.
// You can easly navigate to the custom-context example to see how you can add new functions
// to your own context (need a custom handler).
//
// This way is far easier to understand and it's faster when you want to override existing methods:
import (
	"reflect"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

// Create your own custom Context, put any fields you wanna need.
type MyContext struct {
	// Optional Part 1: embed (optional but required if you don't want to override all context's methods)
	iris.Context
}

var _ iris.Context = &MyContext{} // optionally: validate on compile-time if MyContext implements context.Context.

// The only one important if you will override the Context
// with an embedded context.Context inside it.
// Required in order to run the handlers via this "*MyContext".
func (ctx *MyContext) Do(handlers context.Handlers) {
	context.Do(ctx, handlers)
}

// The second one important if you will override the Context
// with an embedded context.Context inside it.
// Required in order to run the chain of handlers via this "*MyContext".
func (ctx *MyContext) Next() {
	context.Next(ctx)
}

// Override any context's method you want...
// [...]

func (ctx *MyContext) HTML(format string, args ...interface{}) (int, error) {
	ctx.Application().Logger().Infof("Executing .HTML function from MyContext")

	ctx.ContentType("text/html")
	return ctx.Writef(format, args...)
}

func main() {
	app := iris.New()
	// app.Logger().SetLevel("debug")

	// The only one Required:
	// here is how you define how your own context will
	// be created and acquired from the iris' generic context pool.
	app.ContextPool.Attach(func() iris.Context {
		return &MyContext{
			// Optional Part 3:
			Context: context.NewContext(app),
		}
	})

	// Register a view engine on .html files inside the ./view/** directory.
	app.RegisterView(iris.HTML("./view", ".html"))

	// register your route, as you normally do
	app.Handle("GET", "/", recordWhichContextJustForProofOfConcept, func(ctx iris.Context) {
		// use the context's overridden HTML method.
		ctx.HTML("<h1> Hello from my custom context's HTML! </h1>")
	})

	// this will be executed by the MyContext.Context
	// if MyContext is not directly define the View function by itself.
	app.Handle("GET", "/hi/{firstname:alphabetical}", recordWhichContextJustForProofOfConcept, func(ctx iris.Context) {
		firstname := ctx.Params().Get("firstname")

		ctx.ViewData("firstname", firstname)
		ctx.Gzip(true)

		ctx.View("hi.html")
	})

	app.Listen(":8080")
}

// should always print "($PATH) Handler is executing from 'MyContext'"
func recordWhichContextJustForProofOfConcept(ctx iris.Context) {
	ctx.Application().Logger().Infof("(%s) Handler is executing from: '%s'", ctx.Path(), reflect.TypeOf(ctx).Elem().Name())
	ctx.Next()
}

// Look "new-implementation" to see how you can create an entirely new Context with new functions.
