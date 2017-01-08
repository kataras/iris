// This example is for Iris v6(HTTP/2).
// The only httpexpect change-> from: httpexpect.NewFastBinder(handler) to: httpexpect.NewBinder(handler).
//
// For Iris v5(fasthttp) example look here:
// https://github.com/gavv/httpexpect/blob/cccd8d0064fdfdafa29a83f7304fb9747f0b29e5/_examples/iris.go
package examples

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/iris-contrib/middleware/basicauth"
	"github.com/kataras/iris"
)

// IrisHandler creates fasthttp.RequestHandler using Iris web framework.
func IrisHandler() http.Handler {
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

	api.Post("/params/:x/:y", func(c *iris.Context) {
		for k, v := range c.FormValues() {
			print(k)
			for i := range v {
				println(v[i])
			}
		}
		c.JSON(iris.StatusOK, iris.Map{
			"x":  c.Param("x"),
			"y":  c.Param("y"),
			"q":  c.URLParam("q"),
			"p1": c.FormValue("p1"),
			"p2": c.FormValue("p2"),
		})
	})

	auth := basicauth.Default(map[string]string{
		"ford": "betelgeuse7",
	})

	api.Get("/auth", auth, func(c *iris.Context) {
		c.Writef("authenticated!")
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
		// return true to continue, return false to stop and flush
		c.StreamWriter(func(w io.Writer) bool {
			for i := 0; i < 10; i++ {
				fmt.Fprintf(w, "%d", i)
			}
			return false
		})
		// if we had to write here then the StreamWriter callback should return true.
	})

	api.Post("/stream", func(c *iris.Context) {
		// Optional: Limit the request body size by c.SetMaxRequestBodySize(20 << 10)
		// or by middleware api.Post("/stream", iris.LimitRequestBodySize(20 << 10), func(c *iris.Context){...})
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.EmitError(iris.StatusBadRequest)
			return
		}
		c.Write(body)
	})

	sub := api.Party("subdomain.")

	sub.Post("/set", func(c *iris.Context) {
		c.Session().Set("message", "hello from subdomain")
	})

	sub.Get("/get", func(c *iris.Context) {
		c.Writef(c.Session().GetString("message"))
	})

	api.Build()
	return api.Router
}
