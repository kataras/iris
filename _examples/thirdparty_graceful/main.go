package main

import (
	"github.com/kataras/iris"
	"gopkg.in/tylerb/graceful.v1"
	"time"
)

func main() {
	api := iris.New()
	api.Get("/home", func(c *iris.Context) {
		c.Write("Hello from the /home")
	})

	graceful.Run(":8080", 10*time.Second, api)
}
