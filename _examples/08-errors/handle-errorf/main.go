// errors.HandleErrorf: keep the classification, add the call site.
//
// A domain sentinel says what went wrong ("plan not found") but not which
// resource. HandleErrorf wraps the error with %w and then handles it exactly
// like HandleError: the central map still recognizes the sentinel, so the
// status stays 404, while the rendered message carries the call-site detail
// and the original error travels in the response's data list (and the logs).
//
// Learn: add call-site detail to a mapped error without changing the status
// code the central map chose for it.
// Before: ../central-error-map
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/plans/starter    # 200
//	curl -i http://localhost:8080/plans/42         # 404, message "plan with id: 42", the sentinel under "data"
//	curl -i http://localhost:8080/plans/42/plain   # 404, message is only "plan not found"
//
// Next: ../validation-errors
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	opts := errors.NewOptions().MapErrors(errors.NotFound, ErrPlanNotFound)

	app := iris.New()
	app.Use(opts.Handler())

	// 404 with "message": "plan with id: 42" and the sentinel kept under
	// "data": [{"message": "plan not found"}].
	app.Get("/plans/{id}", func(ctx iris.Context) {
		id := ctx.Params().GetString("id")

		p, err := plan(id)
		if err != nil {
			errors.HandleErrorf(ctx, err, "plan with id: %s", id)
			return
		}

		ctx.OK(p, nil)
	})

	// The same 404, but the message is only the sentinel's text:
	// "message": "plan not found". Nothing tells the client which id failed.
	app.Get("/plans/{id}/plain", func(ctx iris.Context) {
		p, err := plan(ctx.Params().GetString("id"))
		ctx.OK(p, err)
	})

	app.Listen(":8080")
}

// ErrPlanNotFound is the domain sentinel, mapped once in main.
var ErrPlanNotFound = errors.New("plan not found")

// plan is the service call: the same error for every missing id.
func plan(id string) (any, error) {
	if id == "starter" {
		return iris.Map{"id": id, "seats": 3}, nil
	}

	return nil, ErrPlanNotFound
}
