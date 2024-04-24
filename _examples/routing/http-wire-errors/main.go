package main

import (
	"github.com/kataras/iris/v12"
	// IMPORTANT, import this sub-package.
	// Note tht it does NOT break compatibility with the
	// standard "errors" package as the New,
	// Is, As, Unwrap functions are aliases to the standard package.
	"github.com/kataras/iris/v12/x/errors"
)

// Optionally, register custom error codes.
//
// The default list of error code names:
// errors.Cancelled
// errors.Unknown
// errors.InvalidArgument
// errors.DeadlineExceeded
// errors.NotFound
// errors.AlreadyExists
// errors.PermissionDenied
// errors.Unauthenticated
// errors.ResourceExhausted
// errors.FailedPrecondition
// errors.Aborted
// errors.OutOfRange
// errors.Unimplemented
// errors.Internal
// errors.Unavailable
// errors.DataLoss
var (
	Custom = errors.Register("CUSTOM_CANONICAL_ERROR_NAME", iris.StatusBadRequest)
)

func main() {
	app := iris.New()

	// Custom error code name.
	app.Get("/custom", fireCustomErrorCodeName)

	// Send a simple 400 request with message and an error
	// or with more details and data.
	app.Post("/invalid_argument", fireInvalidArgument)

	// Compatibility with the iris.Problem type (and any other custom type).
	app.Get("/problem", fireErrorWithProblem)

	app.Listen(":8080")
}

func fireCustomErrorCodeName(ctx iris.Context) {
	Custom.Details(ctx, "message", "details with arguments: %s", "an argument")
}

func fireInvalidArgument(ctx iris.Context) {
	var req = struct {
		Username string `json:"username"`
	}{}
	if err := ctx.ReadJSON(&req); err != nil {
		errors.InvalidArgument.Err(ctx, err)
		return
	}

	ctx.WriteString(req.Username)

	// Other examples: errors.InvalidArgument/NotFound/Internal and e.t.c.
	// .Message(ctx, "message %s", "optional argument")
	// .Details(ctx, "message", "details %s", "optional details argument")
	// .Data(ctx, "message", anyTypeOfValue)
	// .DataWithDetails(ctx, "unable to read the body", "malformed json", iris.Map{"custom": "data of any type"})
	// .Log(ctx, "message %s", "optional argument")
	// .LogErr(ctx, err)
}

func fireErrorWithProblem(ctx iris.Context) {
	myCondition := true
	if myCondition {
		problem := iris.NewProblem().
			// The type URI, if relative it automatically convert to absolute.
			Type("/product-error").
			// The title, if empty then it gets it from the status code.
			Title("Product validation problem").
			// Any optional details.
			Detail("details about the product error").
			// The status error code of the problem, can be optional here.
			// Status(iris.StatusBadRequest).
			// Any custom key-value pair.
			Key("product_name", "the product name")

		errors.InvalidArgument.Data(ctx, "unable to process the request", problem)
		return

		/* Prints to the client:
		{
		  "http_error_code": {
		    "canonical_name": "INVALID_ARGUMENT",
		    "status": 400
		  },
		  "message": "unable to process the request",
		  "data": {
		    "detail": "details about the product error",
		    "product_name": "the product name",
		    "title": "Product validation problem",
		    "type": "/product-error"
		  }
		}
		*/
	}

}
