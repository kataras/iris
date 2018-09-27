package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	// subdomains works with all available routers, like other features too.

	app.Get("/", func(ctx context.Context) {
		ctx.BeginTransaction(func(t *context.Transaction) {
			// OPTIONAl STEP: , if true then the next transictions will not be executed if this transiction fails
			// t.SetScope(context.RequestTransactionScope)

			// OPTIONAL STEP:
			// create a new custom type of error here to keep track of the status code and reason message
			err := context.NewTransactionErrResult()

			// we should use t.Context if we want to rollback on any errors lives inside this function clojure.
			t.Context().Text("Blablabla this should not be sent to the client because we will fill the err with a message and status")

			// virtualize a fake error here, for the shake of the example
			fail := true
			if fail {
				err.StatusCode = iris.StatusInternalServerError
				// NOTE: if empty reason then the default or the custom http error will be fired (like ctx.FireStatusCode)
				err.Reason = "Error: Virtual failure!!"
			}

			// OPTIONAl STEP:
			// but useful if we want to post back an error message to the client if the transaction failed.
			// if the reason is empty then the transaction completed successfully,
			// otherwise we rollback the whole response writer's body,
			// headers and cookies, status code and everything lives inside this transaction
			t.Complete(err)
		})

		ctx.BeginTransaction(func(t *context.Transaction) {
			t.Context().HTML("<h1>This will sent at all cases because it lives on different transaction and it doesn't fails</h1>")
			// * if we don't have any 'throw error' logic then no need of scope.Complete()
		})

		// OPTIONALLY, depends on the usage:
		// at any case, what ever happens inside the context's transactions send this to the client
		ctx.HTML("<h1>Let's add a second html message to the response, " +
			"if the transaction was failed and it was request scoped then this message would " +
			"not been shown. But it has a transient scope(default) so, it is visible as expected!</h1>")
	})

	app.Run(iris.Addr(":8080"))
}
