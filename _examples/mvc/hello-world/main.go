package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	// auto-completion does not working well with type aliases
	// when embedded fields.
	// We should complete a report on golang repo for that at some point.
	//
	// Therefore import the "mvc" package manually
	// here at "hello-world" so users can see that
	// import path somewhere else than the "FAQ" section.

	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
)

// This example is equivalent to the
// https://github.com/kataras/iris/blob/master/_examples/hello-world/main.go
//
// It seems that additional code you
// have to write doesn't worth it
// but remember that, this example
// does not make use of iris mvc features like
// the Model, Persistence or the View engine neither the Session,
// it's very simple for learning purposes,
// probably you'll never use such
// as simple controller anywhere in your app.
//
// The cost we have on this example for using MVC
// on the "/hello" path which serves JSON
// is ~2MB per 20MB throughput on my personal laptop,
// it's tolerated for the majority of the applications
// but you can choose
// what suits you best with Iris, low-level handlers: performance
// or high-level controllers: easier to maintain and smaller codebase on large applications.

func main() {
	app := iris.New()
	// Optionally, add two built'n handlers
	// that can recover from any http-relative panics
	// and log the requests to the terminal.
	app.Use(recover.New())
	app.Use(logger.New())

	app.Controller("/", new(ExampleController))

	// http://localhost:8080
	// http://localhost:8080/ping
	// http://localhost:8080/hello
	app.Run(iris.Addr(":8080"))
}

// ExampleController serves the "/", "/ping" and "/hello".
type ExampleController struct {
	// if you build with go1.8 you have to use the mvc package always,
	// otherwise
	// you can, optionally
	// use the type alias `iris.C`,
	// same for
	// context.Context -> iris.Context,
	// mvc.Result -> iris.Result,
	// mvc.Response -> iris.Response,
	// mvc.View -> iris.View
	mvc.C
}

// Get serves
// Method:   GET
// Resource: http://localhost:8080
func (c *ExampleController) Get() mvc.Result {
	return mvc.Response{
		ContentType: "text/html",
		Text:        "<h1>Welcome</h1>",
	}
}

// GetPing serves
// Method:   GET
// Resource: http://localhost:8080/ping
func (c *ExampleController) GetPing() string {
	return "pong"
}

// GetHello serves
// Method:   GET
// Resource: http://localhost:8080/hello
func (c *ExampleController) GetHello() interface{} {
	return map[string]string{"message": "Hello Iris!"}
}

// GetUserBy serves
// Method:   GET
// Resource: http://localhost:8080/user/{username:string}
// By is a reserved "keyword" to tell the framework that you're going to
// bind path parameters in the function's input arguments, and it also
// helps to have "Get" and "GetBy" in the same controller.
//
// func (c *ExampleController) GetUserBy(username string) mvc.Result {
// 	return mvc.View{
// 		Name: "user/username.html",
// 		Data: username,
// 	}
// }

/* Can use more than one, the factory will make sure
that the correct http methods are being registered for each route
for this controller, uncomment these if you want:

func (c *ExampleController) Post() {}
func (c *ExampleController) Put() {}
func (c *ExampleController) Delete() {}
func (c *ExampleController) Connect() {}
func (c *ExampleController) Head() {}
func (c *ExampleController) Patch() {}
func (c *ExampleController) Options() {}
func (c *ExampleController) Trace() {}
*/

/*
func (c *ExampleController) All() {}
//        OR
func (c *ExampleController) Any() {}
*/
