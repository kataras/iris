package main

import (
	"io"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())

	// Custom Render Policy to override or create new content-type render
	// i,e: "text/html" with a prefix,
	// we will just write to the writer and return false
	// to continue to the next contentType-matched renderer if any.
	app.Adapt(iris.RenderPolicy(func(out io.Writer, contentType string, binding interface{}, options ...map[string]interface{}) (bool, error) {

		if contentType == "text/html" {
			if content, ok := binding.(string); ok {
				out.Write([]byte("<pre>My Custom prefix</pre><br/>" + content))
			}

		}
		// continue to the next, no error
		// note: if we wanted to stop here we would return true instead of false.
		return false, nil
	}))

	app.Get("", func(ctx *iris.Context) {

		// These content-types are not managed by our RenderPolicy:
		// text, binary and html content-types are
		// not rendered via serializers, you have to
		// use the ctx.Render functions instead.
		// so something like this:
		// ctx.Text(iris.StatusOK, "my text content body here!")
		// will NOT work with out custom render policy.
		ctx.Render("text/html",
			"my text content body here!")
	})

	app.Listen(":8080")
}
