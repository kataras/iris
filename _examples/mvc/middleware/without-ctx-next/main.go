/*Package main is a simple example of the behavior change of the execution flow of the handlers,
normally we need the `ctx.Next()` to call the next handler in a route's handler chain,
but with the new `ExecutionRules` we can change this default behavior.
Please read below before continue.

The `Party#SetExecutionRules` alters the execution flow of the route handlers outside of the handlers themselves.

For example, if for some reason the desired result is the (done or all) handlers to be executed no matter what
even if no `ctx.Next()` is called in the previous handlers, including the begin(`Use`),
the main(`Handle`) and the done(`Done`) handlers themselves, then:
Party#SetExecutionRules(iris.ExecutionRules {
  Begin: iris.ExecutionOptions{Force: true},
  Main:  iris.ExecutionOptions{Force: true},
  Done:  iris.ExecutionOptions{Force: true},
})

Note that if `true` then the only remained way to "break" the handler chain is by `ctx.StopExecution()` now that `ctx.Next()` does not matter.

These rules are per-party, so if a `Party` creates a child one then the same rules will be applied to that as well.
Reset of these rules (before `Party#Handle`) can be done with `Party#SetExecutionRules(iris.ExecutionRules{})`.

The most common scenario for its use can be found inside Iris MVC Applications;
when we want the `Done` handlers of that specific mvc app's `Party`
to be executed but we don't want to add `ctx.Next()` on the `exampleController#EndRequest`*/
package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) { ctx.Redirect("/example") })

	// example := app.Party("/example")
	// example.SetExecutionRules && mvc.New(example) or...
	m := mvc.New(app.Party("/example"))

	// IMPORTANT
	// All options can be filled with Force:true, they all play nice together.
	m.Router.SetExecutionRules(iris.ExecutionRules{
		// Begin:  <- from `Use[all]` to `Handle[last]` future route handlers, execute all, execute all even if `ctx.Next()` is missing.
		// Main:   <- all `Handle` future route handlers, execute all >> >>.
		Done: iris.ExecutionOptions{Force: true}, // <- from `Handle[last]` to `Done[all]` future route handlers, execute all >> >>.
	})
	m.Router.Done(doneHandler)
	// m.Router.Done(...)
	// ...
	//

	m.Handle(&exampleController{})

	app.Run(iris.Addr(":8080"))
}

func doneHandler(ctx iris.Context) {
	ctx.WriteString("\nFrom Done Handler")
}

type exampleController struct{}

func (c *exampleController) Get() string {
	return "From Main Handler"
	// Note that here we don't binding the `Context`, and we don't call its `Next()`
	// function in order to call the `doneHandler`,
	// this is done automatically for us because we changed the execution rules with the `SetExecutionRules`.
	//
	// Therefore the final output is:
	// From Main Handler
	// From Done Handler
}
