package errors

import (
	stdContext "context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/x/pagination"

	"golang.org/x/exp/constraints"
)

// RecoveryHandler is a middleware which recovers from panics and sends an appropriate error response
// to the logger and the client.
func RecoveryHandler(ctx *context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			var err error
			switch v := rec.(type) {
			case error:
				err = v
			case string:
				err = New(v)
			default:
				err = fmt.Errorf("%v", v)
			}

			Internal.LogErr(ctx, err)
			ctx.StopExecution()
		}
	}()

	ctx.Next()
}

// Handle handles a generic response and error from a service call and sends a JSON response to the client.
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

// ContextValidatorFunc is a function which takes a context and a generic type T and returns an error.
// It is used to validate the context before calling a service function.
//
// See Validation package-level function.
type ContextValidatorFunc[T any] func(*context.Context, T) error

const contextValidatorFuncKey = "iris.errors.ContextValidatorFunc"

// Validation adds a context validator function to the context.
// It returns a middleware which can be used to validate the context before calling a service function.
// It panics if the given validators are empty or nil.
//
// Example:
//
// r.Post("/", Validation(validateCreateRequest), createHandler(service))
//
//	func validateCreateRequest(ctx iris.Context, r *CreateRequest) error {
//		return validation.Join(
//			validation.String("fullname", r.Fullname).NotEmpty().Fullname().Length(3, 50),
//			validation.Number("age", r.Age).InRange(18, 130),
//			validation.Slice("hobbies", r.Hobbies).Length(1, 10),
//		)
//	}
func Validation[T any](validators ...ContextValidatorFunc[T]) context.Handler {
	validator := joinContextValidators[T](validators)

	return func(ctx *context.Context) {
		ctx.Values().Set(contextValidatorFuncKey, validator)
		ctx.Next()
	}
}

func joinContextValidators[T any](validators []ContextValidatorFunc[T]) ContextValidatorFunc[T] {
	if len(validators) == 0 || validators[0] == nil {
		panic("at least one validator is required")
	}

	if len(validators) == 1 {
		return validators[0]
	}

	return func(ctx *context.Context, req T) error {
		for _, validator := range validators {
			if validator == nil {
				continue
			}

			if err := validator(ctx, req); err != nil {
				return err
			}
		}

		return nil
	}
}

// ContextValidator is an interface which can be implemented by a request payload struct
// in order to validate the context before calling a service function.
type ContextValidator interface {
	ValidateContext(*context.Context) error
}

func validateContext[T any](ctx *context.Context, req T) bool {
	var err error

	// Always run the request's validator first,
	// so dynamic validators can be customized per path and method.
	if contextValidator, ok := any(&req).(ContextValidator); ok {
		err = contextValidator.ValidateContext(ctx)
	}

	if err == nil {
		if v := ctx.Values().Get(contextValidatorFuncKey); v != nil {
			if contextValidatorFunc, ok := v.(ContextValidatorFunc[T]); ok {
				err = contextValidatorFunc(ctx, req)
			} else if contextValidatorFunc, ok := v.(ContextValidatorFunc[*T]); ok { // or a pointer of T.
				err = contextValidatorFunc(ctx, &req)
			}
		}
	}

	if err != nil {
		if HandleError(ctx, err) {
			return false
		}
	}

	return true
}

func bindResponse[T, R any, F ResponseFunc[T, R]](ctx *context.Context, fn F, fnInput ...T) (R, bool) {
	var req T
	switch len(fnInput) {
	case 0:
		var ok bool
		req, ok = ReadPayload[T](ctx)
		if !ok {
			var resp R
			return resp, false
		}
	case 1:
		req = fnInput[0]
	default:
		panic("invalid number of arguments")
	}

	if !validateContext(ctx, req) {
		var resp R
		return resp, false
	}

	resp, err := fn(ctx, req)
	return resp, !HandleError(ctx, err)
}

// OK handles a generic response and error from a service call and sends a JSON response to the client.
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

// HandlerInputFunc is a function which takes a context and returns a generic type T.
// It is used to call a service function with a generic type T.
// It is used for functions which do not bind a request payload.
// It is used for XHandler functions.
// Developers can design their own HandlerInputFunc functions and use them with the XHandler functions.
// To make a value required, stop the context execution through the context.StopExecution function and fire an error
// or just use one of the [InvalidArgument].X methods.
//
// See PathParam, Query and Value package-level helpers too.
type HandlerInputFunc[T any] interface {
	func(ctx *context.Context) T
}

// GetRequestInputs returns a slice of generic type T from a slice of HandlerInputFunc[T].
// It is exported so end-developers can use it to get the inputs from custom HandlerInputFunc[T] functions.
func GetRequestInputs[T any, I HandlerInputFunc[T]](ctx *context.Context, fnInputFunc []I) ([]T, bool) {
	inputs := make([]T, 0, len(fnInputFunc))
	for _, callIn := range fnInputFunc {
		if callIn == nil {
			continue
		}

		input := callIn(ctx)
		if ctx.IsStopped() { // if the input is required and it's not provided, then the context is stopped.
			return nil, false
		}
		inputs = append(inputs, input)
	}

	return inputs, true
}

