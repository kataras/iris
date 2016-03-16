package main

import "github.com/kataras/iris"

func main() {
	// Let's party
	admin := iris.Party("/admin")
	{
		// add a silly middleware
		admin.UseFunc(func(c *iris.Context) {
			//your authentication logic here...
			authorized := true
			if authorized {
				c.Next()
			} else {
				c.SendStatus(401, "Not authorized for some reason")
			}

		})
		admin.Get("/", func(c *iris.Context) {})
		admin.Get("/dashboard", func(c *iris.Context) {})
		admin.Delete("/delete/:userId", func(c *iris.Context) {})
	}

	println("Iris is listening on :8080")
	iris.Listen("8080")

}
