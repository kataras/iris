/*Package main is a simple example of the behavior change of the execution flow of the handlers,
normally we need the `ctx.Next()` to call the next handler in a route's handler chain,
but with the `ExecutionRules` we can change this default behavior.
Please read below before continue.

The `Party#SetExecutionRules` alters the execution flow of the route handlers.

For example, if for some reason the desired result is the (done or all) handlers
to be executed no matter what, even if no `ctx.Next()` is called in the previous handlers:

app.SetExecutionRules(iris.ExecutionRules {
  Begin: iris.ExecutionOptions{Force: true}, # begin handlers(.Use)
  Main:  iris.ExecutionOptions{Force: true}, # main handler (.Handle/Get...)
  Done:  iris.ExecutionOptions{Force: true}, # done handlers (.Done)
})

Note that if `true` then the only remained way to "break" the handler chain
is by calling the `ctx.StopExecution()` (now that `ctx.Next()` doesn't even matter).

These rules are per-party, so if a `Party` creates a child one then
the same rules will be applied to that as well.

Reset of these rules to their defaults (before `Party#Handle`) can be done
with `Party#SetExecutionRules(iris.ExecutionRules{})`.

*/
package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	app.SetExecutionRules(iris.ExecutionRules{
		// * From `Use[all]` to `Handle[last]` future route handlers,
		// execute all (even if `ctx.Next()` is missing):
		// Begin: true,
		//
		// * All `Handle` future route handlers, execute all:
		// Main: true,
		//
		// * From `Handle[last]` to `Done[last]` future route handlers, execute all:
		Done: iris.ExecutionOptions{Force: true},
	})
	app.Done(doneHandler)

	app.Get("/", mainHandler)

	// http://localhost:8080
	app.Listen(":8080")
}

func mainHandler(ctx iris.Context) {
	ctx.WriteString("From Main Handler\n")
	// ctx.Next() is not required now that we have declared
	// Done: iris.ExecutionOptions{Force: true}.
}

func doneHandler(ctx iris.Context) {
	ctx.WriteString("From Done Handler\n")
}
