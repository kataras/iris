package errors

import (
	"net/http"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/x/pagination"
)

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

// Handle handles a generic response and error from a service call and sends a JSON response to the context.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
func Handle(ctx *context.Context, resp interface{}, err error) bool {
	if HandleError(ctx, err) {
		return false
	}

	return ctx.JSON(resp) == nil
}

// IDPayload is a simple struct which describes a json id value.
type IDPayload struct {
	ID string `json:"id"`
}

// HandleCreate handles a create operation and sends a JSON response with the created id to the client.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// If the id is not empty, it sets the status code to 201 (Created) and sends the id as a JSON payload.
func HandleCreate(ctx *context.Context, id string, err error) bool {
	if HandleError(ctx, err) {
		return false
	}

	ctx.StatusCode(http.StatusCreated)

	if id != "" {
		ctx.JSON(IDPayload{ID: id})
	}

	return true
}

// HandleCreateResponse handles a create operation and sends a JSON response with the created resource to the client.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// If the response is not nil, it sets the status code to 201 (Created) and sends the response as a JSON payload.
func HandleCreateResponse(ctx *context.Context, resp interface{}, err error) bool {
	if HandleError(ctx, err) {
		return false
	}

	ctx.StatusCode(http.StatusCreated)
	if resp != nil {
		return ctx.JSON(resp) == nil
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
// It sets the status code to 204 (No Content).
func HandleDelete(ctx *context.Context, err error) bool {
	if HandleError(ctx, err) {
		return false
	}

	ctx.StatusCode(http.StatusNoContent)

	return true
}
