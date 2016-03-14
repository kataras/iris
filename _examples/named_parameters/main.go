package main

import "github.com/kataras/iris"

func main() {

	iris.Get("/hello/:name", func(c *iris.Context) {
		name := c.Param("name")
		c.Write("Hello " + name)
	})

	iris.Listen(":8080")
}
