/*Package main shows how to add done handlers in an MVC application without
the necessity of `ctx.Next()` inside the controller's methods.

When we want the `Done` handlers of that specific mvc app's `Party`
to be executed but we don't want to add `ctx.Next()` on the `exampleController#EndRequest`*/
package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) { ctx.Redirect("/example") })

	exampleRouter := app.Party("/example")
	{
		exampleRouter.SetExecutionRules(iris.ExecutionRules{
			Done: iris.ExecutionOptions{Force: true},
		})

		exampleRouter.Done(doneHandler)

		m := mvc.New(exampleRouter)
		m.Handle(&exampleController{})
	}

	app.Listen(":8080")
}

func doneHandler(ctx iris.Context) {
	ctx.WriteString("\nFrom Done Handler")
}

type exampleController struct{}

func (c *exampleController) Get() string {
	return "From Main Handler"
	// Note that here we don't binding the `Context`, and we don't call its `Next()`
	// function in order to call the `doneHandler`,
	// this is done automatically for us because we changed the execution rules with the
	// `SetExecutionRules`.
	//
	// Therefore the final output is:
	// From Main Handler
	// From Done Handler
}
