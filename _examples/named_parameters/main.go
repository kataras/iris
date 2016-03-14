package main

import (
	"strconv"

	"github.com/kataras/iris"
)

func main() {

	iris.Get("/hello/:name", func(c *iris.Context) {
		name := c.Param("name")
		c.Write("Hello " + name)
	})

	iris.Get("/profile/:fullname/friends/:friendID", func(c *iris.Context) {

		fullname := c.Param("fullname")
		friendID, err := c.ParamInt("friendID")
		if err != nil {
		}
		c.HTML("<b> Hello </b>" + fullname + "<b> with friends ID </b>" + strconv.Itoa(friendID))
	})

	iris.Listen(8080)
}
