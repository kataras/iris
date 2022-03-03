package main

import (
	"fmt"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"
)

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()
	app.Get("/", fireCustomValidationError)
	app.Get("/multi", fireCustomValidationErrors)
	app.Get("/invalid", fireInvalidError)
	return app
}

type MyValidationError struct {
	Field     string      `json:"field"`
	Value     interface{} `json:"value"`
	Reason    string      `json:"reason"`
	Timestamp int64       `json:"timestamp"`
}

func (err MyValidationError) Error() string {
	return fmt.Sprintf("field %q got invalid value of %v: reason: %s", err.Field, err.Value, err.Reason)
}

// Error, GetField, GetValue and GetReason completes
// the x/errors.ValidationError interface which can be used
// for faster rendering without the necessity of registering a custom
// type (see at the end of the example).
//
// func (err MyValidationError) GetField() string {
// 	return err.Field
// }
//
// func (err MyValidationError) GetValue() interface{} {
// 	return err.Value
// }
//
// func (err MyValidationError) GetReason() string {
// 	return err.Reason
// }

const shouldFail = true

func fireCustomValidationError(ctx iris.Context) {
	if shouldFail {
		err := MyValidationError{
			Field:     "username",
			Value:     "",
			Reason:    "empty string",
			Timestamp: time.Now().Unix(),
		}

		// The "validation" field, when used, is always rendering as
		// a JSON array, NOT a single object.
		errors.InvalidArgument.Err(ctx, err)
		return
	}

	ctx.WriteString("OK")
}

// Optionally register custom types that you may need
// to be rendered as validation errors if the given "ErrorCodeName.Err.err"
// input parameter is matched with one of these. Register once, at initialiation.
func init() {
	mapper := errors.NewValidationErrorTypeMapper(MyValidationError{} /*, OtherCustomType{} */)
	errors.RegisterValidationErrorMapper(mapper)
}

// A custom type of the example validation error type
// in order to complete the error interface, so it can be
// pass through the errors.InvalidArgument.Err method.
type MyValidationErrors []MyValidationError

func (m MyValidationErrors) Error() string {
	return "to be an error"
}

func fireCustomValidationErrors(ctx iris.Context) {
	if shouldFail {
		errs := MyValidationErrors{
			{
				Field:     "username",
				Value:     "",
				Reason:    "empty string",
				Timestamp: time.Now().Unix(),
			},
			{
				Field:     "birth_date",
				Value:     "2022-01-01",
				Reason:    "too young",
				Timestamp: time.Now().Unix(),
			},
		}
		errors.InvalidArgument.Err(ctx, errs)
		return
	}

	ctx.WriteString("OK")
}

func fireInvalidError(ctx iris.Context) {
	if shouldFail {
		errors.InvalidArgument.Err(ctx, fmt.Errorf("just a custom error text"))
		return
	}

	ctx.WriteString("OK")
}
