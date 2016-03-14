package main

import (
	"strconv"

	"github.com/kataras/iris"
)

func main() {

	// Match to /hello/iris
	// Not match to /hello or /hello/ or /hello/iris/something
	iris.Get("/hello/:name", func(c *iris.Context) {
		// Retrieve the parameter name
		name := c.Param("name")
		c.Write("Hello " + name)
	})

	// Match to /profile/iris/friends/1
	// Not match to /profile/ , /profile/iris ,
	// Not match to /profile/iris/friends,  /profile/iris/friends ,
	// Not match to /profile/iris/friends/2/something
	iris.Get("/profile/:fullname/friends/:friendID", func(c *iris.Context) {
		// Retrieve the parameters fullname and friendID
		fullname := c.Param("fullname")
		friendID, err := c.ParamInt("friendID")
		if err != nil {
			// Do something with the error
		}
		c.HTML("<b> Hello </b>" + fullname + "<b> with friends ID </b>" + strconv.Itoa(friendID))
	})

	// Listen to port 8080 for example localhost:8080
	iris.Listen(":8080")
}
