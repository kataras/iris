package examples

import (
	"bufio"
	"fmt"

	"github.com/iris-contrib/middleware/basicauth"
	"github.com/kataras/iris"
	"github.com/valyala/fasthttp"
)

// IrisHandler creates fasthttp.RequestHandler using Iris web framework.
func IrisHandler() fasthttp.RequestHandler {
	api := iris.New()

	api.Get("/things", func(c *iris.Context) {
		c.JSON(iris.StatusOK, []interface{}{
			iris.Map{
				"name":        "foo",
				"description": "foo thing",
			},
			iris.Map{
				"name":        "bar",
				"description": "bar thing",
			},
		})
	})

	api.Post("/redirect", func(c *iris.Context) {
		c.Redirect("/things", iris.StatusFound)
	})

	api.Get("/params/:x/:y", func(c *iris.Context) {
		c.JSON(iris.StatusOK, iris.Map{
			"x":  c.Param("x"),
			"y":  c.Param("y"),
			"q":  c.URLParam("q"),
			"p1": c.FormValueString("p1"),
			"p2": c.FormValueString("p2"),
		})
	})

	auth := basicauth.Default(map[string]string{
		"ford": "betelgeuse7",
	})

	api.Get("/auth", auth, func(c *iris.Context) {
		c.Write("authenticated!")
	})

	api.Post("/session/set", func(c *iris.Context) {
		sess := iris.Map{}

		if err := c.ReadJSON(&sess); err != nil {
			panic(err.Error())
		}

		c.Session().Set("name", sess["name"])
	})

	api.Get("/session/get", func(c *iris.Context) {
		name := c.Session().GetString("name")

		c.JSON(iris.StatusOK, iris.Map{
			"name": name,
		})
	})

	api.Get("/stream", func(c *iris.Context) {
		c.StreamWriter(func(w *bufio.Writer) {
			for i := 0; i < 10; i++ {
				fmt.Fprintf(w, "%d", i)

				if err := w.Flush(); err != nil {
					return
				}
			}
		})
	})

	api.Post("/stream", func(c *iris.Context) {
		c.Write(string(c.Request.Body()))
	})

	sub := api.Party("subdomain.")

	sub.Post("/set", func(c *iris.Context) {
		c.Session().Set("message", "hello from subdomain")
	})

	sub.Get("/get", func(c *iris.Context) {
		c.Write(c.Session().GetString("message"))
	})

	api.Build()
	return api.Router
}
