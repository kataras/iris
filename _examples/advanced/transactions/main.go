package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	// subdomains works with all available routers, like other features too.
	app.Adapt(httprouter.New())

	app.Get("/", func(ctx *iris.Context) {
		ctx.BeginTransaction(func(t *iris.Transaction) {
			// OPTIONAl STEP: , if true then the next transictions will not be executed if this transiction fails
			// t.SetScope(iris.RequestTransactionScope)

			// OPTIONAL STEP:
			// create a new custom type of error here to keep track of the status code and reason message
			err := iris.NewTransactionErrResult()

			// we should use t.Context if we want to rollback on any errors lives inside this function clojure.
			t.Context.Text(iris.StatusOK, "Blablabla this should not be sent to the client because we will fill the err with a message and status")

			// virtualize a fake error here, for the shake of the example
			fail := true
			if fail {
				err.StatusCode = iris.StatusInternalServerError
				// NOTE: if empty reason then the default or the custom http error will be fired (like ctx.EmitError)
				err.Reason = "Error: Virtual failure!!"
			}

			// OPTIONAl STEP:
			// but useful if we want to post back an error message to the client if the transaction failed.
			// if the reason is empty then the transaction completed succesfuly,
			// otherwise we rollback the whole response writer's body,
			// headers and cookies, status code and everything lives inside this transaction
			t.Complete(err)
		})

		ctx.BeginTransaction(func(t *iris.Transaction) {
			t.Context.HTML(iris.StatusOK,
				"<h1>This will sent at all cases because it lives on different transaction and it doesn't fails</h1>")
			// * if we don't have any 'throw error' logic then no need of scope.Complete()
		})

		// OPTIONAL, depends on the usage:
		// at any case, what ever happens inside the context's transactions send this to the client
		ctx.HTML(iris.StatusOK, "<h1>Let's add a second html message to the response, "+
			"if the transaction was failed and it was request scoped then this message would "+
			"not been shown. But it has a transient scope(default) so, it is visible as expected!</h1>")
	})

	app.Listen(":8080")
}
