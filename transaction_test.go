package iris_test

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/httptest"

	"testing"
)

func TestTransaction(t *testing.T) {
	app := iris.New()
	app.Adapt(newTestNativeRouter())

	firstTransactionFailureMessage := "Error: Virtual failure!!!"
	secondTransactionSuccessHTMLMessage := "<h1>This will sent at all cases because it lives on different transaction and it doesn't fails</h1>"
	persistMessage := "<h1>I persist show this message to the client!</h1>"

	maybeFailureTransaction := func(shouldFail bool, isRequestScoped bool) func(t *iris.Transaction) {
		return func(t *iris.Transaction) {
			// OPTIONAl, the next transactions and the flow will not be skipped if this transaction fails
			if isRequestScoped {
				t.SetScope(iris.RequestTransactionScope)
			}

			// OPTIONAL STEP:
			// create a new custom type of error here to keep track of the status code and reason message
			err := iris.NewTransactionErrResult()

			t.Context.Text(iris.StatusOK, "Blablabla this should not be sent to the client because we will fill the err with a message and status")

			fail := shouldFail

			if fail {
				err.StatusCode = iris.StatusInternalServerError
				err.Reason = firstTransactionFailureMessage
			}

			// OPTIONAl STEP:
			// but useful if we want to post back an error message to the client if the transaction failed.
			// if the reason is empty then the transaction completed successfully,
			// otherwise we rollback the whole response body and cookies and everything lives inside the transaction.Request.
			t.Complete(err)
		}
	}

	successTransaction := func(scope *iris.Transaction) {
		if scope.Context.Request.RequestURI == "/failAllBecauseOfRequestScopeAndFailure" {
			t.Fatalf("We are inside successTransaction but the previous REQUEST SCOPED TRANSACTION HAS FAILED SO THiS SHOULD NOT BE RAN AT ALL")
		}
		scope.Context.HTML(iris.StatusOK,
			secondTransactionSuccessHTMLMessage)
		// * if we don't have any 'throw error' logic then no need of scope.Complete()
	}

	persistMessageHandler := func(ctx *iris.Context) {
		// OPTIONAL, depends on the usage:
		// at any case, what ever happens inside the context's transactions send this to the client
		ctx.HTML(iris.StatusOK, persistMessage)
	}

	app.Get("/failFirsTransactionButSuccessSecondWithPersistMessage", func(ctx *iris.Context) {
		ctx.BeginTransaction(maybeFailureTransaction(true, false))
		ctx.BeginTransaction(successTransaction)
		persistMessageHandler(ctx)
	})

	app.Get("/failFirsTransactionButSuccessSecond", func(ctx *iris.Context) {
		ctx.BeginTransaction(maybeFailureTransaction(true, false))
		ctx.BeginTransaction(successTransaction)
	})

	app.Get("/failAllBecauseOfRequestScopeAndFailure", func(ctx *iris.Context) {
		ctx.BeginTransaction(maybeFailureTransaction(true, true))
		ctx.BeginTransaction(successTransaction)
	})

	customErrorTemplateText := "<h1>custom error</h1>"
	app.OnError(iris.StatusInternalServerError, func(ctx *iris.Context) {
		ctx.Text(iris.StatusInternalServerError, customErrorTemplateText)
	})

	failureWithRegisteredErrorHandler := func(ctx *iris.Context) {
		ctx.BeginTransaction(func(transaction *iris.Transaction) {
			transaction.SetScope(iris.RequestTransactionScope)
			err := iris.NewTransactionErrResult()
			err.StatusCode = iris.StatusInternalServerError // set only the status code in order to execute the registered template
			transaction.Complete(err)
		})

		ctx.Text(iris.StatusOK, "this will not be sent to the client because first is requested scope and it's failed")
	}

	app.Get("/failAllBecauseFirstTransactionFailedWithRegisteredErrorTemplate", failureWithRegisteredErrorHandler)

	e := httptest.New(app, t)

	e.GET("/failFirsTransactionButSuccessSecondWithPersistMessage").
		Expect().
		Status(iris.StatusOK).
		ContentType("text/html", app.Config.Charset).
		Body().
		Equal(secondTransactionSuccessHTMLMessage + persistMessage)

	e.GET("/failFirsTransactionButSuccessSecond").
		Expect().
		Status(iris.StatusOK).
		ContentType("text/html", app.Config.Charset).
		Body().
		Equal(secondTransactionSuccessHTMLMessage)

	e.GET("/failAllBecauseOfRequestScopeAndFailure").
		Expect().
		Status(iris.StatusInternalServerError).
		Body().
		Equal(firstTransactionFailureMessage)

	e.GET("/failAllBecauseFirstTransactionFailedWithRegisteredErrorTemplate").
		Expect().
		Status(iris.StatusInternalServerError).
		Body().
		Equal(customErrorTemplateText)
}
