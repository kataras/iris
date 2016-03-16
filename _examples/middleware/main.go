package main

import (
	"fmt"
	"github.com/kataras/iris"
)

func main() {

	// register global middleware, you can pass more than one handler comma separated
	iris.UseFunc(func(c *iris.Context) {
		fmt.Println("(1)Global logger: ", c.Request.URL.Path)
		c.Next()
	})

	// register a global structed iris.Handler as middleware
	myglobal := MyGlobalMiddlewareStructed{loggerId: "my logger id"}
	iris.Use(myglobal)

	// register route's middleware
	iris.Get("/home", func(c *iris.Context) {
		fmt.Println("(1)HOME logger for /home")
		c.Next()
	}, func(c *iris.Context) {
		fmt.Println("(2)HOME logger for /home")
		c.Next()
	}, func(c *iris.Context) {
		c.Write("Hello from /home")
	})

	// register a structed iris.Handler as middleware to the route
	iris.Get("/hello", iris.ToHandlerFunc(myglobal))

	iris.Listen("8080")
}

// a silly example
type MyGlobalMiddlewareStructed struct {
	loggerId string
}

var _ iris.Handler = &MyGlobalMiddlewareStructed{}

//Important staff, iris middleware must implement the iris.Handler interface which is:
func (m MyGlobalMiddlewareStructed) Serve(c *iris.Context) {
	fmt.Println("Hello from logger with id: ", m.loggerId)
	c.Next()
}