// PathParam returns a HandlerInputFunc which reads a path parameter from the context and returns it as a generic type T.
// It is used for XHandler functions.
func PathParam[T any, I HandlerInputFunc[T]](paramName string) I {
	return func(ctx *context.Context) T {
		paramValue := ctx.Params().Store.Get(paramName)
		if paramValue == nil {
			var t T
			return t
		}

		return paramValue.(T)
	}
}

// Value returns a HandlerInputFunc which returns a generic type T.
// It is used for XHandler functions.
func Value[T any, I HandlerInputFunc[T]](value T) I {
	return func(ctx *context.Context) T {
		return value
	}
}

// Query returns a HandlerInputFunc which reads a URL query from the context and returns it as a generic type T.
// It is used for XHandler functions.
func Query[T any, I HandlerInputFunc[T]]() I {
	return func(ctx *context.Context) T {
		value, ok := ReadQuery[T](ctx)
		if !ok {
			var t T
			return t
		}

		return value
	}
}

// Handler handles a generic response and error from a service call and sends a JSON response to the client with status code of 200.
//
// See OK package-level function for more.
func Handler[T, R any, F ResponseFunc[T, R], I HandlerInputFunc[T]](fn F, fnInput ...I) context.Handler {
	return func(ctx *context.Context) {
		inputs, ok := GetRequestInputs(ctx, fnInput)
		if !ok {
			return
		}

		OK(ctx, fn, inputs...)
	}
}

// ListResponseFunc is a function which takes a context,
// a pagination.ListOptions and a generic type T and returns a slice []R, total count of the items and an error.
//
// It's used on the List function.
type ListResponseFunc[T, R any, C constraints.Integer | constraints.Float] interface {
	func(stdContext.Context, pagination.ListOptions, T /* filter options */) ([]R, C, error)
}

// List handles a generic response and error from a service paginated call and sends a JSON response to the client.
// It returns a boolean value indicating whether the handle was successful or not.
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// It reads the pagination.ListOptions from the URL Query and any filter options of generic T from the request body.
// It sets the status code to 200 (OK) and sends a *pagination.List[R] response as a JSON payload.
func List[T, R any, C constraints.Integer | constraints.Float, F ListResponseFunc[T, R, C]](ctx *context.Context, fn F, fnInput ...T) bool {
	listOpts, filter, ok := ReadPaginationOptions[T](ctx)
	if !ok {
		return false
	}

	if !validateContext(ctx, filter) {
		return false
	}

	items, totalCount, err := fn(ctx, listOpts, filter)
	if err != nil {
		HandleError(ctx, err)
		return false
	}

	resp := pagination.NewList(items, int64(totalCount), filter, listOpts)
	return Handle(ctx, resp, nil)
}

// ListHandler handles a generic response and error from a service paginated call and sends a JSON response to the client.
//
// See List package-level function for more.
func ListHandler[T, R any, C constraints.Integer | constraints.Float, F ListResponseFunc[T, R, C], I HandlerInputFunc[T]](fn F, fnInput ...I) context.Handler {
	return func(ctx *context.Context) {
		inputs, ok := GetRequestInputs(ctx, fnInput)
		if !ok {
			return
		}

		List(ctx, fn, inputs...)
	}
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

// CreateHandler handles a create operation and sends a JSON response with the created resource to the client with status code of 201.
//
// See Create package-level function for more.
func CreateHandler[T, R any, F ResponseFunc[T, R], I HandlerInputFunc[T]](fn F, fnInput ...I) context.Handler {
	return func(ctx *context.Context) {
		inputs, ok := GetRequestInputs(ctx, fnInput)
		if !ok {
			return
		}

		Create(ctx, fn, inputs...)
	}
}

// NoContent handles a generic response and error from a service call and sends a JSON response to the client.
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

// NoContentHandler handles a generic response and error from a service call and sends a JSON response to the client with status code of 204.
//
// See NoContent package-level function for more.
func NoContentHandler[T any, F ResponseOnlyErrorFunc[T], I HandlerInputFunc[T]](fn F, fnInput ...I) context.Handler {
	return func(ctx *context.Context) {
		inputs, ok := GetRequestInputs(ctx, fnInput)
		if !ok {
			return
		}

		NoContent(ctx, fn, inputs...)
	}
}

// NoContent handles a generic response and error from a service call and sends a JSON response to the client.
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

// NoContentOrNotModifiedHandler handles a generic response and error from a service call and sends a JSON response to the client with status code of 204 or 304.
//
// See NoContentOrNotModified package-level function for more.
func NoContentOrNotModifiedHandler[T any, F ResponseFunc[T, bool], I HandlerInputFunc[T]](fn F, fnInput ...I) context.Handler {
	return func(ctx *context.Context) {
		inputs, ok := GetRequestInputs(ctx, fnInput)
		if !ok {
			return
		}

		NoContentOrNotModified(ctx, fn, inputs...)
	}
}

// ReadPayload reads a JSON payload from the context and returns it as a generic type T.
// It also returns a boolean value indicating whether the read was successful or not.
// If the read fails, it sends an appropriate error response to the client.
func ReadPayload[T any](ctx *context.Context) (T, bool) {
	var payload T
	err := ctx.ReadJSON(&payload)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			InvalidArgument.Details(ctx, "unable to parse body", "empty body")
			return payload, false
		}

		HandleError(ctx, err)
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
		HandleError(ctx, err)
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
