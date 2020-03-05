package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	// Catch a specific error code.
	app.OnErrorCode(iris.StatusInternalServerError, func(ctx iris.Context) {
		ctx.HTML("Message: <b>" + ctx.Values().GetString("message") + "</b>")
	})

	// Catch all error codes [app.OnAnyErrorCode...]

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML(`Click <a href="/my500">here</a> to pretend an HTTP error`)
	})

	app.Get("/my500", func(ctx iris.Context) {
		ctx.Values().Set("message", "this is the error message")
		ctx.StatusCode(500)
	})

	app.Get("/u/{firstname:alphabetical}", func(ctx iris.Context) {
		ctx.Writef("Hello %s", ctx.Params().Get("firstname"))
	})

	// Read more at: https://github.com/kataras/iris/issues/1335
	app.Get("/product-problem", problemExample)

	app.Get("/product-error", func(ctx iris.Context) {
		ctx.Writef("explain the error")
	})

	// http://localhost:8080
	// http://localhost:8080/my500
	// http://localhost:8080/u/gerasimos
	// http://localhost:8080/product-problem
	app.Listen(":8080")
}

func newProductProblem(productName, detail string) iris.Problem {
	return iris.NewProblem().
		// The type URI, if relative it automatically convert to absolute.
		Type("/product-error").
		// The title, if empty then it gets it from the status code.
		Title("Product validation problem").
		// Any optional details.
		Detail(detail).
		// The status error code, required.
		Status(iris.StatusBadRequest).
		// Any custom key-value pair.
		Key("productName", productName)
	// Optional cause of the problem, chain of Problems.
	// Cause(iris.NewProblem().Type("/error").Title("cause of the problem").Status(400))
}

func problemExample(ctx iris.Context) {
	/*
		p := iris.NewProblem().
			Type("/validation-error").
			Title("Your request parameters didn't validate").
			Detail("Optional details about the error.").
			Status(iris.StatusBadRequest).
		 	Key("customField1", customValue1)
		 	Key("customField2", customValue2)
		ctx.Problem(p)

		// OR
		ctx.Problem(iris.Problem{
			"type":   "/validation-error",
			"title":  "Your request parameters didn't validate",
			"detail": "Optional details about the error.",
			"status": iris.StatusBadRequest,
		 	"customField1": customValue1,
		 	"customField2": customValue2,
		})

		// OR
	*/

	// Response like JSON but with indent of "  " and
	// content type of "application/problem+json"
	ctx.Problem(newProductProblem("product name", "problem error details"), iris.ProblemOptions{
		// Optional JSON renderer settings.
		JSON: iris.JSON{
			Indent: "  ",
		},
		// OR
		// Render as XML:
		//
		// RenderXML: true,
		// XML:       iris.XML{Indent: "  "},
		// and ctx.StatusCode(200) to see the result on browser as a user.
		//
		// The below `RetryAfter` field sets the "Retry-After" response header.
		//
		// Can accept:
		// time.Time for HTTP-Date,
		// time.Duration, int64, float64, int for seconds
		// or string for date or duration.
		// Examples:
		// time.Now().Add(5 * time.Minute),
		// 300 * time.Second,
		// "5m",
		//
		RetryAfter: 300,
		// A function that, if specified, can dynamically set
		// retry-after based on the request. Useful for ProblemOptions reusability.
		// Overrides the RetryAfter field.
		//
		// RetryAfterFunc: func(iris.Context) interface{} { [...] }
	})
}
