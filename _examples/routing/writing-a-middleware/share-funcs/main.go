// Package main shows how you can share a
// function between handlers of the same chain.
// Note that, this case is very rarely used and it exists,
// mostly, for 3rd-party middleware creators.
//
// The middleware creator registers a dynamic function by Context.SetFunc and
// the route handler just needs to call Context.CallFunc(funcName, arguments),
// without knowning what is the specific middleware's implementation or who was the creator
// of that function, it may be a basicauth middleware's logout or session's logout.
//
// See Context.SetLogoutFunc and Context.Logout methods too (these are not covered here).
package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
)

func main() {
	app := newApp()

	// GET: http://localhost:8080
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()
	app.Use(middleware)
	// OR app.UseRouter(middleware)
	// to register it everywhere,
	// including the HTTP errors.

	app.Get("/", handler)
	app.Get("/2", middleware2, handler2)
	app.Get("/3", middleware3, handler3)

	return app
}

// Assume: this is a middleware which does not export
// the 'hello' function for several reasons
// but we offer a 'greeting' optional feature to the route handler.
func middleware(ctx iris.Context) {
	ctx.SetFunc("greet", hello)
	ctx.Next()
}

// Assume: this is a handler which needs to "greet" the client but
// the function for that job is not predictable,
// it may change - dynamically (SetFunc) - depending on
// the middlewares registered before this route handler.
// E.g. it may be a "Hello $name" or "Greetings $Name".
func handler(ctx iris.Context) {
	outputs, err := ctx.CallFunc("greet", "Gophers")
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	response := outputs[0].Interface().(string)
	ctx.WriteString(response)
}

func middleware2(ctx iris.Context) {
	ctx.SetFunc("greet", sayHello)
	ctx.Next()
}

func handler2(ctx iris.Context) {
	_, err := ctx.CallFunc("greet", "Gophers [2]")
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}
}

func middleware3(ctx iris.Context) {
	ctx.SetFunc("job", function3)
	ctx.Next()
}

func handler3(ctx iris.Context) {
	_, err := ctx.CallFunc("job")
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	ctx.WriteString("OK, job was executed.\nSee the command prompt.")
}

/*
| ------------------------ |
| function implementations |
| ------------------------ |
*/

func hello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

func sayHello(ctx iris.Context, name string) {
	ctx.WriteString(hello(name))
}

func function3() {
	fmt.Printf("function3 called\n")
}
