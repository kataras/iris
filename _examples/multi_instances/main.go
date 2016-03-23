package main

import (
	"github.com/kataras/iris"
)

func main() {
	server1 := iris.New()
	server1.Get("/", func(c *iris.Context) {
		c.Write("Hello from the server1 on :8080")
	})

	server2 := iris.New()
	server2.Get("/", func(c *iris.Context) {
		c.Write("Hello from the server2 on :80")
	})

	// remember that .Listen on Iris is a synchronized method
	//, so it's blocking, you have to run it in go routine
	// when ever you want your code below that to be executed
	go server1.Listen(":8080")
	server2.Listen(":80")

}
