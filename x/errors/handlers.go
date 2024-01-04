package errors

import (
	stdContext "context"
	"net/http"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/x/pagination"
)

// Handle handles a generic response and error from a service call and sends a JSON response to the context.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
func Handle(ctx *context.Context, resp interface{}, err error) bool {
	if HandleError(ctx, err) {
		return false
	}

	ctx.StatusCode(http.StatusOK)

	if resp != nil {
		if ctx.JSON(resp) != nil {
			return false
		}
	}

	return true
}

// IDPayload is a simple struct which describes a json id value.
type IDPayload[T string | int] struct {
	ID T `json:"id"`
}

// HandleCreate handles a create operation and sends a JSON response with the created resource to the client.
// It returns a boolean value indicating whether the handle was successful or not.
//
// If the "respOrID" response is not nil, it sets the status code to 201 (Created) and sends the response as a JSON payload,
// however if the given "respOrID" is a string or an int, it sends the response as a JSON payload of {"id": resp}.
// If the "err" error is not nil, it calls HandleError to send an appropriate error response to the client.
// It sets the status code to 201 (Created) and sends any response as a JSON payload,
func HandleCreate(ctx *context.Context, respOrID any, err error) bool {
	if HandleError(ctx, err) {
		return false
	}

	ctx.StatusCode(http.StatusCreated)

	if respOrID != nil {
		switch responseValue := respOrID.(type) {
		case string:
			if ctx.JSON(IDPayload[string]{ID: responseValue}) != nil {
				return false
			}
		case int:
			if ctx.JSON(IDPayload[int]{ID: responseValue}) != nil {
				return false
			}
		default:
			if ctx.JSON(responseValue) != nil {
				return false
			}
		}
	}

	return true
}

// HandleUpdate handles an update operation and sends a status code to the client.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// If the updated value is true, it sets the status code to 204 (No Content).
// If the updated value is false, it sets the status code to 304 (Not Modified).
func HandleUpdate(ctx *context.Context, updated bool, err error) bool {
	if HandleError(ctx, err) {
		return false
	}

	if updated {
		ctx.StatusCode(http.StatusNoContent)
	} else {
		ctx.StatusCode(http.StatusNotModified)
	}

	return true
}

// HandleDelete handles a delete operation and sends a status code to the client.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// If the deleted value is true, it sets the status code to 204 (No Content).
// If the deleted value is false, it sets the status code to 304 (Not Modified).
func HandleDelete(ctx *context.Context, deleted bool, err error) bool {
	return HandleUpdate(ctx, deleted, err)
}

// HandleDelete handles a delete operation and sends a status code to the client.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// It sets the status code to 204 (No Content).
func HandleDeleteNoContent(ctx *context.Context, err error) bool {
	return HandleUpdate(ctx, true, err)
}

// ResponseFunc is a function which takes a context and a generic type T and returns a generic type R and an error.
// It is used to bind a request payload to a generic type T and call a service function with it.
type ResponseFunc[T, R any] interface {
	func(stdContext.Context, T) (R, error)
}

// ResponseOnlyErrorFunc is a function which takes a context and a generic type T and returns an error.
// It is used to bind a request payload to a generic type T and call a service function with it.
// It is used for functions which do not return a response.
type ResponseOnlyErrorFunc[T any] interface {
	func(stdContext.Context, T) error
}

func bindResponse[T, R any, F ResponseFunc[T, R]](ctx *context.Context, fn F, fnInput ...T) (R, bool) {
	var req T
	switch len(fnInput) {
	case 0:
		err := ctx.ReadJSON(&req)
		if err != nil {
			var resp R
			return resp, !HandleError(ctx, err)
		}
	case 1:
		req = fnInput[0]
	default:
		panic("invalid number of arguments")
	}

	resp, err := fn(ctx, req)
	return resp, !HandleError(ctx, err)
}

// OK handles a generic response and error from a service call and sends a JSON response to the context.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// It sets the status code to 200 (OK) and sends any response as a JSON payload.
//
// Useful for Get/List/Fetch operations.
func OK[T, R any, F ResponseFunc[T, R]](ctx *context.Context, fn F, fnInput ...T) bool { // or Fetch.
	resp, ok := bindResponse(ctx, fn, fnInput...)
	if !ok {
		return false
	}

	return Handle(ctx, resp, nil)
}

// Create handles a create operation and sends a JSON response with the created resource to the client.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// It sets the status code to 201 (Created) and sends any response as a JSON payload
// note that if the response is a string, then it sends an {"id": resp} JSON payload).
//
// Useful for Insert operations.
func Create[T, R any, F ResponseFunc[T, R]](ctx *context.Context, fn F, fnInput ...T) bool {
	resp, ok := bindResponse(ctx, fn, fnInput...)
	if !ok {
		return false
	}

	return HandleCreate(ctx, resp, nil)
}

// NoContent handles a generic response and error from a service call and sends a JSON response to the context.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// It sets the status code to 204 (No Content).
//
// Useful for Update and Deletion operations.
func NoContent[T any, F ResponseOnlyErrorFunc[T]](ctx *context.Context, fn F, fnInput ...T) bool {
	toFn := func(c stdContext.Context, req T) (bool, error) {
		return true, fn(ctx, req)
	}

	return NoContentOrNotModified(ctx, toFn, fnInput...)
}

// NoContent handles a generic response and error from a service call and sends a JSON response to the context.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// If the response is true, it sets the status code to 204 (No Content).
// If the response is false, it sets the status code to 304 (Not Modified).
//
// Useful for Update and Deletion operations.
func NoContentOrNotModified[T any, F ResponseFunc[T, bool]](ctx *context.Context, fn F, fnInput ...T) bool {
	resp, ok := bindResponse(ctx, fn, fnInput...)
	if !ok {
		return false
	}

	return HandleUpdate(ctx, bool(resp), nil)
}

// ReadPayload reads a JSON payload from the context and returns it as a generic type T.
// It also returns a boolean value indicating whether the read was successful or not.
// If the read fails, it sends an appropriate error response to the client.
func ReadPayload[T any](ctx *context.Context) (T, bool) {
	var payload T
	err := ctx.ReadJSON(&payload)
	if err != nil {
		if vErrs, ok := AsValidationErrors(err); ok {
			InvalidArgument.Data(ctx, "validation failure", vErrs)
		} else {
			InvalidArgument.Details(ctx, "unable to parse body", err.Error())
		}
		return payload, false
	}

	return payload, true
}

// ReadQuery reads URL query values from the context and returns it as a generic type T.
// It also returns a boolean value indicating whether the read was successful or not.
// If the read fails, it sends an appropriate error response to the client.
func ReadQuery[T any](ctx *context.Context) (T, bool) {
	var payload T
	err := ctx.ReadQuery(&payload)
	if err != nil {
		if vErrs, ok := AsValidationErrors(err); ok {
			InvalidArgument.Data(ctx, "validation failure", vErrs)
		} else {
			InvalidArgument.Details(ctx, "unable to parse query", err.Error())
		}
		return payload, false
	}

	return payload, true
}

// ReadPaginationOptions reads the ListOptions from the URL Query and
// any filter options of generic T from the request body.
func ReadPaginationOptions[T /* T is FilterOptions */ any](ctx *context.Context) (pagination.ListOptions, T, bool) {
	list, ok := ReadQuery[pagination.ListOptions](ctx)
	if !ok {
		var t T
		return list, t, false
	}

	filter, ok := ReadPayload[T](ctx)
	if !ok {
		var t T
		return list, t, false
	}

	return list, filter, true
}
